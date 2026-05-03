package repository

import (
	"context"
	"errors"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/RenaLio/tudou/internal/models"
	jsoncache "github.com/RenaLio/tudou/pkg/cache"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type setEnabledTestQueryErrKey struct{}
type modelQueryCountEnabledKey struct{}

func TestAIModelRepo_GetByName_WhitespaceBehavesLikeDB(t *testing.T) {
	repo, _ := newAIModelRepoForTest(t)
	ctx := context.Background()

	model := &models.AIModel{
		ID:        1010,
		Name:      "Model-WS",
		Type:      models.ModelTypeChat,
		IsEnabled: true,
	}
	if err := repo.Create(ctx, model); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if _, err := repo.GetByName(ctx, model.Name); err != nil {
		t.Fatalf("GetByName (prime cache) failed: %v", err)
	}

	_, err := repo.GetByName(ctx, " "+model.Name+" ")
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected gorm.ErrRecordNotFound for whitespace-wrapped name, got %v", err)
	}
}

func TestAIModelByNameCacheKey_PreservesCase(t *testing.T) {
	upper := aiModelByNameCacheKey("GPT-4o")
	lower := aiModelByNameCacheKey("gpt-4o")
	if upper == lower {
		t.Fatalf("expected different cache keys for different name cases, got same: %q", upper)
	}
}

func TestAIModelRepo_SetEnabled_DoesNotReadBackAndStillInvalidatesNameCache(t *testing.T) {
	repo, db := newAIModelRepoForTest(t)
	ctx := context.Background()

	model := &models.AIModel{
		ID:        1001,
		Name:      "Model-A",
		Type:      models.ModelTypeChat,
		IsEnabled: true,
	}
	if err := repo.Create(ctx, model); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Prime cache by name with old enabled state.
	cached, err := repo.GetByName(ctx, model.Name)
	if err != nil {
		t.Fatalf("GetByName (prime cache) failed: %v", err)
	}
	if !cached.IsEnabled {
		t.Fatalf("expected seeded model to be enabled")
	}

	const callbackName = "test:force_query_error_for_set_enabled"
	queryErr := errors.New("forced query error")
	if err := db.Callback().Query().Before("gorm:query").Register(callbackName, func(tx *gorm.DB) {
		if tx == nil || tx.Statement == nil || tx.Statement.Context == nil {
			return
		}
		if forced, _ := tx.Statement.Context.Value(setEnabledTestQueryErrKey{}).(bool); forced {
			tx.AddError(queryErr)
		}
	}); err != nil {
		t.Fatalf("register query callback failed: %v", err)
	}
	defer func() {
		_ = db.Callback().Query().Remove(callbackName)
	}()

	failCtx := context.WithValue(ctx, setEnabledTestQueryErrKey{}, true)
	err = repo.SetEnabled(failCtx, model.ID, false)
	if err != nil {
		t.Fatalf("expected SetEnabled to avoid readback query path, got err: %v", err)
	}

	// If SetEnabled can't refresh cache, stale name cache must be invalidated.
	after, err := repo.GetByName(ctx, model.Name)
	if err != nil {
		t.Fatalf("GetByName after SetEnabled failed: %v", err)
	}
	if after.IsEnabled {
		t.Fatalf("expected fresh DB value is_enabled=false, got true (stale cache)")
	}
}

