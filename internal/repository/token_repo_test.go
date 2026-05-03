package repository

import (
	"context"
	"errors"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/RenaLio/tudou/internal/models"
	jsoncache "github.com/RenaLio/tudou/pkg/cache"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type queryCountEnabledKey struct{}

func TestTokenRepo_GetAvailableByToken_CacheHitReducesDBQueries(t *testing.T) {
	repo, db := newTokenRepoForTest(t)
	tokenValue := seedAvailableTokenForTest(t, db)

	const callbackName = "test:count_queries_for_token_cache"
	var queryCount int64
	if err := db.Callback().Query().Before("gorm:query").Register(callbackName, func(tx *gorm.DB) {
		if tx == nil || tx.Statement == nil || tx.Statement.Context == nil {
			return
		}
		if enabled, _ := tx.Statement.Context.Value(queryCountEnabledKey{}).(bool); enabled {
			atomic.AddInt64(&queryCount, 1)
		}
	}); err != nil {
		t.Fatalf("register query callback failed: %v", err)
	}
	defer func() {
		_ = db.Callback().Query().Remove(callbackName)
	}()

	countQueries := func(call func(ctx context.Context) error) (int64, error) {
		start := atomic.LoadInt64(&queryCount)
		err := call(context.WithValue(context.Background(), queryCountEnabledKey{}, true))
		end := atomic.LoadInt64(&queryCount)
		return end - start, err
	}

	firstCount, err := countQueries(func(ctx context.Context) error {
		token, callErr := repo.GetAvailableByToken(ctx, tokenValue)
		if callErr != nil {
			return callErr
		}
		if token == nil || token.Group.Name == "" || token.User.Username == "" {
			t.Fatalf("expected token with preloaded user/group, got %+v", token)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("first GetAvailableByToken failed: %v", err)
	}

	secondCount, err := countQueries(func(ctx context.Context) error {
		_, callErr := repo.GetAvailableByToken(ctx, tokenValue)
		return callErr
	})
	if err != nil {
		t.Fatalf("second GetAvailableByToken failed: %v", err)
	}

	if secondCount >= firstCount {
		t.Fatalf("expected second call to use fewer DB queries with cache, first=%d second=%d", firstCount, secondCount)
	}
}

func TestTokenRepo_Create_DoesNotCacheSparseAvailableToken(t *testing.T) {
	repo, db := newTokenRepoForTest(t)

	user := &models.User{
		ID:       2201,
		Username: "create-cache-user",
		Password: "pwd",
		Status:   models.UserStatusEnabled,
		Role:     models.UserRoleUser,
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}
	group := &models.ChannelGroup{
		ID:                  3201,
		Name:                "create-cache-group",
		LoadBalanceStrategy: models.LoadBalanceStrategyPerformance,
	}
	if err := db.Create(group).Error; err != nil {
		t.Fatalf("create group failed: %v", err)
	}

	exp := time.Now().Add(time.Hour)
	token := &models.Token{
		ID:                  4201,
		UserID:              user.ID,
		GroupID:             group.ID,
		Token:               "create-cache-token",
		Name:                "create-cache-token-name",
		Status:              models.TokenStatusEnabled,
		LimitMicros:         10_000_000,
		ExpiresAt:           &exp,
		LoadBalanceStrategy: models.LoadBalanceStrategyPerformance,
	}
	if err := repo.Create(context.Background(), token); err != nil {
		t.Fatalf("repo.Create failed: %v", err)
	}

	got, err := repo.GetAvailableByToken(context.Background(), token.Token)
	if err != nil {
		t.Fatalf("GetAvailableByToken failed: %v", err)
	}
	if got == nil {
		t.Fatalf("expected token, got nil")
	}
	if got.User.ID == 0 || got.User.Username == "" {
		t.Fatalf("expected preloaded user from DB path, got %+v", got.User)
	}
	if got.Group.ID == 0 || got.Group.Name == "" {
		t.Fatalf("expected preloaded group from DB path, got %+v", got.Group)
	}
}

func TestTokenRepo_Update_TransactionCommitInvalidatesCache(t *testing.T) {
	repo, db := newTokenRepoForTest(t)
	ctx := context.Background()
	tokenValue := seedAvailableTokenForTest(t, db)

	token, err := repo.GetByToken(ctx, tokenValue)
	if err != nil {
		t.Fatalf("GetByToken (prime cache) failed: %v", err)
	}

	token.Name = "token-cache-name-updated"
	if err := repo.Transaction(ctx, func(txCtx context.Context) error {
		return repo.Update(txCtx, token)
	}); err != nil {
		t.Fatalf("transactional Update failed: %v", err)
	}

	after, err := repo.GetByToken(ctx, tokenValue)
	if err != nil {
		t.Fatalf("GetByToken after transactional Update failed: %v", err)
	}
	if after.Name != "token-cache-name-updated" {
		t.Fatalf("expected fresh token after tx commit, got stale name=%q", after.Name)
	}
}

func TestTokenRepo_UpdateStatus_TransactionCommitInvalidatesCache(t *testing.T) {
	repo, db := newTokenRepoForTest(t)
	ctx := context.Background()
	tokenValue := seedAvailableTokenForTest(t, db)

	token, err := repo.GetByToken(ctx, tokenValue)
	if err != nil {
		t.Fatalf("GetByToken (prime cache) failed: %v", err)
	}
	if token.Status != models.TokenStatusEnabled {
		t.Fatalf("expected seeded token enabled")
	}

	if err := repo.Transaction(ctx, func(txCtx context.Context) error {
		return repo.UpdateStatus(txCtx, token.ID, models.TokenStatusDisabled)
	}); err != nil {
		t.Fatalf("transactional UpdateStatus failed: %v", err)
	}

	after, err := repo.GetByToken(ctx, tokenValue)
	if err != nil {
		t.Fatalf("GetByToken after transactional UpdateStatus failed: %v", err)
	}
	if after.Status != models.TokenStatusDisabled {
		t.Fatalf("expected disabled status after tx commit, got stale status=%s", after.Status)
	}
}

func TestTokenRepo_Delete_TransactionCommitInvalidatesCache(t *testing.T) {
	repo, db := newTokenRepoForTest(t)
	ctx := context.Background()
	tokenValue := seedAvailableTokenForTest(t, db)

	token, err := repo.GetByToken(ctx, tokenValue)
	if err != nil {
		t.Fatalf("GetByToken (prime cache) failed: %v", err)
	}

	if err := repo.Transaction(ctx, func(txCtx context.Context) error {
		return repo.Delete(txCtx, token.ID)
	}); err != nil {
		t.Fatalf("transactional Delete failed: %v", err)
	}

	_, err = repo.GetByToken(ctx, tokenValue)
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected not found after transactional Delete, got err=%v", err)
	}
}

func TestTokenRepo_Create_TransactionCommitInvalidatesCache(t *testing.T) {
	repo, db := newTokenRepoForTest(t)
	ctx := context.Background()
	user := &models.User{
		ID:       2301,
		Username: "tx-create-user",
		Password: "pwd",
		Status:   models.UserStatusEnabled,
		Role:     models.UserRoleUser,
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}
	group := &models.ChannelGroup{
		ID:                  3301,
		Name:                "tx-create-group",
		LoadBalanceStrategy: models.LoadBalanceStrategyPerformance,
	}
	if err := db.Create(group).Error; err != nil {
		t.Fatalf("create group failed: %v", err)
	}

	expiresAt := time.Now().Add(time.Hour)
	token := &models.Token{
		ID:                  4301,
		UserID:              user.ID,
		GroupID:             group.ID,
		Token:               "tx-create-token",
		Name:                "tx-create-name",
		Status:              models.TokenStatusEnabled,
		LimitMicros:         10_000_000,
		ExpiresAt:           &expiresAt,
		LoadBalanceStrategy: models.LoadBalanceStrategyPerformance,
	}
	stale := *token
	stale.Name = "tx-create-token-stale"
	if err := jsoncache.Set(repo.cache, tokenByIDCacheKey(stale.ID), stale); err != nil {
		t.Fatalf("seed stale id cache failed: %v", err)
	}
	if err := jsoncache.Set(repo.cache, tokenByValueCacheKey(stale.Token), stale); err != nil {
		t.Fatalf("seed stale value cache failed: %v", err)
	}
	if err := jsoncache.Set(repo.cache, tokenByValueRelationsCacheKey(stale.Token), stale); err != nil {
		t.Fatalf("seed stale relations cache failed: %v", err)
	}

	if err := repo.Transaction(ctx, func(txCtx context.Context) error {
		return repo.Create(txCtx, token)
	}); err != nil {
		t.Fatalf("transactional Create failed: %v", err)
	}

	after, err := repo.GetByToken(ctx, token.Token)
	if err != nil {
		t.Fatalf("GetByToken after transactional Create failed: %v", err)
	}
	if after.Name != token.Name {
		t.Fatalf("expected fresh token after tx commit, got stale name=%q", after.Name)
	}
}

func TestTokenRepo_BatchCreate_TransactionCommitInvalidatesCache(t *testing.T) {
	repo, db := newTokenRepoForTest(t)
	ctx := context.Background()
	user := &models.User{
		ID:       2302,
		Username: "tx-batch-user",
		Password: "pwd",
		Status:   models.UserStatusEnabled,
		Role:     models.UserRoleUser,
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}
	group := &models.ChannelGroup{
		ID:                  3302,
		Name:                "tx-batch-group",
		LoadBalanceStrategy: models.LoadBalanceStrategyPerformance,
	}
	if err := db.Create(group).Error; err != nil {
		t.Fatalf("create group failed: %v", err)
	}

	expiresAt := time.Now().Add(time.Hour)
	token := &models.Token{
		ID:                  4302,
		UserID:              user.ID,
		GroupID:             group.ID,
		Token:               "tx-batch-token",
		Name:                "tx-batch-name",
		Status:              models.TokenStatusEnabled,
		LimitMicros:         10_000_000,
		ExpiresAt:           &expiresAt,
		LoadBalanceStrategy: models.LoadBalanceStrategyPerformance,
	}
	stale := *token
	stale.Name = "tx-batch-token-stale"
	if err := jsoncache.Set(repo.cache, tokenByIDCacheKey(stale.ID), stale); err != nil {
		t.Fatalf("seed stale id cache failed: %v", err)
	}
	if err := jsoncache.Set(repo.cache, tokenByValueCacheKey(stale.Token), stale); err != nil {
		t.Fatalf("seed stale value cache failed: %v", err)
	}
	if err := jsoncache.Set(repo.cache, tokenByValueRelationsCacheKey(stale.Token), stale); err != nil {
		t.Fatalf("seed stale relations cache failed: %v", err)
	}

	if err := repo.Transaction(ctx, func(txCtx context.Context) error {
		return repo.BatchCreate(txCtx, []*models.Token{token})
	}); err != nil {
		t.Fatalf("transactional BatchCreate failed: %v", err)
	}

	after, err := repo.GetByToken(ctx, token.Token)
	if err != nil {
		t.Fatalf("GetByToken after transactional BatchCreate failed: %v", err)
	}
	if after.Name != token.Name {
		t.Fatalf("expected fresh token after tx commit, got stale name=%q", after.Name)
	}
}

func newTokenRepoForTest(t *testing.T) (*tokenRepo, *gorm.DB) {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "token_repo_test.sqlite")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		PrepareStmt:                              true,
		DisableForeignKeyConstraintWhenMigrating: true,
		SkipDefaultTransaction:                   true,
	})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.ChannelGroup{}, &models.Token{}, &models.TokenStats{}); err != nil {
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

	return &tokenRepo{
		Repository: &Repository{
			db:    db,
			cache: c,
		},
	}, db
}

