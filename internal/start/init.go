package start

import (
	"context"
	"errors"

	v1 "github.com/RenaLio/tudou/api/v1"
	"github.com/RenaLio/tudou/internal/loadbalancer"
	"github.com/RenaLio/tudou/internal/models"
	"github.com/RenaLio/tudou/internal/server"
	"github.com/RenaLio/tudou/internal/service"
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

func InitLBRegistry(db *gorm.DB) (loadbalancer.LoadBalancer, loadbalancer.MetricsCollector) {
	registry := loadbalancer.NewRegistry()
	// load 一些数据

	//
	tempCollector := loadbalancer.NewAsyncMetricsCollector(registry, 1024)
	lb := loadbalancer.NewDynamicLoadBalancer(registry)
	return lb, tempCollector
}