func TestAIModelRepo_SetEnabled_NoReadbackQuery(t *testing.T) {
	repo, db := newAIModelRepoForTest(t)
	ctx := context.Background()

	model := &models.AIModel{
		ID:        1003,
		Name:      "Model-C",
		Type:      models.ModelTypeChat,
		IsEnabled: true,
	}
	if err := repo.Create(ctx, model); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if _, err := repo.GetByName(ctx, model.Name); err != nil {
		t.Fatalf("GetByName (prime cache) failed: %v", err)
	}

	const callbackName = "test:count_queries_for_set_enabled_failure"
	queryErr := errors.New("forced query error")
	var queryCount int64
	if err := db.Callback().Query().Before("gorm:query").Register(callbackName, func(tx *gorm.DB) {
		atomic.AddInt64(&queryCount, 1)
		if tx == nil || tx.Statement == nil || tx.Statement.Context == nil {
			return
		}
		if forced, _ := tx.Statement.Context.Value(setEnabledTestQueryErrKey{}).(bool); forced {
			tx.AddError(queryErr)
		}
	}); err != nil {
		t.Fatalf("register query callback failed: %v", err)
	}
	defer func() {
		_ = db.Callback().Query().Remove(callbackName)
	}()

	start := atomic.LoadInt64(&queryCount)
	failCtx := context.WithValue(ctx, setEnabledTestQueryErrKey{}, true)
	err := repo.SetEnabled(failCtx, model.ID, false)
	if err != nil {
		t.Fatalf("expected SetEnabled to avoid readback query path, got err: %v", err)
	}
	used := atomic.LoadInt64(&queryCount) - start
	if used != 0 {
		t.Fatalf("expected zero readback queries during SetEnabled, got %d", used)
	}
}

func TestAIModelRepo_SetEnabled_SuccessRefreshesNameCache(t *testing.T) {
	repo, _ := newAIModelRepoForTest(t)
	ctx := context.Background()

	model := &models.AIModel{
		ID:        1002,
		Name:      "Model-B",
		Type:      models.ModelTypeChat,
		IsEnabled: true,
	}
	if err := repo.Create(ctx, model); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if _, err := repo.GetByName(ctx, model.Name); err != nil {
		t.Fatalf("GetByName (prime cache) failed: %v", err)
	}

	if err := repo.SetEnabled(ctx, model.ID, false); err != nil {
		t.Fatalf("SetEnabled failed: %v", err)
	}

	after, err := repo.GetByName(ctx, model.Name)
	if err != nil {
		t.Fatalf("GetByName after SetEnabled failed: %v", err)
	}
	if after.IsEnabled {
		t.Fatalf("expected refreshed cache is_enabled=false, got true")
	}
}

func TestAIModelRepo_SetEnabled_SuccessInvalidatesCache_NoImmediateRefill(t *testing.T) {
	repo, db := newAIModelRepoForTest(t)
	ctx := context.Background()

	model := &models.AIModel{
		ID:        1004,
		Name:      "Model-D",
		Type:      models.ModelTypeChat,
		IsEnabled: true,
	}
	if err := repo.Create(ctx, model); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if _, err := repo.GetByID(ctx, model.ID); err != nil {
		t.Fatalf("GetByID (prime cache) failed: %v", err)
	}

	const callbackName = "test:count_queries_after_set_enabled_success"
	var queryCount int64
	if err := db.Callback().Query().Before("gorm:query").Register(callbackName, func(tx *gorm.DB) {
		if tx == nil || tx.Statement == nil || tx.Statement.Context == nil {
			return
		}
		if enabled, _ := tx.Statement.Context.Value(modelQueryCountEnabledKey{}).(bool); enabled {
			atomic.AddInt64(&queryCount, 1)
		}
	}); err != nil {
		t.Fatalf("register query callback failed: %v", err)
	}
	defer func() {
		_ = db.Callback().Query().Remove(callbackName)
	}()

	if err := repo.SetEnabled(ctx, model.ID, false); err != nil {
		t.Fatalf("SetEnabled failed: %v", err)
	}

	start := atomic.LoadInt64(&queryCount)
	probeCtx := context.WithValue(ctx, modelQueryCountEnabledKey{}, true)
	after, err := repo.GetByID(probeCtx, model.ID)
	if err != nil {
		t.Fatalf("GetByID after SetEnabled failed: %v", err)
	}
	used := atomic.LoadInt64(&queryCount) - start
	if used != 1 {
		t.Fatalf("expected first read after SetEnabled to hit DB once due to invalidation, got %d", used)
	}
	if after.IsEnabled {
		t.Fatalf("expected disabled model after SetEnabled, got enabled")
	}
}

