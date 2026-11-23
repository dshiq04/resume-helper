package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// PlatformInfo describes where the application is running. The about card
// shows the form factor and the operating system with its concrete version so
// support reports can identify the exact environment.
type PlatformInfo struct {
	Kind    string `json:"kind"`    // computer, tablet or phone
	OS      string `json:"os"`      // windows, linux, macos, android, ios or ipados
	Name    string `json:"name"`    // operating system name, e.g. Windows 11 Pro
	Version string `json:"version"` // version, build or distribution, e.g. 24H2 (10.0.22631.4317)
}

// currentPlatform normalises the platform specific detection so the API always
// returns a complete record.
func currentPlatform() PlatformInfo {
	info := detectPlatform()
	if info.Kind == "" {
		info.Kind = "computer"
	}
	if info.OS == "" {
		info.OS = "unknown"
	}
	if info.Name == "" {
		info.Name = info.OS
	}
	return info
}

// platformInfo returns the current form factor and operating system as JSON.
func (s *Server) platformInfo(c *gin.Context) {
	c.JSON(http.StatusOK, currentPlatform())
}
