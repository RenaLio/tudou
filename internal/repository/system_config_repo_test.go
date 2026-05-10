package repository

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/RenaLio/tudou/internal/models"
	jsoncache "github.com/RenaLio/tudou/pkg/cache"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func newSystemConfigRepoForTest(t *testing.T) (*systemConfigRepo, *gorm.DB) {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "system_config_repo_test.sqlite")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		PrepareStmt:                              true,
		DisableForeignKeyConstraintWhenMigrating: true,
		SkipDefaultTransaction:                   true,
	})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(&models.SystemConfig{}); err != nil {
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

	return &systemConfigRepo{
		Repository: &Repository{
			db:    db,
			cache: c,
		},
	}, db
}

func seedSystemConfig(t *testing.T, db *gorm.DB, key string, value any) *models.SystemConfig {
	t.Helper()
	cfg := &models.SystemConfig{
		ID:    int64(len(key)) * 1000, // deterministic ID for tests
		Key:   key,
		Value: models.ConfigValue{ValueData: value},
		Type:  models.ConfigTypeString,
		Scope: models.ConfigScopeSystem,
	}
	if err := db.Create(cfg).Error; err != nil {
		t.Fatalf("seed config %q failed: %v", key, err)
	}
	return cfg
}

func TestSystemConfigRepo_CacheGetByKey(t *testing.T) {
	repo, db := newSystemConfigRepoForTest(t)
	seedSystemConfig(t, db, "site.name", "Test Site")

	// First read → DB + cache fill
	got, err := repo.GetByKey(context.Background(), "site.name")
	if err != nil {
		t.Fatalf("GetByKey: %v", err)
	}
	if got.GetString() != "Test Site" {
		t.Fatalf("unexpected value: %v", got.Value.ValueData)
	}

	// Mutate DB directly → cache should still return old value
	if err := db.Model(&models.SystemConfig{}).Where("key = ?", "site.name").Update("value", `{"value":"Stale"}`).Error; err != nil {
		t.Fatalf("direct update: %v", err)
	}
	got2, err := repo.GetByKey(context.Background(), "site.name")
	if err != nil {
		t.Fatalf("GetByKey cached: %v", err)
	}
	if got2.GetString() != "Test Site" {
		t.Fatalf("expected cached value 'Test Site', got %q", got2.GetString())
	}
}

func TestSystemConfigRepo_CacheGetByID(t *testing.T) {
	repo, db := newSystemConfigRepoForTest(t)
	cfg := seedSystemConfig(t, db, "site.logo", "/logo.png")

	got, err := repo.GetByID(context.Background(), cfg.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Key != "site.logo" {
		t.Fatalf("unexpected key: %s", got.Key)
	}
}

func TestSystemConfigRepo_CreateInvalidatesCache(t *testing.T) {
	repo, _ := newSystemConfigRepoForTest(t)

	cfg := &models.SystemConfig{
		ID:    9001,
		Key:   "new.key",
		Value: models.ConfigValue{ValueData: "hello"},
		Type:  models.ConfigTypeString,
		Scope: models.ConfigScopeSystem,
	}
	if err := repo.Create(context.Background(), cfg); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := repo.GetByKey(context.Background(), "new.key")
	if err != nil {
		t.Fatalf("GetByKey: %v", err)
	}
	if got.GetString() != "hello" {
		t.Fatalf("expected 'hello', got %q", got.GetString())
	}
}

func TestSystemConfigRepo_UpsertInvalidatesCache(t *testing.T) {
	repo, db := newSystemConfigRepoForTest(t)
	cfg := seedSystemConfig(t, db, "site.name", "Original")

	// Warm cache
	got, err := repo.GetByKey(context.Background(), "site.name")
	if err != nil {
		t.Fatalf("GetByKey: %v", err)
	}
	if got.GetString() != "Original" {
		t.Fatalf("unexpected: %q", got.GetString())
	}

	// Upsert → should invalidate cache
	cfg.Value = models.ConfigValue{ValueData: "Updated"}
	if err := repo.Upsert(context.Background(), cfg); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	got2, err := repo.GetByKey(context.Background(), "site.name")
	if err != nil {
		t.Fatalf("GetByKey after upsert: %v", err)
	}
	if got2.GetString() != "Updated" {
		t.Fatalf("expected 'Updated', got %q", got2.GetString())
	}
}

func TestSystemConfigRepo_SetValueByKeyInvalidatesCache(t *testing.T) {
	repo, db := newSystemConfigRepoForTest(t)
	seedSystemConfig(t, db, "site.name", "Original")

	// Warm cache
	_, _ = repo.GetByKey(context.Background(), "site.name")

	// SetValueByKey → should invalidate
	if err := repo.SetValueByKey(context.Background(), "site.name", "Changed"); err != nil {
		t.Fatalf("SetValueByKey: %v", err)
	}

	got, err := repo.GetByKey(context.Background(), "site.name")
	if err != nil {
		t.Fatalf("GetByKey: %v", err)
	}
	if got.GetString() != "Changed" {
		t.Fatalf("expected 'Changed', got %q", got.GetString())
	}
}

func TestSystemConfigRepo_DeleteByKeyInvalidatesCache(t *testing.T) {
	repo, db := newSystemConfigRepoForTest(t)
	seedSystemConfig(t, db, "site.name", "ToDelete")

	// Warm cache
	_, _ = repo.GetByKey(context.Background(), "site.name")

	// Delete → should invalidate
	if err := repo.DeleteByKey(context.Background(), "site.name"); err != nil {
		t.Fatalf("delete: %v", err)
	}

	_, err := repo.GetByKey(context.Background(), "site.name")
	if err == nil {
		t.Fatalf("expected error after delete, got nil")
	}
}

func TestSystemConfigRepo_CacheDisabledInTransaction(t *testing.T) {
	repo, db := newSystemConfigRepoForTest(t)
	cfg := seedSystemConfig(t, db, "tx.test", "before")

	// Warm cache
	_, _ = repo.GetByKey(context.Background(), "tx.test")

	// Update inside transaction → cache should NOT be read (no stale data)
	err := repo.Transaction(context.Background(), func(ctx context.Context) error {
		cfg.Value = models.ConfigValue{ValueData: "during_tx"}
		return repo.Upsert(ctx, cfg)
	})
	if err != nil {
		t.Fatalf("transaction: %v", err)
	}

	// After commit, cache should be invalidated
	got, err := repo.GetByKey(context.Background(), "tx.test")
	if err != nil {
		t.Fatalf("GetByKey after tx: %v", err)
	}
	if got.GetString() != "during_tx" {
		t.Fatalf("expected 'during_tx', got %q", got.GetString())
	}
}
