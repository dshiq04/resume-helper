package database

import (
	"path/filepath"
	"testing"

	"resume-helper/internal/config"
)

func TestBuildAndParseMySQLSettings(t *testing.T) {
	dsn, err := BuildDSN(Settings{
		Driver:   "mysql",
		Host:     "127.0.0.1",
		Port:     3307,
		User:     "resume",
		Password: "p@ss word",
		Database: "resume_helper",
	})
	if err != nil {
		t.Fatalf("BuildDSN(): %v", err)
	}

	parsed := ParseSettings("mysql", dsn, t.TempDir())
	if parsed.ParseError != "" {
		t.Fatalf("ParseSettings() error: %s", parsed.ParseError)
	}
	if parsed.Host != "127.0.0.1" || parsed.Port != 3307 {
		t.Fatalf("unexpected address: %s:%d", parsed.Host, parsed.Port)
	}
	if parsed.User != "resume" || parsed.Password != "p@ss word" || parsed.Database != "resume_helper" {
		t.Fatalf("unexpected mysql settings: %+v", parsed)
	}
}

func TestBuildAndParsePostgresSettings(t *testing.T) {
	dsn, err := BuildDSN(Settings{
		Driver:   "postgres",
		Host:     "db.example.com",
		Port:     5433,
		User:     "resume",
		Password: "s3cret/with?chars",
		Database: "resume helper",
		SSLMode:  "require",
	})
	if err != nil {
		t.Fatalf("BuildDSN(): %v", err)
	}

	parsed := ParseSettings("postgres", dsn, t.TempDir())
	if parsed.ParseError != "" {
		t.Fatalf("ParseSettings() error: %s", parsed.ParseError)
	}
	if parsed.Host != "db.example.com" || parsed.Port != 5433 {
		t.Fatalf("unexpected address: %s:%d", parsed.Host, parsed.Port)
	}
	if parsed.User != "resume" || parsed.Password != "s3cret/with?chars" || parsed.Database != "resume helper" {
		t.Fatalf("unexpected postgres settings: %+v", parsed)
	}
	if parsed.SSLMode != "require" {
		t.Fatalf("SSLMode = %q, want require", parsed.SSLMode)
	}
}

func TestTestConnectionSQLite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "connection.db")
	if err := TestConnection(config.DatabaseConfig{Driver: "sqlite", DSN: path}); err != nil {
		t.Fatalf("TestConnection(): %v", err)
	}
}
