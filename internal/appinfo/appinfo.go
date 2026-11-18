// Package appinfo exposes the application metadata (version, name, title and
// default language) that can be configured through the portable .env file.
package appinfo

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	fallbackVersion  = "1.2"
	fallbackName     = "ResumeTrail"
	fallbackTitle    = "履途 ResumeTrail"
	fallbackLanguage = "zh-CN"
)

var (
	// Version is the application version shown in the UI, the health endpoint
	// and backup files.
	Version = fallbackVersion
	// Name is the English application name used for process and window
	// identifiers.
	Name = fallbackName
	// Title is the display name used for window titles, the tray menu and the
	// desktop shortcut file name.
	Title = fallbackTitle
	// DefaultLanguage is the locale used when the browser offers no supported
	// language and the user has not picked one yet.
	DefaultLanguage = fallbackLanguage
)

// Load reads .env from dir (or from the path in APP_ENV_FILE) and refreshes the
// package-level metadata. Variables that already exist in the process
// environment take precedence over the file.
//
// A missing .env file is not an error; the built-in defaults are kept.
func Load(dir string) error {
	path := strings.TrimSpace(os.Getenv("APP_ENV_FILE"))
	if path == "" {
		path = filepath.Join(dir, ".env")
	}
	if err := loadFile(path); err != nil {
		return err
	}

	Version = envOrDefault("APP_VERSION", fallbackVersion)
	Name = envOrDefault("APP_NAME", fallbackName)
	Title = envOrDefault("APP_TITLE", fallbackTitle)
	DefaultLanguage = normalizeLanguage(envOrDefault("APP_DEFAULT_LANGUAGE", fallbackLanguage))
	return nil
}

func loadFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read %s: %w", path, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		// Never overwrite values supplied by the real environment.
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		if err := os.Setenv(key, parseValue(strings.TrimSpace(value))); err != nil {
			return fmt.Errorf("apply %s from %s: %w", key, path, err)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	return nil
}

func parseValue(value string) string {
	if len(value) >= 2 {
		first, last := value[0], value[len(value)-1]
		if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
			return value[1 : len(value)-1]
		}
	}
	if index := strings.Index(value, " #"); index >= 0 {
		value = value[:index]
	}
	return strings.TrimSpace(value)
}

func envOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func normalizeLanguage(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "zh", "zh-cn", "zh-hans", "zh-sg":
		return "zh-CN"
	case "zh-tw", "zh-hant", "zh-hk", "zh-mo":
		return "zh-TW"
	case "en", "en-us", "en-gb":
		return "en"
	case "ja", "ja-jp":
		return "ja"
	case "ko", "ko-kr":
		return "ko"
	default:
		return fallbackLanguage
	}
}