func seedAvailableTokenForTest(t *testing.T, db *gorm.DB) string {
	t.Helper()

	user := &models.User{
		ID:       2001,
		Username: "token-cache-user",
		Password: "pwd",
		Status:   models.UserStatusEnabled,
		Role:     models.UserRoleUser,
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}

	group := &models.ChannelGroup{
		ID:                  3001,
		Name:                "token-cache-group",
		LoadBalanceStrategy: models.LoadBalanceStrategyPerformance,
	}
	if err := db.Create(group).Error; err != nil {
		t.Fatalf("create group failed: %v", err)
	}

	expiresAt := time.Now().Add(time.Hour)
	token := &models.Token{
		ID:                  4001,
		UserID:              user.ID,
		GroupID:             group.ID,
		Token:               "token-cache-value",
		Name:                "token-cache-name",
		Status:              models.TokenStatusEnabled,
		LimitMicros:         10_000_000,
		ExpiresAt:           &expiresAt,
		LoadBalanceStrategy: models.LoadBalanceStrategyPerformance,
	}
	if err := db.Create(token).Error; err != nil {
		t.Fatalf("create token failed: %v", err)
	}

	stats := &models.TokenStats{
		TokenID:         token.ID,
		TokenName:       token.Name,
		TotalCostMicros: 100,
	}
	if err := db.Create(stats).Error; err != nil {
		t.Fatalf("create token stats failed: %v", err)
	}

	return token.Token
}

