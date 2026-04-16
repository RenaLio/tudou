package repository

import (
	"context"
	"strings"

	"github.com/RenaLio/tudou/internal/models"
	"gorm.io/gorm/clause"
)

type UserUsageHourlyStatsListOption struct {
	Page     int
	PageSize int
	OrderBy  string
	UserID   int64
	DateFrom string
	HourFrom int
	DateTo   string
	HourTo   int
}

type UserUsageHourlyStatsRepo interface {
	Upsert(ctx context.Context, stats *models.UserUsageHourlyStats) error
	GetByUserDateHour(ctx context.Context, userID int64, date string, hour int) (*models.UserUsageHourlyStats, error)
	List(ctx context.Context, opt UserUsageHourlyStatsListOption) ([]*models.UserUsageHourlyStats, int64, error)
}

type userUsageHourlyStatsRepo struct {
	*Repository
}

func NewUserUsageHourlyStatsRepo(r *Repository) UserUsageHourlyStatsRepo {
	return &userUsageHourlyStatsRepo{Repository: r}
}

func (r *userUsageHourlyStatsRepo) Upsert(ctx context.Context, stats *models.UserUsageHourlyStats) error {
	stats.Date = strings.TrimSpace(stats.Date)
	return r.DB(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "user_id"},
			{Name: "date"},
			{Name: "hour"},
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

func (r *userUsageHourlyStatsRepo) GetByUserDateHour(ctx context.Context, userID int64, date string, hour int) (*models.UserUsageHourlyStats, error) {
	stats := new(models.UserUsageHourlyStats)
	if err := r.DB(ctx).Where("user_id = ? AND date = ? AND hour = ?", userID, strings.TrimSpace(date), hour).First(stats).Error; err != nil {
		return nil, err
	}
	return stats, nil
}

func (r *userUsageHourlyStatsRepo) List(ctx context.Context, opt UserUsageHourlyStatsListOption) ([]*models.UserUsageHourlyStats, int64, error) {
	db := r.DB(ctx).Model(&models.UserUsageHourlyStats{})
	if opt.UserID > 0 {
		db = db.Where("user_id = ?", opt.UserID)
	}
	// 开始时间：日期+小时
	if strings.TrimSpace(opt.DateFrom) != "" {
		if opt.HourFrom >= 0 && opt.HourFrom <= 23 {
			db = db.Where("(date > ? OR (date = ? AND hour >= ?))", strings.TrimSpace(opt.DateFrom), strings.TrimSpace(opt.DateFrom), opt.HourFrom)
		} else {
			db = db.Where("date >= ?", strings.TrimSpace(opt.DateFrom))
		}
	}
	// 截止时间：日期+小时
	if strings.TrimSpace(opt.DateTo) != "" {
		if opt.HourTo >= 0 && opt.HourTo <= 23 {
			db = db.Where("(date < ? OR (date = ? AND hour <= ?))", strings.TrimSpace(opt.DateTo), strings.TrimSpace(opt.DateTo), opt.HourTo)
		} else {
			db = db.Where("date <= ?", strings.TrimSpace(opt.DateTo))
		}
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	_, pageSize, offset := normalizePagination(opt.Page, opt.PageSize)
	orderBy := sanitizeOrderBy(opt.OrderBy, "date DESC, hour ASC")

	items := make([]*models.UserUsageHourlyStats, 0, pageSize)
	if err := db.Order(orderBy).Offset(offset).Limit(pageSize).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}
