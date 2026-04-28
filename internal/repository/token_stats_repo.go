package repository

import (
	"context"
	"errors"

	"github.com/RenaLio/tudou/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type TokenStatsRepo interface {
	Upsert(ctx context.Context, stats *models.TokenStats) error
	GetByTokenID(ctx context.Context, tokenID int64) (*models.TokenStats, error)
	ListAll(ctx context.Context) ([]*models.TokenStats, error)
	ListByTokenIDs(ctx context.Context, tokenIDs []int64) ([]*models.TokenStats, error)
}

type tokenStatsRepo struct {
	*Repository
}

func (r *tokenStatsRepo) ListAll(ctx context.Context) ([]*models.TokenStats, error) {
	items := make([]*models.TokenStats, 0)
	if err := r.DB(ctx).Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func NewTokenStatsRepo(r *Repository) TokenStatsRepo {
	return &tokenStatsRepo{Repository: r}
}

func (r *tokenStatsRepo) Upsert(ctx context.Context, stats *models.TokenStats) error {
	if stats == nil {
		return errors.New("token stats is nil")
	}
	if stats.TokenID <= 0 {
		return errors.New("invalid token id")
	}
	return r.DB(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "token_id"}},
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

func (r *tokenStatsRepo) GetByTokenID(ctx context.Context, tokenID int64) (*models.TokenStats, error) {
	if tokenID <= 0 {
		return nil, errors.New("invalid token id")
	}
	stats := new(models.TokenStats)
	if err := r.DB(ctx).Where("token_id = ?", tokenID).First(stats).Error; err != nil {
		return nil, err
	}
	return stats, nil
}

func (r *tokenStatsRepo) ListByTokenIDs(ctx context.Context, tokenIDs []int64) ([]*models.TokenStats, error) {
	tokenIDs = uniqueInt64(tokenIDs)
	if len(tokenIDs) == 0 {
		return []*models.TokenStats{}, nil
	}
	items := make([]*models.TokenStats, 0, len(tokenIDs))
	if err := r.DB(ctx).Where("token_id IN ?", tokenIDs).Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *tokenStatsRepo) IsNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
