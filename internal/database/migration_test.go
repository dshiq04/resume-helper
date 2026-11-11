package database

import (
	"path/filepath"
	"testing"
	"time"

	"resume-helper/internal/config"
	"resume-helper/internal/models"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type legacyApplication struct {
	ID          uint   `gorm:"primaryKey"`
	Category    string `gorm:"type:varchar(32);not null;index"`
	CompanyName string `gorm:"type:varchar(255);not null;index"`
	Link        string `gorm:"type:text"`
	Status      string `gorm:"type:varchar(32);not null;default:screening;index"`
	Pinned      bool   `gorm:"not null;default:false;index"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (legacyApplication) TableName() string {
	return "applications"
}

func TestOpenBackfillsAppliedAt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.db")
	raw, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		t.Fatalf("open legacy database: %v", err)
	}
	if err := raw.AutoMigrate(&legacyApplication{}); err != nil {
		t.Fatalf("migrate legacy schema: %v", err)
	}
	createdAt := time.Now().Add(-24 * time.Hour).Truncate(time.Second)
	if err := raw.Create(&legacyApplication{
		Category:    models.CategorySpring,
		CompanyName: "Legacy Corp",
		Status:      models.StatusScreening,
		CreatedAt:   createdAt,
		UpdatedAt:   createdAt,
	}).Error; err != nil {
		t.Fatalf("insert legacy row: %v", err)
	}
	if sqlDB, err := raw.DB(); err == nil {
		_ = sqlDB.Close()
	}

	db, err := Open(config.DatabaseConfig{Driver: "sqlite", DSN: path})
	if err != nil {
		t.Fatalf("Open(): %v", err)
	}
	defer func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	}()

	var record models.Application
	if err := db.First(&record).Error; err != nil {
		t.Fatalf("load migrated row: %v", err)
	}
	if record.AppliedAt.IsZero() {
		t.Fatal("applied_at should be backfilled for legacy rows")
	}
	if record.AppliedAt.Sub(createdAt).Abs() > time.Second {
		t.Fatalf("applied_at = %v, want %v", record.AppliedAt, createdAt)
	}
}
