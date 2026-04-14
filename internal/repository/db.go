package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/RenaLio/tudou/internal/config"
	"github.com/RenaLio/tudou/internal/pkg/log"
	"github.com/RenaLio/tudou/internal/pkg/zapgorm"
	"github.com/RenaLio/tudou/pkg/cache"
	"github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func NewDB(conf *config.Config, logger *log.Logger) *gorm.DB {
	dbLogger := zapgorm.New(logger.Logger)
	driver := conf.Data.DB.User.Driver
	dsn := conf.Data.DB.User.DSN

	// GORM doc: https://gorm.io/docs/connecting_to_the_database.html
	var db *gorm.DB
	switch driver {
	case "mysql":
		db = initMysql(dsn, dbLogger)
	case "postgres":
		db = initPostgres(dsn, dbLogger)
	case "sqlite":
		db = initSqlite(dsn, dbLogger)
	default:
		panic("unknown db driver")
	}

	if conf.Debug {
		db = db.Debug()
	}
	return db
}

func initSqlite(dsn string, dbLogger gormlogger.Interface) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		//Logger:                                   dbLogger,
		PrepareStmt:                              true,
		DisableForeignKeyConstraintWhenMigrating: true,
		SkipDefaultTransaction:                   true,
	})
	if err != nil {
		panic(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		panic(err)
	}
	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetMaxIdleConns(50)
	sqlDB.SetConnMaxLifetime(time.Hour)
	return db
}

// initMysql 初始化 MySQL 连接
func initMysql(dsn string, dbLogger gormlogger.Interface) *gorm.DB {
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger:                                   dbLogger,
		PrepareStmt:                              true,
		DisableForeignKeyConstraintWhenMigrating: true,
		SkipDefaultTransaction:                   true,
	})
	if err != nil {
		panic(fmt.Errorf("mysql connection failed: %w", err))
	}

	setupCommonPool(db)
	return db
}

func initPostgres(dsn string, dbLogger gormlogger.Interface) *gorm.DB {
	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN:                  dsn,
		PreferSimpleProtocol: true, // disables implicit prepared statement usage
	}), &gorm.Config{
		Logger:                                   dbLogger,
		DisableForeignKeyConstraintWhenMigrating: true,
		PrepareStmt:                              true,
		SkipDefaultTransaction:                   true,
	})
	if err != nil {
		panic(fmt.Errorf("postgres connection failed: %w", err))
	}

	setupCommonPool(db)
	return db
}

func setupCommonPool(db *gorm.DB) {
	sqlDB, err := db.DB()
	if err != nil {
		panic(fmt.Errorf("failed to get sql.DB: %w", err))
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)
}

func NewCache(conf *config.Config, logger *log.Logger) *cache.JsonCache {
	ctx := context.Background()
	cache, err := cache.New(ctx, cache.DefaultConfig())
	if err != nil {
		panic(err)
	}
	return cache
}
