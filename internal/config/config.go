package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
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
	// DSN is the GORM driver DSN. When empty for sqlite, resume-helper.db is
	// created next to the executable.
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

// Load reads the portable config file and applies environment overrides.
//
// By default both config.yaml and the SQLite database live in the application
// directory (the folder containing the executable). During `go run` the
// working directory is used instead. RESUME_HELPER_DATA_DIR can point both
// files at a custom directory, and RESUME_HELPER_CONFIG can select a specific
// config file.
func Load() (*Config, error) {
	cfg := Default()
	dataDir, err := dataDirectory()
	if err != nil {
		return nil, err
	}
	cfg.DataDir = dataDir

	path, err := findConfigFile(dataDir)
	if err != nil {
		return nil, err
	}
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
			return nil, fmt.Errorf(
				"write default config to %s: %w (move the application to a writable folder or set RESUME_HELPER_DATA_DIR)",
				path,
				err,
			)
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

	if cfg.Database.Driver == "sqlite" {
		dsn := strings.TrimSpace(cfg.Database.DSN)
		switch {
		case dsn == "":
			cfg.Database.DSN = filepath.Join(cfg.DataDir, "resume-helper.db")
		case dsn == ":memory:" || strings.HasPrefix(dsn, "file:"):
			// Keep special SQLite DSNs untouched.
		case !filepath.IsAbs(dsn):
			cfg.Database.DSN = filepath.Join(cfg.DataDir, dsn)
		}
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

func findConfigFile(dataDir string) (string, error) {
	if path := strings.TrimSpace(os.Getenv("RESUME_HELPER_CONFIG")); path != "" {
		abs, err := filepath.Abs(path)
		if err != nil {
			return "", fmt.Errorf("resolve config path %s: %w", path, err)
		}
		if _, err := os.Stat(abs); err != nil {
			return "", fmt.Errorf("config file %s: %w", abs, err)
		}
		return abs, nil
	}
	path := filepath.Join(dataDir, "config.yaml")
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}
	return "", nil
}

// dataDirectory returns the portable application directory. The executable's
// folder is used for packaged builds; `go run` uses the current directory so
// development files stay in the project instead of a temporary build folder.
func dataDirectory() (string, error) {
	if value := strings.TrimSpace(os.Getenv("RESUME_HELPER_DATA_DIR")); value != "" {
		dir, err := filepath.Abs(value)
		if err != nil {
			return "", fmt.Errorf("resolve data directory %s: %w", value, err)
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", fmt.Errorf("create data directory %s: %w", dir, err)
		}
		return dir, nil
	}

	if executable, err := os.Executable(); err == nil {
		executable = filepath.Clean(executable)
		if !isDevelopmentExecutable(executable) {
			return filepath.Dir(executable), nil
		}
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("locate working directory: %w", err)
	}
	return cwd, nil
}

func isDevelopmentExecutable(path string) bool {
	lower := strings.ToLower(path)
	if strings.Contains(lower, "go-build") {
		return true
	}
	dev := strings.ToLower(strings.TrimSpace(os.Getenv("RESUME_HELPER_DEV")))
	return dev == "1" || dev == "true"
}

func defaultConfigFile(_ string) string {
	return `# ResumeTrail portable configuration
# This file lives next to the application. Keep it there so all data stays
# inside the application folder and nothing is written to AppData or other
# system directories.
#
# Edit this file and restart the application for changes to take effect.
# The database password is never stored here; application passwords are
# hashed with bcrypt inside the database.

server:
  host: 127.0.0.1
  port: 19876

database:
  # sqlite | mysql | postgres
  driver: sqlite
  # SQLite: empty means resume-helper.db next to this file. Relative paths
  # are also resolved next to this config file.
  # Default: resume-helper.db
  # MySQL example:
  #   dsn: "user:password@tcp(127.0.0.1:3306)/resume_helper?charset=utf8mb4&parseTime=True&loc=Local"
  # PostgreSQL example:
  #   dsn: "host=127.0.0.1 user=postgres password=secret dbname=resume_helper port=5432 sslmode=disable TimeZone=Local"
  dsn: ""
`
}
