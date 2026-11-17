package server

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"resume-helper/internal/models"

	"github.com/gin-gonic/gin"
)

const (
	closeActionKey     = "desktop.close_action"
	idleLockEnabledKey = "security.idle_lock_enabled"
	idleLockMinutesKey = "security.idle_lock_minutes"
	defaultIdleMinutes = 15
	minIdleMinutes     = 1
	maxIdleMinutes     = 240
	CloseActionAsk     = "ask"
	CloseActionTray    = "tray"
	CloseActionQuit    = "quit"
)

type preferencesResponse struct {
	CloseAction           string `json:"closeAction"`
	IdleLockEnabled       bool   `json:"idleLockEnabled"`
	IdleLockMinutes       int    `json:"idleLockMinutes"`
	ShortcutPromptEnabled bool   `json:"shortcutPromptEnabled"`
}

func (s *Server) getSetting(key string) (string, bool) {
	var setting models.Setting
	if err := s.database().First(&setting, "key = ?", key).Error; err != nil {
		return "", false
	}
	return setting.Value, true
}

func (s *Server) saveSetting(key, value string) error {
	return s.database().Save(&models.Setting{Key: key, Value: value}).Error
}

// CloseAction reports what closing the main window should do. Unknown values
// fall back to asking the user.
func (s *Server) CloseAction() string {
	value, ok := s.getSetting(closeActionKey)
	if !ok {
		return CloseActionAsk
	}
	switch strings.ToLower(strings.TrimSpace(value)) {
	case CloseActionTray:
		return CloseActionTray
	case CloseActionQuit:
		return CloseActionQuit
	default:
		return CloseActionAsk
	}
}

func (s *Server) SetCloseAction(action string) error {
	switch action {
	case CloseActionAsk, CloseActionTray, CloseActionQuit:
	default:
		return errors.New("unknown close action")
	}
	return s.saveSetting(closeActionKey, action)
}

func (s *Server) IdleLockEnabled() bool {
	value, ok := s.getSetting(idleLockEnabledKey)
	if !ok {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "true", "1", "on", "yes":
		return true
	default:
		return false
	}
}

func (s *Server) IdleLockMinutes() int {
	value, ok := s.getSetting(idleLockMinutesKey)
	if !ok {
		return defaultIdleMinutes
	}
	minutes, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || minutes < minIdleMinutes || minutes > maxIdleMinutes {
		return defaultIdleMinutes
	}
	return minutes
}

func (s *Server) preferences() preferencesResponse {
	return preferencesResponse{
		CloseAction:           s.CloseAction(),
		IdleLockEnabled:       s.IdleLockEnabled(),
		IdleLockMinutes:       s.IdleLockMinutes(),
		ShortcutPromptEnabled: s.ShortcutPromptEnabled(),
	}
}

func (s *Server) getPreferences(c *gin.Context) {
	c.JSON(http.StatusOK, s.preferences())
}

func (s *Server) updatePreferences(c *gin.Context) {
	var input struct {
		CloseAction     *string `json:"closeAction"`
		IdleLockEnabled *bool   `json:"idleLockEnabled"`
		IdleLockMinutes *int    `json:"idleLockMinutes"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		apiError(c, http.StatusBadRequest, "invalid_body", "Invalid request body.")
		return
	}

	if input.CloseAction != nil {
		if err := s.SetCloseAction(*input.CloseAction); err != nil {
			apiError(c, http.StatusBadRequest, "invalid_close_action", "Unknown close action.")
			return
		}
	}
	if input.IdleLockMinutes != nil {
		minutes := *input.IdleLockMinutes
		if minutes < minIdleMinutes || minutes > maxIdleMinutes {
			apiError(c, http.StatusBadRequest, "invalid_idle_minutes", "Idle minutes must be between 1 and 240.")
			return
		}
		if err := s.saveSetting(idleLockMinutesKey, strconv.Itoa(minutes)); err != nil {
			apiError(c, http.StatusInternalServerError, "save_failed", "Failed to save the idle lock setting.")
			return
		}
	}
	if input.IdleLockEnabled != nil {
		value := "false"
		if *input.IdleLockEnabled {
			value = "true"
		}
		if err := s.saveSetting(idleLockEnabledKey, value); err != nil {
			apiError(c, http.StatusInternalServerError, "save_failed", "Failed to save the idle lock setting.")
			return
		}
	}

	c.JSON(http.StatusOK, s.preferences())
}

func (s *Server) removeSetting(key string) error {
	return s.database().Where("key = ?", key).Delete(&models.Setting{}).Error
}
