package server

import (
	"errors"
	"net/http"
	"strings"

	"resume-helper/internal/desktop"
	"resume-helper/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const shortcutPromptEnabledKey = "shortcut.prompt_enabled"

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
		"supported":     supported,
		"exists":        exists,
		"path":          path,
		"promptEnabled": s.ShortcutPromptEnabled(),
	})
}

// ShortcutPromptEnabled reports whether closing the main window should ask the
// user to create a desktop shortcut. The default is enabled.
func (s *Server) ShortcutPromptEnabled() bool {
	var setting models.Setting
	err := s.database().First(&setting, "key = ?", shortcutPromptEnabledKey).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return true
	}
	if err != nil {
		return true
	}
	value := strings.ToLower(strings.TrimSpace(setting.Value))
	return value != "false" && value != "0" && value != "off"
}

func (s *Server) SetShortcutPromptEnabled(enabled bool) error {
	value := "true"
	if !enabled {
		value = "false"
	}
	return s.database().Save(&models.Setting{
		Key:   shortcutPromptEnabledKey,
		Value: value,
	}).Error
}

func (s *Server) updateShortcutPromptSetting(c *gin.Context) {
	var input struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		apiError(c, http.StatusBadRequest, "invalid_body", "Invalid request body.")
		return
	}
	if err := s.SetShortcutPromptEnabled(input.Enabled); err != nil {
		apiError(c, http.StatusInternalServerError, "shortcut_failed", "Failed to save the shortcut prompt setting.")
		return
	}
	c.JSON(http.StatusOK, gin.H{"promptEnabled": input.Enabled})
}

func (s *Server) resolveShortcutPrompt(c *gin.Context) {
	var input struct {
		Create   bool `json:"create"`
		Remember bool `json:"remember"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		apiError(c, http.StatusBadRequest, "invalid_body", "Invalid request body.")
		return
	}

	if input.Remember {
		if err := s.SetShortcutPromptEnabled(false); err != nil {
			apiError(c, http.StatusInternalServerError, "shortcut_failed", "Failed to save the shortcut prompt setting.")
			return
		}
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
