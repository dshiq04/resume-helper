package database

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"resume-helper/internal/config"
	"resume-helper/internal/models"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Open connects to the configured database and applies the schema migration.
func Open(cfg config.DatabaseConfig) (*gorm.DB, error) {
	var dialector gorm.Dialector

	switch cfg.Driver {
	case "sqlite":
		dialector = sqlite.Open(cfg.DSN)
	case "mysql":
		dialector = mysql.Open(cfg.DSN)
	case "postgres":
		dialector = postgres.Open(cfg.DSN)
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", cfg.Driver)
	}

	dbLogger := logger.New(log.New(os.Stdout, "\r\n", log.LstdFlags), logger.Config{
		SlowThreshold:             500 * time.Millisecond,
		LogLevel:                  logger.Warn,
		IgnoreRecordNotFoundError: true,
		Colorful:                  false,
	})
	db, err := gorm.Open(dialector, &gorm.Config{
		Logger:                                   dbLogger,
		DisableForeignKeyConstraintWhenMigrating: true,
		NowFunc: func() time.Time {
			return time.Now().Local()
		},
	})
	if err != nil {
		return nil, fmt.Errorf("connect to %s: %w", cfg.Driver, err)
	}

	if cfg.Driver == "sqlite" {
		sqlDB, err := db.DB()
		if err != nil {
			return nil, fmt.Errorf("get sqlite connection: %w", err)
		}
		// A single writer is plenty for a desktop application and avoids
		// SQLITE_BUSY under rapid inline edits.
		sqlDB.SetMaxOpenConns(1)
		sqlDB.SetMaxIdleConns(1)
		for _, pragma := range []string{
			"PRAGMA journal_mode=WAL;",
			"PRAGMA busy_timeout=5000;",
			"PRAGMA foreign_keys=ON;",
		} {
			if err := db.Exec(pragma).Error; err != nil {
				return nil, fmt.Errorf("apply sqlite pragma: %w", err)
			}
		}
	} else {
		sqlDB, err := db.DB()
		if err != nil {
			return nil, fmt.Errorf("get database connection: %w", err)
		}
		sqlDB.SetMaxOpenConns(10)
		sqlDB.SetMaxIdleConns(5)
		sqlDB.SetConnMaxLifetime(time.Hour)
	}

	if err := db.AutoMigrate(
		&models.Application{},
		&models.Setting{},
	); err != nil {
		return nil, fmt.Errorf("migrate database: %w", err)
	}

	return db, nil
}

// SafeDSN removes credentials from a DSN so it can be shown in the UI.
func SafeDSN(driver, dsn string) string {
	if driver == "sqlite" {
		return dsn
	}
	if dsn == "" {
		return ""
	}
	return "••••••••"
}

// DatabaseLabel returns a human readable database description.
func DatabaseLabel(driver string) string {
	switch strings.ToLower(driver) {
	case "mysql":
		return "MySQL"
	case "postgres", "postgresql":
		return "PostgreSQL"
	default:
		return "SQLite"
	}
}
