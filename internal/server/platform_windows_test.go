//go:build windows

package server

import (
	"strings"
	"testing"
)

func TestCurrentPlatformWindows(t *testing.T) {
	info := currentPlatform()
	t.Logf("platform: %+v", info)
	if info.Kind != "computer" {
		t.Fatalf("kind = %q, want computer", info.Kind)
	}
	if info.OS != "windows" {
		t.Fatalf("os = %q, want windows", info.OS)
	}
	if !strings.HasPrefix(info.Name, "Windows") {
		t.Fatalf("name = %q, want a Windows product name", info.Name)
	}
	if !strings.Contains(info.Version, ".") {
		t.Fatalf("version = %q, want a Windows build number", info.Version)
	}
}
