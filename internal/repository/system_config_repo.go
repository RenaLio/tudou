package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/RenaLio/tudou/internal/models"
	jsoncache "github.com/RenaLio/tudou/pkg/cache"
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

const systemConfigCacheKeyPrefix = "repo:system_config"

func NewSystemConfigRepo(r *Repository) SystemConfigRepo {
	return &systemConfigRepo{Repository: r}
}

func (r *systemConfigRepo) Create(ctx context.Context, config *models.SystemConfig) error {
	if err := Create[models.SystemConfig](ctx, config, r.DB(ctx)); err != nil {
		return err
	}
	if r.cacheEnabled(ctx) {
		r.setConfigCache(ctx, config)
	} else {
		r.invalidateConfigCacheOnCommit(ctx, config.ID, config.Key)
	}
	return nil
}

func (r *systemConfigRepo) Upsert(ctx context.Context, config *models.SystemConfig) error {
	// Double-delete: pre-invalidate before DB write
	r.invalidateConfigCache(ctx, config.ID, config.Key)
	err := r.DB(ctx).Clauses(clause.OnConflict{
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
	if err != nil {
		return err
	}
	r.invalidateConfigCacheOnCommit(ctx, config.ID, config.Key)
	return nil
}

func (r *systemConfigRepo) GetByID(ctx context.Context, id int64) (*models.SystemConfig, error) {
	if r.cacheEnabled(ctx) {
		cached, err := jsoncache.Get[models.SystemConfig](r.cache, systemConfigByIDCacheKey(id))
		if err == nil {
			return &cached, nil
		}
	}
	item, err := GetByID[models.SystemConfig](ctx, id, r.DB(ctx))
	if r.cacheEnabled(ctx) && item != nil {
		_ = jsoncache.Set(r.cache, systemConfigByIDCacheKey(item.ID), *item)
		_ = jsoncache.Set(r.cache, systemConfigByKeyCacheKey(item.Key), *item)
	}
	return item, err
}

func (r *systemConfigRepo) GetByKey(ctx context.Context, key string) (*models.SystemConfig, error) {
	if r.cacheEnabled(ctx) {
		cached, err := jsoncache.Get[models.SystemConfig](r.cache, systemConfigByKeyCacheKey(key))
		if err == nil {
			return &cached, nil
		}
	}
	item, err := GetByKey[models.SystemConfig](ctx, "key", key, r.DB(ctx))
	if r.cacheEnabled(ctx) && item != nil {
		_ = jsoncache.Set(r.cache, systemConfigByKeyCacheKey(item.Key), *item)
		_ = jsoncache.Set(r.cache, systemConfigByIDCacheKey(item.ID), *item)
	}
	return item, err
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
	// Double-delete: pre-invalidate before DB write
	cachedID := r.cachedConfigIDByKey(ctx, key)
	r.invalidateConfigCache(ctx, cachedID, key)
	if err := r.DB(ctx).
		Model(&models.SystemConfig{}).
		Where("key = ?", key).
		Update("value", models.ConfigValue{ValueData: val}).Error; err != nil {
		return err
	}
	r.invalidateConfigCache(ctx, cachedID, key)
	r.invalidateConfigCacheOnCommit(ctx, cachedID, key)
	return nil
}

func (r *systemConfigRepo) DeleteByKey(ctx context.Context, key string) error {
	// Double-delete: pre-invalidate before DB write
	cachedID := r.cachedConfigIDByKey(ctx, key)
	r.invalidateConfigCache(ctx, cachedID, key)
	if err := r.DB(ctx).Where("key = ?", key).Delete(&models.SystemConfig{}).Error; err != nil {
		return err
	}
	r.invalidateConfigCache(ctx, cachedID, key)
	r.invalidateConfigCacheOnCommit(ctx, cachedID, key)
	return nil
}

func (r *systemConfigRepo) IsNotFound(err error) bool {
	return IsNotFound[models.SystemConfig](err)
}

// --- cache helpers ---

func (r *systemConfigRepo) cacheEnabled(ctx context.Context) bool {
	return r != nil && r.cache != nil && ctx != nil && ctx.Value(ctxTxKey) == nil
}

func systemConfigByIDCacheKey(id int64) string {
	return systemConfigCacheKeyPrefix + ":id:" + fmt.Sprintf("%d", id)
}

func systemConfigByKeyCacheKey(key string) string {
	return systemConfigCacheKeyPrefix + ":key:" + strings.TrimSpace(key)
}

func (r *systemConfigRepo) setConfigCache(ctx context.Context, cfg *models.SystemConfig) {
	if !r.cacheEnabled(ctx) || cfg == nil {
		return
	}
	_ = jsoncache.Set(r.cache, systemConfigByIDCacheKey(cfg.ID), *cfg)
	_ = jsoncache.Set(r.cache, systemConfigByKeyCacheKey(cfg.Key), *cfg)
}

func (r *systemConfigRepo) delConfigCacheByID(id int64) {
	if r == nil || r.cache == nil {
		return
	}
	_ = r.cache.Delete(systemConfigByIDCacheKey(id))
}

func (r *systemConfigRepo) delConfigCacheByKey(key string) {
	if r == nil || r.cache == nil {
		return
	}
	_ = r.cache.Delete(systemConfigByKeyCacheKey(key))
}

func (r *systemConfigRepo) invalidateConfigCache(ctx context.Context, id int64, key string) {
	r.delConfigCacheByID(id)
	r.delConfigCacheByKey(key)
}

func (r *systemConfigRepo) invalidateConfigCacheOnCommit(ctx context.Context, id int64, key string) {
	if r == nil || r.cache == nil {
		return
	}
	keyCopy := strings.TrimSpace(key)
	r.onCommitted(ctx, func() {
		r.invalidateConfigCache(context.Background(), id, keyCopy)
	})
}

func (r *systemConfigRepo) cachedConfigIDByKey(ctx context.Context, key string) int64 {
	if r == nil || r.cache == nil {
		return 0
	}
	cached, err := jsoncache.Get[models.SystemConfig](r.cache, systemConfigByKeyCacheKey(key))
	if err != nil {
		return 0
	}
	return cached.ID
}
