package database

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5"
)

// Settings is the graphical, driver-agnostic representation of a database
// connection used by the settings page and the test-connection endpoint.
type Settings struct {
	Driver string `json:"driver"`

	// DSN is the raw connection string. It is returned by the settings
	// endpoint as a fallback when a previously saved DSN cannot be parsed
	// into the structured fields below.
	DSN string `json:"dsn"`

	SQLitePath string `json:"sqlitePath"`
	Host       string `json:"host"`
	Port       int    `json:"port"`
	User       string `json:"user"`
	Password   string `json:"password"`
	Database   string `json:"database"`
	SSLMode    string `json:"sslMode"`

	ParseError string `json:"parseError,omitempty"`
}

// ParseSettings converts a DSN into editable fields. When parsing fails the
// raw DSN is still returned so the UI can offer an advanced editor instead of
// silently discarding the user's configuration.
func ParseSettings(driver, dsn, dataDir string) Settings {
	normalized, err := normalizeDriver(driver)
	if err != nil {
		return Settings{Driver: driver, DSN: dsn, ParseError: err.Error()}
	}
	settings := Settings{Driver: normalized}
	raw := strings.TrimSpace(dsn)

	switch normalized {
	case "sqlite":
		settings.DSN = raw
		settings.SQLitePath = displaySQLitePath(raw, dataDir)
	case "mysql":
		settings.DSN = raw
		cfg, err := mysqldriver.ParseDSN(raw)
		if err != nil {
			settings.ParseError = err.Error()
			return settings
		}
		host, port := splitAddress(cfg.Addr, "127.0.0.1", 3306)
		settings.Host = host
		settings.Port = port
		settings.User = cfg.User
		settings.Password = cfg.Passwd
		settings.Database = cfg.DBName
	case "postgres":
		settings.DSN = raw
		cfg, err := pgx.ParseConfig(raw)
		if err != nil {
			settings.ParseError = err.Error()
			return settings
		}
		settings.Host = cfg.Host
		settings.Port = int(cfg.Port)
		settings.User = cfg.User
		settings.Password = cfg.Password
		settings.Database = cfg.Database
		settings.SSLMode = postgresSSLMode(raw, cfg)
		if settings.Host == "" {
			settings.Host = "127.0.0.1"
		}
		if settings.Port == 0 {
			settings.Port = 5432
		}
		if settings.SSLMode == "" {
			settings.SSLMode = "disable"
		}
	}
	return settings
}

// BuildDSN converts the structured settings back into a driver DSN.
func BuildDSN(settings Settings) (string, error) {
	normalized, err := normalizeDriver(settings.Driver)
	if err != nil {
		return "", err
	}

	switch normalized {
	case "sqlite":
		return strings.TrimSpace(settings.SQLitePath), nil
	case "mysql":
		host := strings.TrimSpace(settings.Host)
		if host == "" {
			host = "127.0.0.1"
		}
		port := settings.Port
		if port == 0 {
			port = 3306
		}
		if port < 1 || port > 65535 {
			return "", fmt.Errorf("MySQL port must be between 1 and 65535")
		}
		name := strings.TrimSpace(settings.Database)
		if name == "" {
			return "", fmt.Errorf("MySQL database name is required")
		}
		cfg := mysqldriver.NewConfig()
		cfg.User = settings.User
		cfg.Passwd = settings.Password
		cfg.Net = "tcp"
		cfg.Addr = net.JoinHostPort(host, strconv.Itoa(port))
		cfg.DBName = name
		cfg.ParseTime = true
		cfg.Loc = time.Local
		cfg.Timeout = 5 * time.Second
		cfg.Params = map[string]string{"charset": "utf8mb4"}
		return cfg.FormatDSN(), nil
	case "postgres":
		host := strings.TrimSpace(settings.Host)
		if host == "" {
			host = "127.0.0.1"
		}
		port := settings.Port
		if port == 0 {
			port = 5432
		}
		if port < 1 || port > 65535 {
			return "", fmt.Errorf("PostgreSQL port must be between 1 and 65535")
		}
		name := strings.TrimSpace(settings.Database)
		if name == "" {
			return "", fmt.Errorf("PostgreSQL database name is required")
		}
		sslMode := strings.TrimSpace(settings.SSLMode)
		if sslMode == "" {
			sslMode = "disable"
		}
		u := &url.URL{
			Scheme: "postgres",
			Host:   net.JoinHostPort(host, strconv.Itoa(port)),
			Path:   "/" + name,
		}
		if settings.Password != "" {
			u.User = url.UserPassword(settings.User, settings.Password)
		} else {
			u.User = url.User(settings.User)
		}
		query := u.Query()
		query.Set("sslmode", sslMode)
		query.Set("connect_timeout", "5")
		query.Set("TimeZone", "Local")
		u.RawQuery = query.Encode()
		return u.String(), nil
	default:
		return "", fmt.Errorf("unsupported database driver: %s", settings.Driver)
	}
}

func normalizeDriver(driver string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(driver)) {
	case "", "sqlite", "sqlite3":
		return "sqlite", nil
	case "postgresql", "pg", "postgres":
		return "postgres", nil
	case "mysql":
		return "mysql", nil
	default:
		return "", fmt.Errorf("unsupported database driver: %s", driver)
	}
}

func splitAddress(address, fallbackHost string, fallbackPort int) (string, int) {
	host, portText, err := net.SplitHostPort(strings.TrimSpace(address))
	if err != nil {
		if strings.TrimSpace(address) == "" {
			return fallbackHost, fallbackPort
		}
		return strings.TrimSpace(address), fallbackPort
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port == 0 {
		port = fallbackPort
	}
	if host == "" {
		host = fallbackHost
	}
	return host, port
}

func displaySQLitePath(dsn, dataDir string) string {
	if dsn == "" || dsn == ":memory:" || strings.HasPrefix(dsn, "file:") {
		return dsn
	}
	if !filepath.IsAbs(dsn) {
		return dsn
	}
	if rel, err := filepath.Rel(dataDir, dsn); err == nil &&
		rel != "." && rel != ".." &&
		!strings.HasPrefix(rel, ".."+string(os.PathSeparator)) &&
		!filepath.IsAbs(rel) {
		return rel
	}
	return dsn
}

func postgresSSLMode(dsn string, cfg *pgx.ConnConfig) string {
	if parsed, err := url.Parse(dsn); err == nil && parsed.Scheme != "" {
		if value := strings.TrimSpace(parsed.Query().Get("sslmode")); value != "" {
			return value
		}
	}
	// Keyword/value DSN: sslmode is not exposed through pgx.ConnConfig.
	for _, field := range strings.Fields(dsn) {
		key, value, ok := strings.Cut(field, "=")
		if ok && strings.EqualFold(strings.TrimSpace(key), "sslmode") {
			return strings.Trim(strings.TrimSpace(value), `'"`)
		}
	}
	if cfg.TLSConfig != nil {
		return "prefer"
	}
	return "disable"
}
