package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/RenaLio/tudou/internal/models"
	jsoncache "github.com/RenaLio/tudou/pkg/cache"
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

const tokenCacheKeyPrefix = "repo:token"

func NewTokenRepo(r *Repository) TokenRepo {
	return &tokenRepo{Repository: r}
}

func (r *tokenRepo) Create(ctx context.Context, token *models.Token) error {
	if token != nil {
		r.invalidateTokenCache(ctx, token.ID, token.Token)
	}
	if err := Create[models.Token](ctx, token, r.DB(ctx)); err != nil {
		return err
	}
	if token != nil {
		r.invalidateTokenCacheOnCommit(ctx, token.ID, token.Token)
	}
	return nil
}

func (r *tokenRepo) BatchCreate(ctx context.Context, tokens []*models.Token) error {
	for _, tk := range tokens {
		if tk == nil {
			continue
		}
		r.invalidateTokenCache(ctx, tk.ID, tk.Token)
	}
	if err := BatchCreate[*models.Token](ctx, tokens, r.DB(ctx)); err != nil {
		return err
	}
	for _, tk := range tokens {
		if tk == nil {
			continue
		}
		r.invalidateTokenCacheOnCommit(ctx, tk.ID, tk.Token)
	}
	return nil
}

func (r *tokenRepo) GetByID(ctx context.Context, id int64) (*models.Token, error) {
	if r.cacheEnabled(ctx) {
		cached, err := jsoncache.Get[models.Token](r.cache, tokenByIDCacheKey(id))
		if err == nil {
			return &cached, nil
		}
	}
	token, err := GetByID[models.Token](ctx, id, r.DB(ctx))
	if err != nil {
		return nil, err
	}
	if r.cacheEnabled(ctx) && token != nil {
		_ = jsoncache.Set(r.cache, tokenByIDCacheKey(token.ID), *token)
		if strings.TrimSpace(token.Token) != "" {
			_ = jsoncache.Set(r.cache, tokenByValueCacheKey(token.Token), *token)
		}
	}
	return token, nil
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
	if r.cacheEnabled(ctx) {
		cached, err := jsoncache.Get[models.Token](r.cache, tokenByValueCacheKey(tokenValue))
		if err == nil {
			return &cached, nil
		}
	}
	token, err := GetByKey[models.Token](ctx, "token", tokenValue, r.DB(ctx))
	if err != nil {
		return nil, err
	}
	if r.cacheEnabled(ctx) && token != nil {
		_ = jsoncache.Set(r.cache, tokenByValueCacheKey(tokenValue), *token)
		_ = jsoncache.Set(r.cache, tokenByIDCacheKey(token.ID), *token)
	}
	return token, nil
}

