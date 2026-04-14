package repository

import (
	"context"
	"strings"

	"github.com/RenaLio/tudou/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SystemConfigListOption struct {
	Page        int
	PageSize    int
	OrderBy     string
	Keyword     string
	Scope       *models.ConfigScope
	OnlyVisible bool
}

type SystemConfigRepo interface {
	Create(ctx context.Context, config *models.SystemConfig) error
	Upsert(ctx context.Context, config *models.SystemConfig) error
	GetByID(ctx context.Context, id int64) (*models.SystemConfig, error)
	GetByKey(ctx context.Context, key string) (*models.SystemConfig, error)
	List(ctx context.Context, opt SystemConfigListOption) ([]*models.SystemConfig, int64, error)
	SetValueByKey(ctx context.Context, key string, val any) error
	DeleteByKey(ctx context.Context, key string) error
}

type systemConfigRepo struct {
	*Repository
}

func NewSystemConfigRepo(r *Repository) SystemConfigRepo {
	return &systemConfigRepo{Repository: r}
}

func (r *systemConfigRepo) Create(ctx context.Context, config *models.SystemConfig) error {
	return Create[models.SystemConfig](ctx, config, r.DB(ctx))
}

func (r *systemConfigRepo) Upsert(ctx context.Context, config *models.SystemConfig) error {
	return r.DB(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "key"}},
		DoUpdates: clause.Assignments(map[string]any{
			"value":       config.Value,
			"type":        config.Type,
			"scope":       config.Scope,
			"description": config.Description,
			"is_editable": config.IsEditable,
			"is_visible":  config.IsVisible,
			"sort":        config.Sort,
			"updated_at":  gorm.Expr("CURRENT_TIMESTAMP"),
		}),
	}).Create(config).Error
}

func (r *systemConfigRepo) GetByID(ctx context.Context, id int64) (*models.SystemConfig, error) {
	return GetByID[models.SystemConfig](ctx, id, r.DB(ctx))
}

func (r *systemConfigRepo) GetByKey(ctx context.Context, key string) (*models.SystemConfig, error) {
	return GetByKey[models.SystemConfig](ctx, "key", key, r.DB(ctx))
}

func (r *systemConfigRepo) List(ctx context.Context, opt SystemConfigListOption) ([]*models.SystemConfig, int64, error) {
	db := r.DB(ctx).Model(&models.SystemConfig{})

	keyword := strings.TrimSpace(opt.Keyword)
	if keyword != "" {
		like := "%" + keyword + "%"
		db = db.Where("key LIKE ? OR description LIKE ?", like, like)
	}
	if opt.Scope != nil {
		db = db.Where("scope = ?", *opt.Scope)
	}
	if opt.OnlyVisible {
		db = db.Where("is_visible = ?", true)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	_, pageSize, offset := normalizePagination(opt.Page, opt.PageSize)
	orderBy := sanitizeOrderBy(opt.OrderBy, "sort ASC, id ASC")

	data := make([]*models.SystemConfig, 0, pageSize)
	if err := db.Order(orderBy).Offset(offset).Limit(pageSize).Find(&data).Error; err != nil {
		return nil, 0, err
	}
	return data, total, nil
}

func (r *systemConfigRepo) SetValueByKey(ctx context.Context, key string, val any) error {
	return r.DB(ctx).
		Model(&models.SystemConfig{}).
		Where("key = ?", key).
		Update("value", models.ConfigValue{ValueData: val}).Error
}

func (r *systemConfigRepo) DeleteByKey(ctx context.Context, key string) error {
	return r.DB(ctx).Where("key = ?", key).Delete(&models.SystemConfig{}).Error
}

func (r *systemConfigRepo) IsNotFound(err error) bool {
	return IsNotFound[models.SystemConfig](err)
}
