package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	AppDirName    = "ResumeHelper"
	defaultPort   = 19876
	defaultHost   = "127.0.0.1"
	defaultDriver = "sqlite"
)

// Config contains every setting that can be supplied through config.yaml or
// environment variables. SourcePath and DataDir are populated at runtime.
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`

	SourcePath string `yaml:"-"`
	DataDir    string `yaml:"-"`
}

type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type DatabaseConfig struct {
	// Driver is one of: sqlite, mysql, postgres.
	Driver string `yaml:"driver"`
	// DSN is the GORM driver DSN. When empty for sqlite a database file inside
	// the user's config directory is used.
	DSN string `yaml:"dsn"`
}

func Default() Config {
	return Config{
		Server: ServerConfig{
			Host: defaultHost,
			Port: defaultPort,
		},
		Database: DatabaseConfig{
			Driver: defaultDriver,
		},
	}
}

// Load reads the first config file it finds and applies environment overrides.
// Search order: RESUME_HELPER_CONFIG, ./config.yaml, <user config>/ResumeHelper/config.yaml.
// If no file exists a commented default is written to the user config folder.
func Load() (*Config, error) {
	cfg := Default()
	dataDir, err := userDataDir()
	if err != nil {
		return nil, err
	}
	cfg.DataDir = dataDir

	path := findConfigFile(dataDir)
	if path != "" {
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read config %s: %w", path, err)
		}
		if err := yaml.Unmarshal(raw, &cfg); err != nil {
			return nil, fmt.Errorf("parse config %s: %w", path, err)
		}
		cfg.SourcePath = path
	} else {
		path = filepath.Join(dataDir, "config.yaml")
		if err := os.WriteFile(path, []byte(defaultConfigFile(dataDir)), 0o644); err != nil {
			return nil, fmt.Errorf("write default config: %w", err)
		}
		cfg.SourcePath = path
	}

	applyEnv(&cfg)

	cfg.Database.Driver = strings.ToLower(strings.TrimSpace(cfg.Database.Driver))
	switch cfg.Database.Driver {
	case "", "sqlite", "sqlite3":
		cfg.Database.Driver = "sqlite"
	case "postgresql", "pg":
		cfg.Database.Driver = "postgres"
	case "mysql":
		cfg.Database.Driver = "mysql"
	default:
		return nil, fmt.Errorf("unsupported database driver %q (use sqlite, mysql or postgres)", cfg.Database.Driver)
	}

	if cfg.Server.Port < 0 || cfg.Server.Port > 65535 {
		return nil, fmt.Errorf("server.port must be between 0 and 65535")
	}

	if cfg.Database.Driver == "sqlite" && strings.TrimSpace(cfg.Database.DSN) == "" {
		cfg.Database.DSN = filepath.Join(dataDir, "resume-helper.db")
	}

	if cfg.Database.Driver != "sqlite" && strings.TrimSpace(cfg.Database.DSN) == "" {
		return nil, fmt.Errorf("database.dsn is required when database.driver is %s", cfg.Database.Driver)
	}

	return &cfg, nil
}

func applyEnv(cfg *Config) {
	if v := os.Getenv("RESUME_HELPER_HOST"); v != "" {
		cfg.Server.Host = v
	}
	if v := os.Getenv("RESUME_HELPER_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.Server.Port = port
		}
	}
	if v := os.Getenv("RESUME_HELPER_DB_DRIVER"); v != "" {
		cfg.Database.Driver = v
	}
	if v := os.Getenv("RESUME_HELPER_DB_DSN"); v != "" {
		cfg.Database.DSN = v
	}
}

func findConfigFile(dataDir string) string {
	if path := strings.TrimSpace(os.Getenv("RESUME_HELPER_CONFIG")); path != "" {
		return path
	}
	if _, err := os.Stat("config.yaml"); err == nil {
		return "config.yaml"
	}
	path := filepath.Join(dataDir, "config.yaml")
	if _, err := os.Stat(path); err == nil {
		return path
	}
	return ""
}

func userDataDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("locate user config directory: %w", err)
	}
	if base == "" {
		return "", errors.New("user config directory is unavailable")
	}
	dir := filepath.Join(base, AppDirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create config directory: %w", err)
	}
	return dir, nil
}

func defaultConfigFile(dataDir string) string {
	dbPath := filepath.ToSlash(filepath.Join(dataDir, "resume-helper.db"))
	return fmt.Sprintf(`# ResumeTrail configuration
# Edit this file and restart the application for changes to take effect.
# The database password is never stored here; application passwords are
# hashed with bcrypt inside the database.

server:
  host: 127.0.0.1
  port: 19876

database:
  # sqlite | mysql | postgres
  driver: sqlite
  # SQLite: optional path. Defaults to %s
  # MySQL example:
  #   dsn: "user:password@tcp(127.0.0.1:3306)/resume_helper?charset=utf8mb4&parseTime=True&loc=Local"
  # PostgreSQL example:
  #   dsn: "host=127.0.0.1 user=postgres password=secret dbname=resume_helper port=5432 sslmode=disable TimeZone=Local"
  dsn: "%s"
`, dbPath, dbPath)
}
