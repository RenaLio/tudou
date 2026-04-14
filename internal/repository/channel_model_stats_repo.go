package repository

import (
	"context"
	"errors"
	"strings"

	"github.com/RenaLio/tudou/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ChannelModelStatsRepo interface {
	Upsert(ctx context.Context, stats *models.ChannelModelStats) error
	GetByChannelModel(ctx context.Context, channelID int64, model string) (*models.ChannelModelStats, error)
	ListByChannelID(ctx context.Context, channelID int64) ([]*models.ChannelModelStats, error)
}

type channelModelStatsRepo struct {
	*Repository
}

func NewChannelModelStatsRepo(r *Repository) ChannelModelStatsRepo {
	return &channelModelStatsRepo{Repository: r}
}

func (r *channelModelStatsRepo) Upsert(ctx context.Context, stats *models.ChannelModelStats) error {
	if stats == nil {
		return errors.New("channel model stats is nil")
	}
	stats.Model = strings.TrimSpace(stats.Model)
	if stats.ChannelID <= 0 || stats.Model == "" {
		return errors.New("invalid channel id or model")
	}
	return r.DB(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "channel_id"},
			{Name: "model"},
		},
		DoUpdates: clause.AssignmentColumns([]string{
			"input_token",
			"output_token",
			"cached_creation_input_tokens",
			"cached_read_input_tokens",
			"request_success",
			"request_failed",
			"total_cost_micros",
			"avg_ttft",
			"avg_tps",
		}),
	}).Create(stats).Error
}

func (r *channelModelStatsRepo) GetByChannelModel(ctx context.Context, channelID int64, model string) (*models.ChannelModelStats, error) {
	model = strings.TrimSpace(model)
	if channelID <= 0 || model == "" {
		return nil, errors.New("invalid channel id or model")
	}
	stats := new(models.ChannelModelStats)
	if err := r.DB(ctx).Where("channel_id = ? AND model = ?", channelID, model).First(stats).Error; err != nil {
		return nil, err
	}
	return stats, nil
}

func (r *channelModelStatsRepo) ListByChannelID(ctx context.Context, channelID int64) ([]*models.ChannelModelStats, error) {
	if channelID <= 0 {
		return nil, errors.New("invalid channel id")
	}
	items := make([]*models.ChannelModelStats, 0, 8)
	if err := r.DB(ctx).Where("channel_id = ?", channelID).Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *channelModelStatsRepo) IsNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
