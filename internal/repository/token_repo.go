package repository

import (
	"context"
	"strings"
	"time"

	"github.com/RenaLio/tudou/internal/models"
	"gorm.io/gorm"
)

type TokenListOption struct {
	Page          int
	PageSize      int
	OrderBy       string
	Keyword       string
	UserID        int64
	GroupID       int64
	Status        *models.TokenStatus
	OnlyAvailable bool
	IDs           []int64
	PreloadUser   bool
	PreloadGroup  bool
	PreloadStats  bool
}

type TokenRepo interface {
	Create(ctx context.Context, token *models.Token) error
	BatchCreate(ctx context.Context, tokens []*models.Token) error
	GetByID(ctx context.Context, id int64) (*models.Token, error)
	GetByIDWithRelations(ctx context.Context, id int64) (*models.Token, error)
	GetByToken(ctx context.Context, tokenValue string) (*models.Token, error)
	GetByTokenWithRelations(ctx context.Context, tokenValue string) (*models.Token, error)
	GetAvailableByToken(ctx context.Context, tokenValue string) (*models.Token, error)
	List(ctx context.Context, opt TokenListOption) ([]*models.Token, int64, error)
	Update(ctx context.Context, token *models.Token) error
	UpdateStatus(ctx context.Context, id int64, status models.TokenStatus) error
	Delete(ctx context.Context, id int64) error
	Exists(ctx context.Context, id int64) (bool, error)
}

type tokenRepo struct {
	*Repository
}

func NewTokenRepo(r *Repository) TokenRepo {
	return &tokenRepo{Repository: r}
}

func (r *tokenRepo) Create(ctx context.Context, token *models.Token) error {
	return Create[models.Token](ctx, token, r.DB(ctx))
}

func (r *tokenRepo) BatchCreate(ctx context.Context, tokens []*models.Token) error {
	return BatchCreate[*models.Token](ctx, tokens, r.DB(ctx))
}

func (r *tokenRepo) GetByID(ctx context.Context, id int64) (*models.Token, error) {
	return GetByID[models.Token](ctx, id, r.DB(ctx))
}

func (r *tokenRepo) GetByIDWithRelations(ctx context.Context, id int64) (*models.Token, error) {
	token := new(models.Token)
	if err := r.DB(ctx).
		Preload("User").
		Preload("Group").
		Preload("Stats").
		Where("id = ?", id).
		First(token).Error; err != nil {
		return nil, err
	}
	return token, nil
}

func (r *tokenRepo) GetByToken(ctx context.Context, tokenValue string) (*models.Token, error) {
	return GetByKey[models.Token](ctx, "token", tokenValue, r.DB(ctx))
}

func (r *tokenRepo) GetByTokenWithRelations(ctx context.Context, tokenValue string) (*models.Token, error) {
	token := new(models.Token)
	if err := r.DB(ctx).
		Preload("User").
		Preload("Group").
		Preload("Stats").
		Where("token = ?", tokenValue).
		First(token).Error; err != nil {
		return nil, err
	}
	return token, nil
}

func (r *tokenRepo) GetAvailableByToken(ctx context.Context, tokenValue string) (*models.Token, error) {
	token, err := r.GetByTokenWithRelations(ctx, tokenValue)
	if err != nil {
		return nil, err
	}
	if !token.IsAvailable() {
		return nil, gorm.ErrRecordNotFound
	}
	return token, nil
}

func (r *tokenRepo) List(ctx context.Context, opt TokenListOption) ([]*models.Token, int64, error) {
	db := r.DB(ctx).Model(&models.Token{})
	if opt.PreloadUser {
		db = db.Preload("User")
	}
	if opt.PreloadGroup {
		db = db.Preload("Group")
	}
	if opt.PreloadStats {
		db = db.Preload("Stats")
	}

	keyword := strings.TrimSpace(opt.Keyword)
	if keyword != "" {
		like := "%" + keyword + "%"
		db = db.Where("name LIKE ? OR token LIKE ?", like, like)
	}

	if opt.UserID > 0 {
		db = db.Where("user_id = ?", opt.UserID)
	}
	if opt.GroupID > 0 {
		db = db.Where("group_id = ?", opt.GroupID)
	}
	if opt.Status != nil {
		db = db.Where("status = ?", *opt.Status)
	}

	if opt.OnlyAvailable {
		now := time.Now()
		db = db.Where("status = ?", models.TokenStatusEnabled).
			Where("expires_at IS NULL OR expires_at > ?", now)
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

	data := make([]*models.Token, 0, pageSize)
	if err := db.Order(orderBy).Offset(offset).Limit(pageSize).Find(&data).Error; err != nil {
		return nil, 0, err
	}
	return data, total, nil
}

func (r *tokenRepo) Update(ctx context.Context, token *models.Token) error {
	return Update[models.Token](ctx, token, token.ID, []string{"ID", "CreatedAt", "DeletedAt", "User", "Group", "Stats"}, r.DB(ctx))
}

func (r *tokenRepo) UpdateStatus(ctx context.Context, id int64, status models.TokenStatus) error {
	return SetField[models.Token](ctx, "status", status, id, r.DB(ctx))
}

func (r *tokenRepo) Delete(ctx context.Context, id int64) error {
	return Delete[models.Token](ctx, id, r.DB(ctx))
}

func (r *tokenRepo) Exists(ctx context.Context, id int64) (bool, error) {
	return Exists[models.Token](ctx, id, r.DB(ctx))
}

func (r *tokenRepo) IsNotFound(err error) bool {
	return IsNotFound[models.Token](err)
}
