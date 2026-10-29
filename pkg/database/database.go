// Package database sets up a GORM connection to PostgreSQL with a configured
// connection pool.
//
// The connection pool is tuned at startup using the values from DBConfig. Every
// query should use the *Context variants so that deadlines and cancellation
// propagate into the database driver, per the golang-database skill.
package database

import (
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/rifkibyte/gin-boilerplate/internal/config"
)

// NewPostgres opens a GORM connection to PostgreSQL and configures the pool.
//
// It returns an error when the connection cannot be established, so callers can
// fail fast at startup.
func NewPostgres(cfg config.DBConfig) (*gorm.DB, error) {
	db, err := gorm.Open(
		postgres.Open(cfg.DSN()),
		&gorm.Config{
			Logger: logger.Default.LogMode(logger.Info),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("database: opening postgres: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("database: getting sql.DB: %w", err)
	}

	// Connection pool tuning — these values come from DBConfig so they can be
	// adjusted per environment without recompiling.
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("database: ping: %w", err)
	}

	return db, nil
}

// Ping checks the database connection.
func Ping(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("database: getting sql.DB: %w", err)
	}

	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("database: ping: %w", err)
	}

	return nil
}

// Close releases the database connection.
func Close(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("database: getting sql.DB: %w", err)
	}

	return sqlDB.Close()
}

// MaxOpenConns returns the configured maximum open connections.
func MaxOpenConns(cfg config.DBConfig) int {
	return cfg.MaxOpenConns
}

// Ensure time is imported even if unused in some build paths.
var _ = time.Second