func TestAIModelRepo_Update_InvalidatesCache_NoImmediateRefill(t *testing.T) {
	repo, db := newAIModelRepoForTest(t)
	ctx := context.Background()

	model := &models.AIModel{
		ID:          1005,
		Name:        "Model-E",
		Type:        models.ModelTypeChat,
		IsEnabled:   true,
		Description: "old",
	}
	if err := repo.Create(ctx, model); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if _, err := repo.GetByName(ctx, model.Name); err != nil {
		t.Fatalf("GetByName (prime cache) failed: %v", err)
	}

	model.Description = "new"
	if err := repo.Update(ctx, model); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	const callbackName = "test:count_queries_after_update_success"
	var queryCount int64
	if err := db.Callback().Query().Before("gorm:query").Register(callbackName, func(tx *gorm.DB) {
		if tx == nil || tx.Statement == nil || tx.Statement.Context == nil {
			return
		}
		if enabled, _ := tx.Statement.Context.Value(modelQueryCountEnabledKey{}).(bool); enabled {
			atomic.AddInt64(&queryCount, 1)
		}
	}); err != nil {
		t.Fatalf("register query callback failed: %v", err)
	}
	defer func() {
		_ = db.Callback().Query().Remove(callbackName)
	}()

	start := atomic.LoadInt64(&queryCount)
	probeCtx := context.WithValue(ctx, modelQueryCountEnabledKey{}, true)
	after, err := repo.GetByName(probeCtx, model.Name)
	if err != nil {
		t.Fatalf("GetByName after Update failed: %v", err)
	}
	used := atomic.LoadInt64(&queryCount) - start
	if used != 1 {
		t.Fatalf("expected first read after Update to hit DB once due to invalidation, got %d", used)
	}
	if after.Description != "new" {
		t.Fatalf("expected updated description, got %q", after.Description)
	}
}

func TestAIModelRepo_Update_DeletesCacheBeforeDBWrite(t *testing.T) {
	repo, db := newAIModelRepoForTest(t)
	ctx := context.Background()

	model := &models.AIModel{
		ID:          1006,
		Name:        "Model-F",
		Type:        models.ModelTypeChat,
		IsEnabled:   true,
		Description: "old",
	}
	if err := repo.Create(ctx, model); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if _, err := repo.GetByID(ctx, model.ID); err != nil {
		t.Fatalf("GetByID (prime id cache) failed: %v", err)
	}
	if _, err := repo.GetByName(ctx, model.Name); err != nil {
		t.Fatalf("GetByName (prime name cache) failed: %v", err)
	}

	const callbackName = "test:check_cache_deleted_before_update"
	var staleHit int64
	if err := db.Callback().Update().Before("gorm:update").Register(callbackName, func(tx *gorm.DB) {
		if repo.cache == nil {
			return
		}
		if _, err := repo.cache.Get(aiModelByIDCacheKey(model.ID)); err == nil {
			atomic.AddInt64(&staleHit, 1)
		}
		if _, err := repo.cache.Get(aiModelByNameCacheKey(model.Name)); err == nil {
			atomic.AddInt64(&staleHit, 1)
		}
	}); err != nil {
		t.Fatalf("register update callback failed: %v", err)
	}
	defer func() {
		_ = db.Callback().Update().Remove(callbackName)
	}()

	model.Description = "new"
	if err := repo.Update(ctx, model); err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if got := atomic.LoadInt64(&staleHit); got != 0 {
		t.Fatalf("expected cache already deleted before DB update, stale hits=%d", got)
	}
}

