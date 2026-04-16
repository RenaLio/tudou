package server

import (
	"context"
	"fmt"

	"github.com/RenaLio/tudou/internal/models"
	"github.com/RenaLio/tudou/internal/pkg/log"
	"gorm.io/gorm"
)

type Migrate struct {
	db     *gorm.DB
	logger *log.Logger
}

func NewMigrate(db *gorm.DB, logger *log.Logger) *Migrate {
	return &Migrate{
		db:     db,
		logger: logger,
	}
}

func (m *Migrate) Start(ctx context.Context) error {
	if err := m.db.AutoMigrate(
		&models.AIModel{},
		&models.Channel{},
		&models.ChannelGroup{},
		&models.GroupChannel{},
		&models.ChannelStats{},
		&models.ChannelModelStats{},
		&models.Token{},
		&models.TokenStats{},
		&models.User{},
		&models.UserStats{},
		&models.SystemConfig{},
		&models.RequestLog{},
		&models.UserUsageDailyStats{},
		&models.UserUsageHourlyStats{},
	); err != nil {
		m.logger.Error(fmt.Sprintf("AutoMigrate error: %v", err))
		return err
	}
	return nil
}

func (m *Migrate) Stop(ctx context.Context) error {
	return nil
}
