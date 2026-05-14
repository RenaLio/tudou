package start

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/RenaLio/tudou/internal/loadbalancer"
	"github.com/RenaLio/tudou/internal/models"
	"github.com/RenaLio/tudou/internal/pkg/sid"
	"github.com/RenaLio/tudou/internal/repository"
	"github.com/RenaLio/tudou/internal/server"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const (
	lbRegistryWarmupWindow = 12 * time.Hour
	lbRegistryWarmupLimit  = 256
)

func InitApp(m *server.Migrate, userRepo repository.UserRepo, channelGroupRepo repository.ChannelGroupRepo, s *sid.Sid) error {
	ctx := context.Background()
	if err := m.Start(ctx); err != nil {
		return err
	}

	// 初始化默认用户
	const adminUsername = "admin"
	_, err := userRepo.GetByUsername(ctx, adminUsername)
	if err == nil {
		// admin 用户已存在，跳过创建
	} else if errors.Is(err, gorm.ErrRecordNotFound) {
		hash, hashErr := bcrypt.GenerateFromPassword([]byte("admin"), bcrypt.DefaultCost)
		if hashErr != nil {
			return hashErr
		}
		user := &models.User{
			ID:       s.GenInt64(),
			Username: adminUsername,
			Password: string(hash),
			Role:     models.UserRoleAdmin,
			Status:   models.UserStatusEnabled,
			Nickname: adminUsername,
		}
		if createErr := userRepo.Create(ctx, user); createErr != nil {
			return createErr
		}
	} else {
		return err
	}

	// 初始化默认分组
	const defaultGroupName = "default"
	_, err = channelGroupRepo.GetByName(ctx, defaultGroupName)
	if err == nil {
		// default 分组已存在，跳过创建
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	group := &models.ChannelGroup{
		ID:                  s.GenInt64(),
		Name:                defaultGroupName,
		NameRemark:          "默认分组",
		LoadBalanceStrategy: models.LoadBalanceStrategyWeighted,
	}
	return channelGroupRepo.Create(ctx, group)
}

func InitLBRegistry(db *gorm.DB, groupRepo repository.ChannelGroupRepo) *loadbalancer.Registry {
	registry := loadbalancer.NewRegistry()
	ctx := context.Background()
	// On a fresh database the tables may not exist yet (migration runs later);
	// return an empty registry so the app can finish migration and start.
	if !hasTable(db, ctx, "channels") || !hasTable(db, ctx, "channel_groups") || !hasTable(db, ctx, "request_logs") {
		return registry
	}

	var channels []*models.Channel
	if err := db.WithContext(ctx).Find(&channels).Error; err != nil {
		panic(err)
	}
	for _, ch := range channels {
		registry.ReloadChannel(ch)
	}

	groups, err := groupRepo.PreLoadRegistryData(ctx)
	if err != nil {
		panic(err)
	}
	for _, g := range groups {
		registry.ReloadGroup(g)
	}

	warmupSince := time.Now().Add(-lbRegistryWarmupWindow)
	recentLogs := make([]*models.RequestLog, 0, lbRegistryWarmupLimit)
	if err := db.WithContext(ctx).
		Model(&models.RequestLog{}).
		Where("created_at >= ?", warmupSince).
		Order("created_at DESC, id DESC").
		Limit(lbRegistryWarmupLimit).
		Find(&recentLogs).Error; err != nil {
		panic(err)
	}
	// Reverse DESC query result to replay metrics in chronological order.
	for i, j := 0, len(recentLogs)-1; i < j; i, j = i+1, j-1 {
		recentLogs[i], recentLogs[j] = recentLogs[j], recentLogs[i]
	}
	replayRequestLogsToRegistry(registry, recentLogs)

	return registry
}

// hasTable checks whether a table exists in the database. Used to guard
// InitLBRegistry queries on a fresh database before migration runs.
func hasTable(db *gorm.DB, ctx context.Context, table string) bool {
	return db.WithContext(ctx).Migrator().HasTable(table)
}

func replayRequestLogsToRegistry(registry *loadbalancer.Registry, logs []*models.RequestLog) {
	if registry == nil || len(logs) == 0 {
		return
	}
	logs = flattenLogsWithRetryTrace(logs)
	if len(logs) == 0 {
		return
	}

	for _, logItem := range logs {
		if logItem == nil || logItem.ChannelID <= 0 || strings.TrimSpace(logItem.Model) == "" {
			continue
		}
		statusCode := 0
		if code, err := strconv.Atoi(strings.TrimSpace(logItem.ErrorCode)); err == nil {
			statusCode = code
		}
		// Runtime collector ignores input validation failures.
		if statusCode == 400 {
			continue
		}

		endpoint := registry.GetEndpoint(logItem.Model, logItem.ChannelID)
		if endpoint == nil {
			continue
		}
		isSuccess := logItem.Status == models.RequestStatusSuccess
		tps := 0.0
		if isSuccess {
			duration := logItem.TransferTime
			if duration <= 0 {
				duration = 1
			}
			tps = float64(logItem.OutputToken) * 1000 / float64(duration)
		}
		endpoint.UpdateMetrics(isSuccess, logItem.IsStream, float64(logItem.TTFT), tps)
		if channel := registry.GetChannelById(logItem.ChannelID); channel != nil {
			channel.UpdateSuccessRate(isSuccess)
		}
	}
}

func flattenLogsWithRetryTrace(logs []*models.RequestLog) []*models.RequestLog {
	if len(logs) == 0 {
		return nil
	}
	out := make([]*models.RequestLog, 0, len(logs))
	for _, logItem := range logs {
		if logItem == nil {
			continue
		}
		out = append(out, logItem)
		if len(logItem.Extra.RetryTrace) == 0 {
			continue
		}
		for _, retry := range logItem.Extra.RetryTrace {
			temp := *logItem
			cloneLog := &temp

			cloneLog.ChannelID = retry.ChannelID
			cloneLog.ChannelName = retry.ChannelName
			cloneLog.UpstreamModel = retry.UpstreamModel
			cloneLog.Status = models.RequestStatusFail
			cloneLog.ErrorCode = strconv.Itoa(retry.StatusCode)
			cloneLog.ErrorMsg = retry.StatusBody

			// Retry attempts are failed probes; no billable usage/latency contribution.
			cloneLog.InputToken = 0
			cloneLog.OutputToken = 0
			cloneLog.CachedCreationInputTokens = 0
			cloneLog.CachedReadInputTokens = 0
			cloneLog.CostMicros = 0
			cloneLog.TransferTime = 0
			cloneLog.TTFT = 0

			out = append(out, cloneLog)
		}
	}
	return out
}
