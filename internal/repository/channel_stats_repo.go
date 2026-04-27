package repository

import (
	"context"
	"errors"
	"time"

	"github.com/RenaLio/tudou/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ChannelStatsRepo interface {
	Upsert(ctx context.Context, stats *models.ChannelStats) error
	GetByChannelID(ctx context.Context, channelID int64) (*models.ChannelStats, error)
	ListByChannelIDs(ctx context.Context, channelIDs []int64) ([]*models.ChannelStats, error)
	ListRequestLogsByChannelIDsAndRange(ctx context.Context, channelIDs []int64, start, end time.Time) ([]*models.RequestLog, error)
}

type channelStatsRepo struct {
	*Repository
}

func NewChannelStatsRepo(r *Repository) ChannelStatsRepo {
	return &channelStatsRepo{Repository: r}
}

func (r *channelStatsRepo) Upsert(ctx context.Context, stats *models.ChannelStats) error {
	if stats == nil {
		return errors.New("channel stats is nil")
	}
	if stats.ChannelID <= 0 {
		return errors.New("invalid channel id")
	}
	return r.DB(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "channel_id"}},
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
			"window_3h",
		}),
	}).Create(stats).Error
}

func (r *channelStatsRepo) GetByChannelID(ctx context.Context, channelID int64) (*models.ChannelStats, error) {
	if channelID <= 0 {
		return nil, errors.New("invalid channel id")
	}
	stats := new(models.ChannelStats)
	if err := r.DB(ctx).Where("channel_id = ?", channelID).First(stats).Error; err != nil {
		return nil, err
	}
	return stats, nil
}

func (r *channelStatsRepo) ListByChannelIDs(ctx context.Context, channelIDs []int64) ([]*models.ChannelStats, error) {
	channelIDs = uniqueInt64(channelIDs)
	if len(channelIDs) == 0 {
		return []*models.ChannelStats{}, nil
	}
	items := make([]*models.ChannelStats, 0, len(channelIDs))
	if err := r.DB(ctx).Where("channel_id IN ?", channelIDs).Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *channelStatsRepo) ListRequestLogsByChannelIDsAndRange(ctx context.Context, channelIDs []int64, start, end time.Time) ([]*models.RequestLog, error) {
	channelIDs = uniqueInt64(channelIDs)
	if len(channelIDs) == 0 {
		return []*models.RequestLog{}, nil
	}
	if !start.Before(end) {
		return nil, errors.New("invalid range: start must be before end")
	}
	items := make([]*models.RequestLog, 0, 256)
	err := r.DB(ctx).
		Model(&models.RequestLog{}).
		Where("channel_id IN ?", channelIDs).
		Where("created_at >= ? AND created_at < ?", start, end).
		Order("created_at ASC, id ASC").
		Find(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (r *channelStatsRepo) IsNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
