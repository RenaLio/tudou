package repository

import (
	"context"

	"github.com/RenaLio/tudou/internal/pkg/log"
	"github.com/RenaLio/tudou/internal/pkg/sid"
	"github.com/RenaLio/tudou/pkg/cache"
	"gorm.io/gorm"
)

type ContextKeyType struct{}

var ctxTxKey ContextKeyType = ContextKeyType{}
var ctxAfterCommitKey afterCommitContextKeyType = afterCommitContextKeyType{}

type afterCommitContextKeyType struct{}

type afterCommitCallbacks struct {
	list []func()
}

func GetContextKey() ContextKeyType {
	return ctxTxKey
}

type Repository struct {
	db     *gorm.DB
	logger *log.Logger
	cache  *cache.JsonCache
	sid    *sid.Sid
}

func NewRepository(
	logger *log.Logger,
	db *gorm.DB,
	cache *cache.JsonCache,
	sid *sid.Sid,
) *Repository {
	return &Repository{
		db:     db,
		cache:  cache,
		logger: logger,
		sid:    sid,
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
	outerCallbacks, _ := ctx.Value(ctxAfterCommitKey).(*afterCommitCallbacks)
	if outerCallbacks != nil {
		// Nested transaction: reuse outer callback bucket so invalidation runs once
		// after the outermost commit succeeds.
		// 嵌套事务复用外层回调容器，确保缓存失效只在最外层事务成功提交后执行一次。
		return r.DB(ctx).Transaction(func(tx *gorm.DB) error {
			txCtx := context.WithValue(ctx, ctxTxKey, tx)
			txCtx = context.WithValue(txCtx, ctxAfterCommitKey, outerCallbacks)
			return fn(txCtx)
		})
	}

	callbacks := &afterCommitCallbacks{}
	err := r.DB(ctx).Transaction(func(tx *gorm.DB) error {
		txCtx := context.WithValue(ctx, ctxTxKey, tx)
		txCtx = context.WithValue(txCtx, ctxAfterCommitKey, callbacks)
		return fn(txCtx)
	})
	if err != nil {
		return err
	}

	for _, callback := range callbacks.list {
		if callback != nil {
			callback()
		}
	}
	return nil
}

func (r *Repository) Log(ctx context.Context) *log.Logger {
	return r.logger.FromContext(ctx)
}

func (r *Repository) NextID() int64 {
	return r.sid.GenInt64()
}

func (r *Repository) onCommitted(ctx context.Context, callback func()) {
	if callback == nil {
		return
	}
	if ctx == nil {
		callback()
		return
	}
	callbacks, _ := ctx.Value(ctxAfterCommitKey).(*afterCommitCallbacks)
	if callbacks == nil {
		// Non-transactional path: execute immediately.
		// 非事务路径下没有提交阶段，直接立即执行。
		callback()
		return
	}
	callbacks.list = append(callbacks.list, callback)
}
