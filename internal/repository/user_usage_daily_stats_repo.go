package repository

import (
	"context"
	"errors"
	"strings"

	"github.com/RenaLio/tudou/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UserUsageDailyStatsListOption struct {
	Page     int
	PageSize int
	OrderBy  string
	UserID   int64
	DateFrom string
	DateTo   string
}

type UserUsageDailyStatsRepo interface {
	Upsert(ctx context.Context, stats *models.UserUsageDailyStats) error
	GetByUserDate(ctx context.Context, userID int64, date string) (*models.UserUsageDailyStats, error)
	List(ctx context.Context, opt UserUsageDailyStatsListOption) ([]*models.UserUsageDailyStats, int64, error)
}

type userUsageDailyStatsRepo struct {
	*Repository
}

func NewUserUsageDailyStatsRepo(r *Repository) UserUsageDailyStatsRepo {
	return &userUsageDailyStatsRepo{Repository: r}
}

func (r *userUsageDailyStatsRepo) Upsert(ctx context.Context, stats *models.UserUsageDailyStats) error {
	if stats == nil {
		return errors.New("user usage daily stats is nil")
	}
	stats.Date = strings.TrimSpace(stats.Date)
	if stats.UserID <= 0 || stats.Date == "" {
		return errors.New("invalid user id or date")
	}
	return r.DB(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "user_id"},
			{Name: "date"},
		},
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

func (r *userUsageDailyStatsRepo) GetByUserDate(ctx context.Context, userID int64, date string) (*models.UserUsageDailyStats, error) {
	date = strings.TrimSpace(date)
	if userID <= 0 || date == "" {
		return nil, errors.New("invalid user id or date")
	}
	stats := new(models.UserUsageDailyStats)
	if err := r.DB(ctx).Where("user_id = ? AND date = ?", userID, date).First(stats).Error; err != nil {
		return nil, err
	}
	return stats, nil
}

func (r *userUsageDailyStatsRepo) List(ctx context.Context, opt UserUsageDailyStatsListOption) ([]*models.UserUsageDailyStats, int64, error) {
	db := r.DB(ctx).Model(&models.UserUsageDailyStats{})
	if opt.UserID > 0 {
		db = db.Where("user_id = ?", opt.UserID)
	}
	if strings.TrimSpace(opt.DateFrom) != "" {
		db = db.Where("date >= ?", strings.TrimSpace(opt.DateFrom))
	}
	if strings.TrimSpace(opt.DateTo) != "" {
		db = db.Where("date <= ?", strings.TrimSpace(opt.DateTo))
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	_, pageSize, offset := normalizePagination(opt.Page, opt.PageSize)
	orderBy := sanitizeOrderBy(opt.OrderBy, "date DESC, user_id ASC")

	items := make([]*models.UserUsageDailyStats, 0, pageSize)
	if err := db.Order(orderBy).Offset(offset).Limit(pageSize).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *userUsageDailyStatsRepo) IsNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
