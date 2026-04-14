package repository

import (
	"context"
	"strings"

	"github.com/RenaLio/tudou/internal/models"
)

type AIModelListOption struct {
	Page      int
	PageSize  int
	OrderBy   string
	Keyword   string
	Type      models.ModelType
	IsEnabled *bool
	IDs       []int64
}

type AIModelRepo interface {
	Create(ctx context.Context, model *models.AIModel) error
	BatchCreate(ctx context.Context, modelsList []*models.AIModel) error
	GetByID(ctx context.Context, id int64) (*models.AIModel, error)
	GetByName(ctx context.Context, name string) (*models.AIModel, error)
	List(ctx context.Context, opt AIModelListOption) ([]*models.AIModel, int64, error)
	Update(ctx context.Context, model *models.AIModel) error
	SetEnabled(ctx context.Context, id int64, enabled bool) error
	Delete(ctx context.Context, id int64) error
	Exists(ctx context.Context, id int64) (bool, error)
}

type aiModelRepo struct {
	*Repository
}

func NewAIModelRepo(r *Repository) AIModelRepo {
	return &aiModelRepo{Repository: r}
}

func (r *aiModelRepo) Create(ctx context.Context, model *models.AIModel) error {
	return Create[models.AIModel](ctx, model, r.DB(ctx))
}

func (r *aiModelRepo) BatchCreate(ctx context.Context, modelsList []*models.AIModel) error {
	return BatchCreate[*models.AIModel](ctx, modelsList, r.DB(ctx))
}

func (r *aiModelRepo) GetByID(ctx context.Context, id int64) (*models.AIModel, error) {
	return GetByID[models.AIModel](ctx, id, r.DB(ctx))

}

func (r *aiModelRepo) GetByName(ctx context.Context, name string) (*models.AIModel, error) {
	return GetByKey[models.AIModel](ctx, "name", name, r.DB(ctx))
}

func (r *aiModelRepo) List(ctx context.Context, opt AIModelListOption) ([]*models.AIModel, int64, error) {
	db := r.DB(ctx).Model(&models.AIModel{})

	keyword := strings.TrimSpace(opt.Keyword)
	if keyword != "" {
		like := "%" + keyword + "%"
		db = db.Where("name LIKE ? OR description LIKE ?", like, like)
	}

	if opt.Type != "" {
		db = db.Where("type = ?", opt.Type)
	}
	if opt.IsEnabled != nil {
		db = db.Where("is_enabled = ?", *opt.IsEnabled)
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
	return Update[models.AIModel](ctx, model, model.ID, []string{"ID", "CreatedAt", "UpdatedAt"}, r.DB(ctx))
}

func (r *aiModelRepo) SetEnabled(ctx context.Context, id int64, enabled bool) error {
	return SetField[models.AIModel](ctx, "is_enabled", enabled, id, r.DB(ctx))
}

func (r *aiModelRepo) Delete(ctx context.Context, id int64) error {
	return Delete[models.AIModel](ctx, id, r.DB(ctx))
}

func (r *aiModelRepo) Exists(ctx context.Context, id int64) (bool, error) {
	return Exists[models.AIModel](ctx, id, r.DB(ctx))
}
