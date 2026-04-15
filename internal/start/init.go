package start

import (
	"context"
	"errors"

	v1 "github.com/RenaLio/tudou/api/v1"
	"github.com/RenaLio/tudou/internal/models"
	"github.com/RenaLio/tudou/internal/server"
	"github.com/RenaLio/tudou/internal/service"
	"gorm.io/gorm"
)

func InitApp(m *server.Migrate, userService service.UserService) error {
	ctx := context.Background()
	if err := m.Start(ctx); err != nil {
		return err
	}

	const adminUsername = "admin"
	_, err := userService.GetByUsername(ctx, adminUsername)
	if err == nil {
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	role := models.UserRoleAdmin
	status := models.UserStatusEnabled
	_, err = userService.Create(ctx, v1.CreateUserRequest{
		Username: adminUsername,
		Password: "admin",
		Role:     &role,
		Status:   &status,
		Nickname: adminUsername,
	})
	return err
}
