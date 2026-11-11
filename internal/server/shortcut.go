package server

import (
	"net/http"

	"resume-helper/internal/desktop"

	"github.com/gin-gonic/gin"
)

func (s *Server) getShortcutStatus(c *gin.Context) {
	supported := desktop.ShortcutSupported()
	path := ""
	exists := false
	if supported {
		var err error
		path, err = desktop.ShortcutPath()
		if err != nil {
			apiError(c, http.StatusInternalServerError, "shortcut_failed", err.Error())
			return
		}
		exists, err = desktop.ShortcutExists()
		if err != nil {
			apiError(c, http.StatusInternalServerError, "shortcut_failed", err.Error())
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"supported": supported,
		"exists":    exists,
		"path":      path,
	})
}

func (s *Server) resolveShortcutPrompt(c *gin.Context) {
	var input struct {
		Create bool `json:"create"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		apiError(c, http.StatusBadRequest, "invalid_body", "Invalid request body.")
		return
	}

	s.shortcutPromptMu.RLock()
	handler := s.shortcutPromptHandler
	s.shortcutPromptMu.RUnlock()
	if handler == nil {
		apiError(c, http.StatusServiceUnavailable, "desktop_only", "Shortcut prompt is only available in desktop mode.")
		return
	}
	if err := handler(input.Create); err != nil {
		apiError(c, http.StatusInternalServerError, "shortcut_failed", err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"resolved": true})
}

func (s *Server) createDesktopShortcut(c *gin.Context) {
	if !desktop.ShortcutSupported() {
		apiError(c, http.StatusBadRequest, "shortcut_unsupported", "Desktop shortcuts are only supported on Windows.")
		return
	}
	if err := desktop.CreateShortcut(); err != nil {
		apiError(c, http.StatusInternalServerError, "shortcut_failed", err.Error())
		return
	}
	path, _ := desktop.ShortcutPath()
	c.JSON(http.StatusOK, gin.H{
		"supported": true,
		"exists":    true,
		"path":      path,
	})
}
