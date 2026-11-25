package database

import (
	"os"
	"path/filepath"
	"strings"

	"resume-helper/internal/config"
)

// Settings is the graphical representation of the SQLite storage location.
// SQLite is the only supported database backend.
type Settings struct {
	Driver     string `json:"driver"`
	SQLitePath string `json:"sqlitePath"`

	ParseError string `json:"parseError,omitempty"`
}

// ParseSettings converts the stored SQLite path into the editable field shown
// by the settings page.
func ParseSettings(driver, dsn, dataDir string) Settings {
	normalized, err := config.NormalizeDriver(driver)
	if err != nil {
		return Settings{Driver: driver, ParseError: err.Error()}
	}
	raw := strings.TrimSpace(dsn)
	return Settings{
		Driver:     normalized,
		SQLitePath: displaySQLitePath(raw, dataDir),
	}
}

// BuildDSN converts the structured settings back into the SQLite file path.
func BuildDSN(settings Settings) (string, error) {
	if _, err := config.NormalizeDriver(settings.Driver); err != nil {
		return "", err
	}
	return strings.TrimSpace(settings.SQLitePath), nil
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
