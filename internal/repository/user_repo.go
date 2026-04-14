package repository

import (
	"context"
	"strings"
	"time"

	"github.com/RenaLio/tudou/internal/models"
	"gorm.io/gorm"
)

type UserListOption struct {
	Page          int
	PageSize      int
	OrderBy       string
	Keyword       string
	Status        *models.UserStatus
	Role          *models.UserRole
	IDs           []int64
	PreloadTokens bool
	PreloadStats  bool
}

type UserRepo interface {
	Create(ctx context.Context, user *models.User) error
	BatchCreate(ctx context.Context, users []*models.User) error
	GetByID(ctx context.Context, id int64) (*models.User, error)
	GetByIDWithRelations(ctx context.Context, id int64) (*models.User, error)
	GetByUsername(ctx context.Context, username string) (*models.User, error)
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	List(ctx context.Context, opt UserListOption) ([]*models.User, int64, error)
	Update(ctx context.Context, user *models.User) error
	RecordLogin(ctx context.Context, id int64, ip string) error
	UpdateStatus(ctx context.Context, id int64, status models.UserStatus) error
	UpdatePassword(ctx context.Context, id int64, hash string) error
	Delete(ctx context.Context, id int64) error
	Exists(ctx context.Context, id int64) (bool, error)
	IsNotFound(err error) bool
}

type userRepo struct {
	*Repository
}

func NewUserRepo(r *Repository) UserRepo {
	return &userRepo{Repository: r}
}

func (r *userRepo) Create(ctx context.Context, user *models.User) error {
	return Create[models.User](ctx, user, r.DB(ctx))
}

func (r *userRepo) BatchCreate(ctx context.Context, users []*models.User) error {
	return BatchCreate[*models.User](ctx, users, r.DB(ctx))
}

func (r *userRepo) GetByID(ctx context.Context, id int64) (*models.User, error) {
	return GetByID[models.User](ctx, id, r.DB(ctx))
}

func (r *userRepo) GetByIDWithRelations(ctx context.Context, id int64) (*models.User, error) {
	user := new(models.User)
	if err := r.DB(ctx).
		Preload("Tokens").
		Preload("Stats").
		Where("id = ?", id).
		First(user).Error; err != nil {
		return nil, err
	}
	return user, nil
}

func (r *userRepo) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	return GetByKey[models.User](ctx, "username", username, r.DB(ctx))
}

func (r *userRepo) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	return GetByKey[models.User](ctx, "email", email, r.DB(ctx))
}

func (r *userRepo) List(ctx context.Context, opt UserListOption) ([]*models.User, int64, error) {
	db := r.DB(ctx).Model(&models.User{})
	if opt.PreloadTokens {
		db = db.Preload("Tokens")
	}
	if opt.PreloadStats {
		db = db.Preload("Stats")
	}

	keyword := strings.TrimSpace(opt.Keyword)
	if keyword != "" {
		like := "%" + keyword + "%"
		db = db.Where("username LIKE ? OR email LIKE ? OR phone LIKE ? OR nickname LIKE ?", like, like, like, like)
	}

	if opt.Status != nil {
		db = db.Where("status = ?", *opt.Status)
	}
	if opt.Role != nil {
		db = db.Where("role = ?", *opt.Role)
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

	data := make([]*models.User, 0, pageSize)
	if err := db.Order(orderBy).Offset(offset).Limit(pageSize).Find(&data).Error; err != nil {
		return nil, 0, err
	}
	return data, total, nil
}

func (r *userRepo) Update(ctx context.Context, user *models.User) error {
	return Update[models.User](ctx, user, user.ID, []string{"ID", "CreatedAt", "DeletedAt", "Tokens", "Stats"}, r.DB(ctx))
}

func (r *userRepo) RecordLogin(ctx context.Context, id int64, ip string) error {
	now := time.Now()
	return r.DB(ctx).Model(&models.User{}).Where("id = ?", id).Updates(map[string]any{
		"last_login_at": now,
		"last_login_ip": ip,
		"login_count":   gorm.Expr("login_count + 1"),
	}).Error
}

func (r *userRepo) UpdateStatus(ctx context.Context, id int64, status models.UserStatus) error {
	return SetField[models.User](ctx, "status", status, id, r.DB(ctx))
}

func (r *userRepo) UpdatePassword(ctx context.Context, id int64, hash string) error {
	return SetField[models.User](ctx, "password", hash, id, r.DB(ctx))
}

func (r *userRepo) Delete(ctx context.Context, id int64) error {
	return Delete[models.User](ctx, id, r.DB(ctx))
}

func (r *userRepo) Exists(ctx context.Context, id int64) (bool, error) {
	return Exists[models.User](ctx, id, r.DB(ctx))
}

func (r *userRepo) IsNotFound(err error) bool {
	return IsNotFound[models.User](err)
}