func (r *tokenRepo) GetByTokenWithRelations(ctx context.Context, tokenValue string) (*models.Token, error) {
	if r.cacheEnabled(ctx) {
		cached, err := jsoncache.Get[models.Token](r.cache, tokenByValueRelationsCacheKey(tokenValue))
		if err == nil {
			return &cached, nil
		}
	}
	token := new(models.Token)
	if err := r.DB(ctx).
		Preload("User").
		Preload("Group").
		Preload("Stats").
		Where("token = ?", tokenValue).
		First(token).Error; err != nil {
		return nil, err
	}
	if r.cacheEnabled(ctx) {
		_ = jsoncache.Set(r.cache, tokenByValueRelationsCacheKey(tokenValue), *token)
		_ = jsoncache.Set(r.cache, tokenByIDCacheKey(token.ID), *token)
		_ = jsoncache.Set(r.cache, tokenByValueCacheKey(token.Token), *token)
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
	if token != nil {
		r.invalidateTokenCache(ctx, token.ID, token.Token)
	}
	if err := Update[models.Token](ctx, token, token.ID, []string{"ID", "CreatedAt", "DeletedAt", "User", "Group", "Stats"}, r.DB(ctx)); err != nil {
		return err
	}
	if token != nil {
		r.invalidateTokenCacheOnCommit(ctx, token.ID, token.Token)
	}
	return nil
}

func (r *tokenRepo) UpdateStatus(ctx context.Context, id int64, status models.TokenStatus) error {
	// UpdateStatus does not load entity from DB after write; invalidate by cached
	// token value (if present) plus id keys.
	// UpdateStatus 写后不再回查 DB，按缓存里的 token 值（若存在）+ id 键做失效。
	tokenValue := r.cachedTokenValueByID(ctx, id)
	r.invalidateTokenCache(ctx, id, tokenValue)

	if err := SetField[models.Token](ctx, "status", status, id, r.DB(ctx)); err != nil {
		return err
	}
	r.invalidateTokenCacheOnCommit(ctx, id, tokenValue)
	return nil
}

func (r *tokenRepo) Delete(ctx context.Context, id int64) error {
	tokenValue := r.cachedTokenValueByID(ctx, id)
	r.invalidateTokenCache(ctx, id, tokenValue)
	if err := Delete[models.Token](ctx, id, r.DB(ctx)); err != nil {
		return err
	}
	r.invalidateTokenCacheOnCommit(ctx, id, tokenValue)
	return nil
}

func (r *tokenRepo) Exists(ctx context.Context, id int64) (bool, error) {
	return Exists[models.Token](ctx, id, r.DB(ctx))
}

func (r *tokenRepo) IsNotFound(err error) bool {
	return IsNotFound[models.Token](err)
}

func (r *tokenRepo) cacheEnabled(ctx context.Context) bool {
	return r != nil && r.cache != nil && ctx != nil && ctx.Value(ctxTxKey) == nil
}

func tokenByIDCacheKey(id int64) string {
	return fmt.Sprintf("%s:id:%d", tokenCacheKeyPrefix, id)
}

func tokenByValueCacheKey(tokenValue string) string {
	return fmt.Sprintf("%s:value:%s", tokenCacheKeyPrefix, strings.TrimSpace(tokenValue))
}

func tokenByValueRelationsCacheKey(tokenValue string) string {
	return fmt.Sprintf("%s:value_rel:%s", tokenCacheKeyPrefix, strings.TrimSpace(tokenValue))
}

func (r *tokenRepo) delTokenCacheByID(ctx context.Context, id int64) {
	if r == nil || r.cache == nil {
		return
	}
	_ = r.cache.Delete(tokenByIDCacheKey(id))
}

func (r *tokenRepo) delTokenCacheByValue(ctx context.Context, tokenValue string) {
	if r == nil || r.cache == nil {
		return
	}
	_ = r.cache.Delete(tokenByValueCacheKey(tokenValue))
}

func (r *tokenRepo) delTokenRelationsCacheByValue(ctx context.Context, tokenValue string) {
	if r == nil || r.cache == nil {
		return
	}
	_ = r.cache.Delete(tokenByValueRelationsCacheKey(tokenValue))
}

func (r *tokenRepo) invalidateTokenCache(ctx context.Context, id int64, tokenValue string) {
	r.delTokenCacheByID(ctx, id)
	if strings.TrimSpace(tokenValue) == "" {
		return
	}
	r.delTokenCacheByValue(ctx, tokenValue)
	r.delTokenRelationsCacheByValue(ctx, tokenValue)
}

func (r *tokenRepo) invalidateTokenCacheOnCommit(ctx context.Context, id int64, tokenValue string) {
	tokenValueCopy := tokenValue
	r.onCommitted(ctx, func() {
		// Keep same "double delete" semantics as other write paths while ensuring
		// transactional updates clear cache only after commit.
		// 与其他写路径保持“双删”语义，同时保证事务内更新在提交后再清缓存。
		r.invalidateTokenCache(context.Background(), id, tokenValueCopy)
	})
}

func (r *tokenRepo) cachedTokenValueByID(ctx context.Context, id int64) string {
	if r == nil || r.cache == nil {
		return ""
	}
	cached, err := jsoncache.Get[models.Token](r.cache, tokenByIDCacheKey(id))
	if err != nil {
		return ""
	}
	return cached.Token
}
