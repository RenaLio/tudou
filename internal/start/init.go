package start

import (
	"context"
	"errors"
	"fmt"

	v1 "github.com/RenaLio/tudou/api/v1"
	"github.com/RenaLio/tudou/internal/loadbalancer"
	"github.com/RenaLio/tudou/internal/models"
	"github.com/RenaLio/tudou/internal/repository"
	"github.com/RenaLio/tudou/internal/server"
	"github.com/RenaLio/tudou/internal/service"
	"github.com/goccy/go-json"
	"gorm.io/gorm"
)

func InitApp(m *server.Migrate, userService service.UserService, channelGroupService service.ChannelGroupService) error {
	ctx := context.Background()
	if err := m.Start(ctx); err != nil {
		return err
	}

	// 初始化默认用户
	const adminUsername = "admin"
	_, err := userService.GetByUsername(ctx, adminUsername)
	if err == nil {
		// admin 用户已存在，跳过创建
	} else if errors.Is(err, gorm.ErrRecordNotFound) {
		role := models.UserRoleAdmin
		status := models.UserStatusEnabled
		_, err = userService.Create(ctx, v1.CreateUserRequest{
			Username: adminUsername,
			Password: "admin",
			Role:     &role,
			Status:   &status,
			Nickname: adminUsername,
		})
		if err != nil {
			return err
		}
	} else {
		return err
	}

	// 初始化默认分组
	const defaultGroupName = "default"
	_, err = channelGroupService.GetByName(ctx, defaultGroupName)
	if err == nil {
		// default 分组已存在，跳过创建
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	strategy := models.LoadBalanceStrategyWeighted
	_, err = channelGroupService.Create(ctx, v1.CreateChannelGroupRequest{
		Name:                defaultGroupName,
		NameRemark:          "默认分组",
		LoadBalanceStrategy: &strategy,
	})
	return err
}

func InitLBRegistry(db *gorm.DB, groupRepo repository.ChannelGroupRepo) *loadbalancer.Registry {
	registry := loadbalancer.NewRegistry()
	ctx := context.Background()
	// loading channels and groups
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

	data, _ := json.Marshal(registry)
	fmt.Println(string(data))

	return registry
}
