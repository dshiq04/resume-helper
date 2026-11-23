package server

import (
	"net/http"
	"testing"

	"resume-helper/internal/appinfo"
)

func TestAppInfoHandshakeUpdatesVersion(t *testing.T) {
	app := newTestApp(t)
	before := appinfo.Get()
	t.Cleanup(func() {
		appinfo.Apply(before)
	})

	resp, body := app.request(t, http.MethodPost, "/api/app-info", map[string]string{
		"version":         "9.9",
		"name":            "TestApp",
		"title":           "Test Title",
		"defaultLanguage": "en",
	}, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("app-info status = %d, body = %s", resp.StatusCode, body)
	}

	_, body = app.request(t, http.MethodGet, "/api/health", nil, "")
	health := decodeJSON[struct {
		Version string `json:"version"`
	}](t, body)
	if health.Version != "9.9" {
		t.Fatalf("health version = %q, want 9.9", health.Version)
	}
}