func TestAIModelRepo_SetEnabled_DeletesCacheBeforeDBWrite(t *testing.T) {
	repo, db := newAIModelRepoForTest(t)
	ctx := context.Background()

	model := &models.AIModel{
		ID:        1007,
		Name:      "Model-G",
		Type:      models.ModelTypeChat,
		IsEnabled: true,
	}
	if err := repo.Create(ctx, model); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if _, err := repo.GetByID(ctx, model.ID); err != nil {
		t.Fatalf("GetByID (prime id cache) failed: %v", err)
	}
	if _, err := repo.GetByName(ctx, model.Name); err != nil {
		t.Fatalf("GetByName (prime name cache) failed: %v", err)
	}

	const callbackName = "test:check_cache_deleted_before_set_enabled"
	var staleHit int64
	if err := db.Callback().Update().Before("gorm:update").Register(callbackName, func(tx *gorm.DB) {
		if repo.cache == nil {
			return
		}
		if _, err := repo.cache.Get(aiModelByIDCacheKey(model.ID)); err == nil {
			atomic.AddInt64(&staleHit, 1)
		}
		if _, err := repo.cache.Get(aiModelByNameCacheKey(model.Name)); err == nil {
			atomic.AddInt64(&staleHit, 1)
		}
	}); err != nil {
		t.Fatalf("register update callback failed: %v", err)
	}
	defer func() {
		_ = db.Callback().Update().Remove(callbackName)
	}()

	if err := repo.SetEnabled(ctx, model.ID, false); err != nil {
		t.Fatalf("SetEnabled failed: %v", err)
	}
	if got := atomic.LoadInt64(&staleHit); got != 0 {
		t.Fatalf("expected cache already deleted before DB update, stale hits=%d", got)
	}
}

func TestAIModelRepo_Update_TransactionCommitInvalidatesCache(t *testing.T) {
	repo, _ := newAIModelRepoForTest(t)
	ctx := context.Background()

	model := &models.AIModel{
		ID:          1008,
		Name:        "Model-H",
		Type:        models.ModelTypeChat,
		IsEnabled:   true,
		Description: "old",
	}
	if err := repo.Create(ctx, model); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if _, err := repo.GetByID(ctx, model.ID); err != nil {
		t.Fatalf("GetByID (prime cache) failed: %v", err)
	}

	model.Description = "new"
	if err := repo.Transaction(ctx, func(txCtx context.Context) error {
		return repo.Update(txCtx, model)
	}); err != nil {
		t.Fatalf("transactional Update failed: %v", err)
	}

	after, err := repo.GetByID(ctx, model.ID)
	if err != nil {
		t.Fatalf("GetByID after transactional Update failed: %v", err)
	}
	if after.Description != "new" {
		t.Fatalf("expected cache invalidated after tx commit, got stale description=%q", after.Description)
	}
}

func TestAIModelRepo_SetEnabled_TransactionCommitInvalidatesCache(t *testing.T) {
	repo, _ := newAIModelRepoForTest(t)
	ctx := context.Background()

	model := &models.AIModel{
		ID:        1009,
		Name:      "Model-I",
		Type:      models.ModelTypeChat,
		IsEnabled: true,
	}
	if err := repo.Create(ctx, model); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if _, err := repo.GetByID(ctx, model.ID); err != nil {
		t.Fatalf("GetByID (prime cache) failed: %v", err)
	}

	if err := repo.Transaction(ctx, func(txCtx context.Context) error {
		return repo.SetEnabled(txCtx, model.ID, false)
	}); err != nil {
		t.Fatalf("transactional SetEnabled failed: %v", err)
	}

	after, err := repo.GetByID(ctx, model.ID)
	if err != nil {
		t.Fatalf("GetByID after transactional SetEnabled failed: %v", err)
	}
	if after.IsEnabled {
		t.Fatalf("expected disabled model after tx commit, got stale enabled=true")
	}
}

