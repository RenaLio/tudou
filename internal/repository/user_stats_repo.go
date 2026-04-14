package repository

import (
	"context"
	"errors"

	"github.com/RenaLio/tudou/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UserStatsRepo interface {
	Upsert(ctx context.Context, stats *models.UserStats) error
	GetByUserID(ctx context.Context, userID int64) (*models.UserStats, error)
	ListByUserIDs(ctx context.Context, userIDs []int64) ([]*models.UserStats, error)
}

type userStatsRepo struct {
	*Repository
}

func NewUserStatsRepo(r *Repository) UserStatsRepo {
	return &userStatsRepo{Repository: r}
}

func (r *userStatsRepo) Upsert(ctx context.Context, stats *models.UserStats) error {
	if stats == nil {
		return errors.New("user stats is nil")
	}
	if stats.UserID <= 0 {
		return errors.New("invalid user id")
	}
	return r.DB(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"input_token",
			"output_token",
			"cached_creation_input_tokens",
			"cached_read_input_tokens",
			"request_success",
			"request_failed",
			"total_cost_micros",
		}),
	}).Create(stats).Error
}

func (r *userStatsRepo) GetByUserID(ctx context.Context, userID int64) (*models.UserStats, error) {
	if userID <= 0 {
		return nil, errors.New("invalid user id")
	}
	stats := new(models.UserStats)
	if err := r.DB(ctx).Where("user_id = ?", userID).First(stats).Error; err != nil {
		return nil, err
	}
	return stats, nil
}

func (r *userStatsRepo) ListByUserIDs(ctx context.Context, userIDs []int64) ([]*models.UserStats, error) {
	userIDs = uniqueInt64(userIDs)
	if len(userIDs) == 0 {
		return []*models.UserStats{}, nil
	}
	items := make([]*models.UserStats, 0, len(userIDs))
	if err := r.DB(ctx).Where("user_id IN ?", userIDs).Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *userStatsRepo) IsNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