func BenchmarkTokenRepo_GetAvailableByToken(b *testing.B) {
	repo, db := newTokenRepoForBenchmark(b)
	tokenValue := seedAvailableTokenForBenchmark(b, db)

	b.Run("cold_db_path", func(b *testing.B) {
		if repo.cache != nil {
			_ = repo.cache.Clear()
		}
		ctx := context.Background()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if repo.cache != nil {
				_ = repo.cache.Clear()
			}
			_, err := repo.GetAvailableByToken(ctx, tokenValue)
			if err != nil {
				b.Fatalf("GetAvailableByToken failed: %v", err)
			}
		}
	})

	b.Run("warm_cache_hit", func(b *testing.B) {
		ctx := context.Background()
		if _, err := repo.GetAvailableByToken(ctx, tokenValue); err != nil {
			b.Fatalf("warmup failed: %v", err)
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := repo.GetAvailableByToken(ctx, tokenValue)
			if err != nil {
				b.Fatalf("GetAvailableByToken failed: %v", err)
			}
		}
	})
}

func newTokenRepoForBenchmark(b *testing.B) (*tokenRepo, *gorm.DB) {
	b.Helper()

	dbPath := filepath.Join(b.TempDir(), "token_repo_bench.sqlite")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		PrepareStmt:                              true,
		DisableForeignKeyConstraintWhenMigrating: true,
		SkipDefaultTransaction:                   true,
	})
	if err != nil {
		b.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.ChannelGroup{}, &models.Token{}, &models.TokenStats{}); err != nil {
		b.Fatalf("auto migrate failed: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		b.Fatalf("get sql db failed: %v", err)
	}
	b.Cleanup(func() {
		_ = sqlDB.Close()
	})

	cacheCfg := jsoncache.DefaultConfig()
	cacheCfg.Verbose = false
	c, err := jsoncache.New(context.Background(), cacheCfg)
	if err != nil {
		b.Fatalf("create cache failed: %v", err)
	}

	return &tokenRepo{
		Repository: &Repository{
			db:    db,
			cache: c,
		},
	}, db
}

