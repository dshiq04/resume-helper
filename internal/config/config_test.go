package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func isolateEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"RESUME_HELPER_CONFIG",
		"RESUME_HELPER_DATA_DIR",
		"RESUME_HELPER_DB_DRIVER",
		"RESUME_HELPER_DB_DSN",
		"RESUME_HELPER_HOST",
		"RESUME_HELPER_PORT",
		"RESUME_HELPER_DEV",
	} {
		t.Setenv(key, "")
	}
}

func TestLoadCreatesPortableConfigInDataDir(t *testing.T) {
	isolateEnv(t)
	dir := t.TempDir()
	t.Setenv("RESUME_HELPER_DATA_DIR", dir)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load(): %v", err)
	}

	if cfg.DataDir != dir {
		t.Fatalf("DataDir = %q, want %q", cfg.DataDir, dir)
	}
	wantConfig := filepath.Join(dir, "config.yaml")
	if cfg.SourcePath != wantConfig {
		t.Fatalf("SourcePath = %q, want %q", cfg.SourcePath, wantConfig)
	}
	wantDB := filepath.Join(dir, "resume-helper.db")
	if cfg.Database.DSN != wantDB {
		t.Fatalf("DSN = %q, want %q", cfg.Database.DSN, wantDB)
	}
	if _, err := os.Stat(wantConfig); err != nil {
		t.Fatalf("default config was not created: %v", err)
	}
	raw, err := os.ReadFile(wantConfig)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), dir) {
		t.Fatal("portable config must not embed an absolute machine path")
	}
}

func TestLoadHonorsExplicitConfigAndRelativeSQLitePath(t *testing.T) {
	isolateEnv(t)
	dataDir := t.TempDir()
	configDir := t.TempDir()
	t.Setenv("RESUME_HELPER_DATA_DIR", dataDir)
	configFile := filepath.Join(configDir, "custom.yaml")
	content := "server:\n  port: 19999\ndatabase:\n  driver: sqlite\n  dsn: \"custom.db\"\n"
	if err := os.WriteFile(configFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("RESUME_HELPER_CONFIG", configFile)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load(): %v", err)
	}
	if cfg.Server.Port != 19999 {
		t.Fatalf("port = %d, want 19999", cfg.Server.Port)
	}
	if cfg.SourcePath != configFile {
		t.Fatalf("SourcePath = %q, want %q", cfg.SourcePath, configFile)
	}
	wantDB := filepath.Join(dataDir, "custom.db")
	if cfg.Database.DSN != wantDB {
		t.Fatalf("relative SQLite DSN = %q, want %q", cfg.Database.DSN, wantDB)
	}
}

func TestLoadRejectsMissingExplicitConfig(t *testing.T) {
	isolateEnv(t)
	t.Setenv("RESUME_HELPER_DATA_DIR", t.TempDir())
	t.Setenv("RESUME_HELPER_CONFIG", filepath.Join(t.TempDir(), "missing.yaml"))

	if _, err := Load(); err == nil {
		t.Fatal("expected an error for a missing explicit config file")
	}
}

func TestExplicitConfigUsesConfigFolderForRelativeSQLite(t *testing.T) {
	isolateEnv(t)
	configDir := t.TempDir()
	configFile := filepath.Join(configDir, "portable.yaml")
	content := "database:\n  driver: sqlite\n  dsn: \"portable.db\"\n"
	if err := os.WriteFile(configFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("RESUME_HELPER_CONFIG", configFile)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load(): %v", err)
	}
	if cfg.DataDir != configDir {
		t.Fatalf("DataDir = %q, want %q", cfg.DataDir, configDir)
	}
	wantDB := filepath.Join(configDir, "portable.db")
	if cfg.Database.DSN != wantDB {
		t.Fatalf("SQLite DSN = %q, want %q", cfg.Database.DSN, wantDB)
	}
}
