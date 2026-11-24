// Package appinfo holds the application metadata (version, name, title and
// default language). The values are supplied by the frontend at runtime from
// frontend/.env, so the build output never needs a .env file.
package appinfo

import (
	"strings"
	"sync/atomic"
)

const (
	fallbackVersion  = "1.2.1"
	fallbackName     = "ResumeTrail"
	fallbackTitle    = "履途 ResumeTrail"
	fallbackLanguage = "zh-CN"
)

// Info is the metadata payload sent by the frontend after it loads.
type Info struct {
	Version         string `json:"version"`
	Name            string `json:"name"`
	Title           string `json:"title"`
	DefaultLanguage string `json:"defaultLanguage"`
}

var current atomic.Pointer[Info]

func init() {
	current.Store(&Info{
		Version:         fallbackVersion,
		Name:            fallbackName,
		Title:           fallbackTitle,
		DefaultLanguage: fallbackLanguage,
	})
}

// Apply merges the metadata sent by the frontend into the current values.
// Empty fields keep their current value so a partial handshake is harmless.
func Apply(info Info) {
	next := *current.Load()
	if value := strings.TrimSpace(info.Version); value != "" {
		next.Version = value
	}
	if value := strings.TrimSpace(info.Name); value != "" {
		next.Name = value
	}
	if value := strings.TrimSpace(info.Title); value != "" {
		next.Title = value
	}
	if value := strings.TrimSpace(info.DefaultLanguage); value != "" {
		next.DefaultLanguage = normalizeLanguage(value)
	}
	current.Store(&next)
}

// Get returns a snapshot of the current metadata.
func Get() Info {
	return *current.Load()
}

func Version() string {
	return Get().Version
}

func Name() string {
	return Get().Name
}

func Title() string {
	return Get().Title
}

func DefaultLanguage() string {
	return Get().DefaultLanguage
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
