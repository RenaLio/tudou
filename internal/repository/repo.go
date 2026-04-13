package repository

import (
	"context"

	"github.com/RenaLio/tudou/internal/cache"
	"github.com/RenaLio/tudou/internal/pkg/log"
	"gorm.io/gorm"
)

type ContextKeyType struct{}

var ctxTxKey ContextKeyType = ContextKeyType{}

func GetContextKey() ContextKeyType {
	return ctxTxKey
}

type Repository struct {
	db     *gorm.DB
	logger *log.Logger
	cache  *cache.JsonCache
}

func NewRepository(
	logger *log.Logger,
	db *gorm.DB,
	cache *cache.JsonCache,
) *Repository {
	return &Repository{
		db:     db,
		cache:  cache,
		logger: logger,
	}
}

type Transaction interface {
	Transaction(ctx context.Context, fn func(ctx context.Context) error) error
}

func NewTransaction(r *Repository) Transaction {
	return r
}

// DB return new gorm db Session or tx
// If you need to create a Transaction, you must call DB(ctx) and Transaction(ctx,fn)
func (r *Repository) DB(ctx context.Context) *gorm.DB {
	v := ctx.Value(ctxTxKey)
	if v != nil {
		if tx, ok := v.(*gorm.DB); ok {
			return tx
		}
	}
	// r.db.withContext(ctx) will create a new transaction if
	// the context is not a transaction.
	return r.db.WithContext(ctx)
}

func (r *Repository) Transaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return r.DB(ctx).Transaction(func(tx *gorm.DB) error {
		ctx = context.WithValue(ctx, ctxTxKey, tx)
		return fn(ctx)
	})
}