func TestAIModelRepo_Delete_TransactionCommitInvalidatesCache(t *testing.T) {
	repo, _ := newAIModelRepoForTest(t)
	ctx := context.Background()

	model := &models.AIModel{
		ID:        1011,
		Name:      "Model-J",
		Type:      models.ModelTypeChat,
		IsEnabled: true,
	}
	if err := repo.Create(ctx, model); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if _, err := repo.GetByID(ctx, model.ID); err != nil {
		t.Fatalf("GetByID (prime cache) failed: %v", err)
	}

	if err := repo.Transaction(ctx, func(txCtx context.Context) error {
		return repo.Delete(txCtx, model.ID)
	}); err != nil {
		t.Fatalf("transactional Delete failed: %v", err)
	}

	_, err := repo.GetByID(ctx, model.ID)
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected not found after transactional Delete, got err=%v", err)
	}
}

func TestAIModelRepo_Create_TransactionCommitInvalidatesCache(t *testing.T) {
	repo, _ := newAIModelRepoForTest(t)
	ctx := context.Background()

	model := &models.AIModel{
		ID:        1012,
		Name:      "Model-K",
		Type:      models.ModelTypeChat,
		IsEnabled: true,
	}
	stale := *model
	stale.Name = "Model-K-stale"
	if err := jsoncache.Set(repo.cache, aiModelByIDCacheKey(model.ID), stale); err != nil {
		t.Fatalf("seed stale id cache failed: %v", err)
	}
	if err := jsoncache.Set(repo.cache, aiModelByNameCacheKey(model.Name), stale); err != nil {
		t.Fatalf("seed stale name cache failed: %v", err)
	}

	if err := repo.Transaction(ctx, func(txCtx context.Context) error {
		return repo.Create(txCtx, model)
	}); err != nil {
		t.Fatalf("transactional Create failed: %v", err)
	}

	after, err := repo.GetByID(ctx, model.ID)
	if err != nil {
		t.Fatalf("GetByID after transactional Create failed: %v", err)
	}
	if after.Name != model.Name {
		t.Fatalf("expected recreated model after tx commit, got name=%q", after.Name)
	}
}

func TestAIModelRepo_BatchCreate_TransactionCommitInvalidatesCache(t *testing.T) {
	repo, _ := newAIModelRepoForTest(t)
	ctx := context.Background()

	model := &models.AIModel{
		ID:        1013,
		Name:      "Model-L",
		Type:      models.ModelTypeChat,
		IsEnabled: true,
	}
	stale := *model
	stale.Name = "Model-L-stale"
	if err := jsoncache.Set(repo.cache, aiModelByIDCacheKey(model.ID), stale); err != nil {
		t.Fatalf("seed stale id cache failed: %v", err)
	}
	if err := jsoncache.Set(repo.cache, aiModelByNameCacheKey(model.Name), stale); err != nil {
		t.Fatalf("seed stale name cache failed: %v", err)
	}

	if err := repo.Transaction(ctx, func(txCtx context.Context) error {
		return repo.BatchCreate(txCtx, []*models.AIModel{model})
	}); err != nil {
		t.Fatalf("transactional BatchCreate failed: %v", err)
	}

	after, err := repo.GetByID(ctx, model.ID)
	if err != nil {
		t.Fatalf("GetByID after transactional BatchCreate failed: %v", err)
	}
	if after.Name != model.Name {
		t.Fatalf("expected batch recreated model after tx commit, got name=%q", after.Name)
	}
}

func newAIModelRepoForTest(t *testing.T) (*aiModelRepo, *gorm.DB) {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "model_repo_test.sqlite")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		PrepareStmt:                              true,
		DisableForeignKeyConstraintWhenMigrating: true,
		SkipDefaultTransaction:                   true,
	})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(&models.AIModel{}); err != nil {
		t.Fatalf("auto migrate failed: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db failed: %v", err)
	}
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	cacheCfg := jsoncache.DefaultConfig()
	cacheCfg.Verbose = false
	c, err := jsoncache.New(context.Background(), cacheCfg)
	if err != nil {
		t.Fatalf("create cache failed: %v", err)
	}

	return &aiModelRepo{
		Repository: &Repository{
			db:    db,
			cache: c,
		},
	}, db
}
