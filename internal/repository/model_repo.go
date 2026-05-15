package repository

import (
	"context"
	"errors"
	"strings"

	"github.com/RenaLio/tudou/internal/models"
	jsoncache "github.com/RenaLio/tudou/pkg/cache"
	"gorm.io/gorm"
)

type AIModelListOption struct {
	Page     int
	PageSize int
	OrderBy  string
	Keyword  string
}

type AIModelRepo interface {
	Create(ctx context.Context, model *models.AIModel) error
	BatchCreate(ctx context.Context, modelsList []*models.AIModel) error
	GetByName(ctx context.Context, name string) (*models.AIModel, error)
	GetExistingNames(ctx context.Context, names []string) ([]string, error)
	List(ctx context.Context, opt AIModelListOption) ([]*models.AIModel, int64, error)
	Update(ctx context.Context, model *models.AIModel) error
	DeleteByName(ctx context.Context, name string) error
	DeleteByNames(ctx context.Context, names []string) (int64, error)
	ExistsByName(ctx context.Context, name string) (bool, error)
}

type aiModelRepo struct {
	*Repository
}

const aiModelCacheKeyPrefix = "repo:ai_model"

func NewAIModelRepo(r *Repository) AIModelRepo {
	return &aiModelRepo{Repository: r}
}

func (r *aiModelRepo) Create(ctx context.Context, model *models.AIModel) error {
	if model != nil {
		r.invalidateModelCache(ctx, model.Name)
	}
	if err := Create[models.AIModel](ctx, model, r.DB(ctx)); err != nil {
		return err
	}
	if model != nil {
		r.invalidateModelCacheOnCommit(ctx, model.Name)
	}
	return nil
}

func (r *aiModelRepo) BatchCreate(ctx context.Context, modelsList []*models.AIModel) error {
	for _, item := range modelsList {
		if item == nil {
			continue
		}
		r.invalidateModelCache(ctx, item.Name)
	}
	if err := BatchCreate[*models.AIModel](ctx, modelsList, r.DB(ctx)); err != nil {
		return err
	}
	for _, item := range modelsList {
		if item == nil {
			continue
		}
		r.invalidateModelCacheOnCommit(ctx, item.Name)
	}
	return nil
}

func (r *aiModelRepo) GetByName(ctx context.Context, name string) (*models.AIModel, error) {
	if r.cacheEnabled(ctx) {
		cached, err := jsoncache.Get[models.AIModel](r.cache, aiModelByNameCacheKey(name))
		if err == nil {
			return &cached, nil
		}
	}

	item, err := GetByKey[models.AIModel](ctx, "name", name, r.DB(ctx))
	if err != nil {
		return nil, err
	}
	r.setModelCache(ctx, item)
	return item, nil
}

func (r *aiModelRepo) GetExistingNames(ctx context.Context, names []string) ([]string, error) {
	if len(names) == 0 {
		return nil, nil
	}
	var existing []string
	if err := r.DB(ctx).Model(&models.AIModel{}).Where("name IN ?", names).Pluck("name", &existing).Error; err != nil {
		return nil, err
	}
	return existing, nil
}

func (r *aiModelRepo) List(ctx context.Context, opt AIModelListOption) ([]*models.AIModel, int64, error) {
	db := r.DB(ctx).Model(&models.AIModel{})

	keyword := strings.TrimSpace(opt.Keyword)
	if keyword != "" {
		like := "%" + keyword + "%"
		db = db.Where("name LIKE ? OR description LIKE ?", like, like)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	_, pageSize, offset := normalizePagination(opt.Page, opt.PageSize)
	orderBy := sanitizeOrderBy(opt.OrderBy, "id DESC")

	data := make([]*models.AIModel, 0, pageSize)
	if err := db.Order(orderBy).Offset(offset).Limit(pageSize).Find(&data).Error; err != nil {
		return nil, 0, err
	}
	return data, total, nil
}

func (r *aiModelRepo) Update(ctx context.Context, model *models.AIModel) error {
	if model == nil {
		return errors.New("model is required")
	}
	name := strings.TrimSpace(model.Name)
	if name == "" {
		return errors.New("name is required")
	}

	r.invalidateModelCache(ctx, name)

	tx := r.DB(ctx).
		Model(&models.AIModel{}).
		Where("name = ?", name).
		Select("*").
		Omit("ID", "Name", "CreatedAt", "UpdatedAt").
		Updates(model)
	if tx.Error != nil {
		return tx.Error
	}
	if tx.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	r.invalidateModelCacheOnCommit(ctx, name)
	return nil
}

func (r *aiModelRepo) DeleteByName(ctx context.Context, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("name is required")
	}
	r.invalidateModelCache(ctx, name)

	tx := r.DB(ctx).Where("name = ?", name).Delete(&models.AIModel{})
	if tx.Error != nil {
		return tx.Error
	}
	if tx.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	r.invalidateModelCacheOnCommit(ctx, name)
	return nil
}

func (r *aiModelRepo) DeleteByNames(ctx context.Context, names []string) (int64, error) {
	if len(names) == 0 {
		return 0, nil
	}

	nameSet := make(map[string]struct{}, len(names))
	normalizedNames := make([]string, 0, len(names))
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if _, ok := nameSet[name]; ok {
			continue
		}
		nameSet[name] = struct{}{}
		normalizedNames = append(normalizedNames, name)
	}
	if len(normalizedNames) == 0 {
		return 0, nil
	}

	r.invalidateModelCache(ctx, normalizedNames...)

	tx := r.DB(ctx).Where("name IN ?", normalizedNames).Delete(&models.AIModel{})
	if tx.Error != nil {
		return 0, tx.Error
	}

	namesCopy := append([]string(nil), normalizedNames...)
	r.onCommitted(ctx, func() {
		r.invalidateModelCache(context.Background(), namesCopy...)
	})
	return tx.RowsAffected, nil
}

func (r *aiModelRepo) ExistsByName(ctx context.Context, name string) (bool, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return false, nil
	}
	count, err := gorm.G[models.AIModel](r.DB(ctx)).Where("name = ?", name).Count(ctx, "id")
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *aiModelRepo) cacheEnabled(ctx context.Context) bool {
	return r != nil && r.cache != nil && ctx != nil && ctx.Value(ctxTxKey) == nil
}

func aiModelByNameCacheKey(name string) string {
	return aiModelCacheKeyPrefix + ":name:" + name
}

func (r *aiModelRepo) setModelCache(ctx context.Context, model *models.AIModel) {
	if !r.cacheEnabled(ctx) || model == nil {
		return
	}
	_ = jsoncache.Set(r.cache, aiModelByNameCacheKey(model.Name), *model)
}

func (r *aiModelRepo) delModelCacheByName(ctx context.Context, name string) {
	if r == nil || r.cache == nil {
		return
	}
	_ = r.cache.Delete(aiModelByNameCacheKey(name))
}

func (r *aiModelRepo) invalidateModelCache(ctx context.Context, names ...string) {
	seen := make(map[string]struct{}, len(names))
	for _, name := range names {
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		r.delModelCacheByName(ctx, name)
	}
}

func (r *aiModelRepo) invalidateModelCacheOnCommit(ctx context.Context, names ...string) {
	namesCopy := append([]string(nil), names...)
	r.onCommitted(ctx, func() {
		// Second delete is delayed to commit when inside a transaction to avoid
		// leaving stale cache after commit.
		// 在事务内将第二次删除延后到提交后执行，避免提交成功后仍残留旧缓存。
		r.invalidateModelCache(context.Background(), namesCopy...)
	})
}