func seedAvailableTokenForBenchmark(b *testing.B, db *gorm.DB) string {
	b.Helper()

	user := &models.User{
		ID:       2101,
		Username: "token-cache-bench-user",
		Password: "pwd",
		Status:   models.UserStatusEnabled,
		Role:     models.UserRoleUser,
	}
	if err := db.Create(user).Error; err != nil {
		b.Fatalf("create user failed: %v", err)
	}

	group := &models.ChannelGroup{
		ID:                  3101,
		Name:                "token-cache-bench-group",
		LoadBalanceStrategy: models.LoadBalanceStrategyPerformance,
	}
	if err := db.Create(group).Error; err != nil {
		b.Fatalf("create group failed: %v", err)
	}

	expiresAt := time.Now().Add(time.Hour)
	token := &models.Token{
		ID:                  4101,
		UserID:              user.ID,
		GroupID:             group.ID,
		Token:               "token-cache-bench-value",
		Name:                "token-cache-bench-name",
		Status:              models.TokenStatusEnabled,
		LimitMicros:         10_000_000,
		ExpiresAt:           &expiresAt,
		LoadBalanceStrategy: models.LoadBalanceStrategyPerformance,
	}
	if err := db.Create(token).Error; err != nil {
		b.Fatalf("create token failed: %v", err)
	}

	stats := &models.TokenStats{
		TokenID:         token.ID,
		TokenName:       token.Name,
		TotalCostMicros: 100,
	}
	if err := db.Create(stats).Error; err != nil {
		b.Fatalf("create token stats failed: %v", err)
	}

	return token.Token
}
