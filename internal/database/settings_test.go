package database

import (
	"path/filepath"
	"testing"

	"resume-helper/internal/config"
)

func TestBuildAndParseSQLiteSettings(t *testing.T) {
	dsn, err := BuildDSN(Settings{
		Driver:     "sqlite",
		SQLitePath: "data/custom.db",
	})
	if err != nil {
		t.Fatalf("BuildDSN(): %v", err)
	}
	if dsn != "data/custom.db" {
		t.Fatalf("dsn = %q, want data/custom.db", dsn)
	}

	dataDir := t.TempDir()
	parsed := ParseSettings("sqlite", dsn, dataDir)
	if parsed.ParseError != "" {
		t.Fatalf("ParseSettings() error: %s", parsed.ParseError)
	}
	if parsed.Driver != "sqlite" || parsed.SQLitePath != "data/custom.db" {
		t.Fatalf("unexpected sqlite settings: %+v", parsed)
	}
}

func TestParseSettingsRejectsUnsupportedDriver(t *testing.T) {
	parsed := ParseSettings("mysql", "user:password@tcp(127.0.0.1:3306)/resume_helper", t.TempDir())
	if parsed.ParseError == "" {
		t.Fatal("unsupported driver should be reported as a parse error")
	}
}

func TestTestConnectionSQLite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "connection.db")
	if err := TestConnection(config.DatabaseConfig{Driver: "sqlite", DSN: path}); err != nil {
		t.Fatalf("TestConnection(): %v", err)
	}
}

func TestTestConnectionRejectsUnsupportedDriver(t *testing.T) {
	if err := TestConnection(config.DatabaseConfig{Driver: "postgres", DSN: "postgres://localhost/db"}); err == nil {
		t.Fatal("expected an error for an unsupported driver")
	}
}
