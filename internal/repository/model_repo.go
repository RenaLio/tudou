package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/RenaLio/tudou/internal/models"
	jsoncache "github.com/RenaLio/tudou/pkg/cache"
)

type AIModelListOption struct {
	Page     int
	PageSize int
	OrderBy  string
	Keyword  string
	IDs      []int64
}

type AIModelRepo interface {
	Create(ctx context.Context, model *models.AIModel) error
	BatchCreate(ctx context.Context, modelsList []*models.AIModel) error
	GetByID(ctx context.Context, id int64) (*models.AIModel, error)
	GetByName(ctx context.Context, name string) (*models.AIModel, error)
	GetExistingNames(ctx context.Context, names []string) ([]string, error)
	List(ctx context.Context, opt AIModelListOption) ([]*models.AIModel, int64, error)
	Update(ctx context.Context, model *models.AIModel) error
	SetEnabled(ctx context.Context, id int64, enabled bool) error
	Delete(ctx context.Context, id int64) error
	Exists(ctx context.Context, id int64) (bool, error)
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
		r.invalidateModelCache(ctx, model.ID, model.Name)
	}
	if err := Create[models.AIModel](ctx, model, r.DB(ctx)); err != nil {
		return err
	}
	if model != nil {
		r.invalidateModelCacheOnCommit(ctx, model.ID, model.Name)
	}
	return nil
}

func (r *aiModelRepo) BatchCreate(ctx context.Context, modelsList []*models.AIModel) error {
	for _, item := range modelsList {
		if item == nil {
			continue
		}
		r.invalidateModelCache(ctx, item.ID, item.Name)
	}
	if err := BatchCreate[*models.AIModel](ctx, modelsList, r.DB(ctx)); err != nil {
		return err
	}
	for _, item := range modelsList {
		if item == nil {
			continue
		}
		r.invalidateModelCacheOnCommit(ctx, item.ID, item.Name)
	}
	return nil
}

func (r *aiModelRepo) GetByID(ctx context.Context, id int64) (*models.AIModel, error) {
	if r.cacheEnabled(ctx) {
		cached, err := jsoncache.Get[models.AIModel](r.cache, aiModelByIDCacheKey(id))
		if err == nil {
			return &cached, nil
		}
	}

	item, err := GetByID[models.AIModel](ctx, id, r.DB(ctx))
	if err != nil {
		return nil, err
	}
	r.setModelCache(ctx, item)
	return item, nil

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

	if len(opt.IDs) > 0 {
		db = db.Where("id IN ?", uniqueInt64(opt.IDs))
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
	oldName := r.cachedModelNameByID(ctx, model.ID)
	r.invalidateModelCache(ctx, model.ID, model.Name, oldName)

	if err := Update[models.AIModel](ctx, model, model.ID, []string{"ID", "CreatedAt", "UpdatedAt"}, r.DB(ctx)); err != nil {
		return err
	}

	r.invalidateModelCacheOnCommit(ctx, model.ID, model.Name, oldName)
	return nil
}

func (r *aiModelRepo) SetEnabled(ctx context.Context, id int64, enabled bool) error {
	oldName := r.cachedModelNameByID(ctx, id)
	r.invalidateModelCache(ctx, id, oldName)

	if err := SetField[models.AIModel](ctx, "is_enabled", enabled, id, r.DB(ctx)); err != nil {
		return err
	}
	r.invalidateModelCacheOnCommit(ctx, id, oldName)
	return nil
}

func (r *aiModelRepo) Delete(ctx context.Context, id int64) error {
	oldName := r.cachedModelNameByID(ctx, id)
	r.invalidateModelCache(ctx, id, oldName)
	if err := Delete[models.AIModel](ctx, id, r.DB(ctx)); err != nil {
		return err
	}
	r.invalidateModelCacheOnCommit(ctx, id, oldName)
	return nil
}

func (r *aiModelRepo) Exists(ctx context.Context, id int64) (bool, error) {
	return Exists[models.AIModel](ctx, id, r.DB(ctx))
}

func (r *aiModelRepo) cacheEnabled(ctx context.Context) bool {
	return r != nil && r.cache != nil && ctx != nil && ctx.Value(ctxTxKey) == nil
}

func aiModelByIDCacheKey(id int64) string {
	return fmt.Sprintf("%s:id:%d", aiModelCacheKeyPrefix, id)
}

func aiModelByNameCacheKey(name string) string {
	return fmt.Sprintf("%s:name:%s", aiModelCacheKeyPrefix, name)
}

func (r *aiModelRepo) setModelCache(ctx context.Context, model *models.AIModel) {
	if !r.cacheEnabled(ctx) || model == nil {
		return
	}
	_ = jsoncache.Set(r.cache, aiModelByIDCacheKey(model.ID), *model)
	_ = jsoncache.Set(r.cache, aiModelByNameCacheKey(model.Name), *model)
}

func (r *aiModelRepo) delModelCacheByID(ctx context.Context, id int64) {
	if r == nil || r.cache == nil {
		return
	}
	_ = r.cache.Delete(aiModelByIDCacheKey(id))
}

func (r *aiModelRepo) delModelCacheByName(ctx context.Context, name string) {
	if r == nil || r.cache == nil {
		return
	}
	_ = r.cache.Delete(aiModelByNameCacheKey(name))
}

func (r *aiModelRepo) cachedModelNameByID(ctx context.Context, id int64) string {
	if r == nil || r.cache == nil {
		return ""
	}
	cached, err := jsoncache.Get[models.AIModel](r.cache, aiModelByIDCacheKey(id))
	if err != nil {
		return ""
	}
	return cached.Name
}

func (r *aiModelRepo) invalidateModelCache(ctx context.Context, id int64, names ...string) {
	r.delModelCacheByID(ctx, id)
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

func (r *aiModelRepo) invalidateModelCacheOnCommit(ctx context.Context, id int64, names ...string) {
	namesCopy := append([]string(nil), names...)
	r.onCommitted(ctx, func() {
		// Second delete is delayed to commit when inside a transaction to avoid
		// leaving stale cache after commit.
		// 在事务内将第二次删除延后到提交后执行，避免提交成功后仍残留旧缓存。
		r.invalidateModelCache(context.Background(), id, namesCopy...)
	})
}
