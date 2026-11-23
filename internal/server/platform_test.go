package server

import (
	"net/http"
	"testing"
)

func TestPlatformEndpoint(t *testing.T) {
	app := newTestApp(t)

	resp, body := app.request(t, http.MethodGet, "/api/system/platform", nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("platform status = %d, body = %s", resp.StatusCode, body)
	}

	info := decodeJSON[PlatformInfo](t, body)
	if info.Kind != "computer" && info.Kind != "tablet" && info.Kind != "phone" {
		t.Fatalf("kind = %q, want computer, tablet or phone", info.Kind)
	}
	if info.OS == "" || info.Name == "" {
		t.Fatalf("platform = %+v, want an os and name", info)
	}
}
