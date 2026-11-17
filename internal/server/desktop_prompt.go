package server

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func (s *Server) openDataTransfer(c *gin.Context) {
	var input struct {
		Mode string `json:"mode"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		apiError(c, http.StatusBadRequest, "invalid_body", "Invalid request body.")
		return
	}
	mode := strings.ToLower(strings.TrimSpace(input.Mode))
	if mode != "import" && mode != "export" {
		mode = "export"
	}

	s.dataTransferMu.RLock()
	opener := s.dataTransferOpener
	s.dataTransferMu.RUnlock()
	if opener == nil {
		apiError(c, http.StatusServiceUnavailable, "desktop_only", "The data transfer window is only available in desktop mode.")
		return
	}
	opener(mode)
	c.JSON(http.StatusOK, gin.H{"opened": true})
}

func (s *Server) resolveClosePrompt(c *gin.Context) {
	var input struct {
		Action   string `json:"action"`
		Remember bool   `json:"remember"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		apiError(c, http.StatusBadRequest, "invalid_body", "Invalid request body.")
		return
	}
	action := strings.ToLower(strings.TrimSpace(input.Action))
	if action != CloseActionTray && action != CloseActionQuit {
		apiError(c, http.StatusBadRequest, "invalid_action", "Unknown close action.")
		return
	}
	if input.Remember {
		if err := s.SetCloseAction(action); err != nil {
			apiError(c, http.StatusInternalServerError, "save_failed", "Failed to save the close behaviour.")
			return
		}
	}

	s.closePromptMu.RLock()
	handler := s.closePromptHandler
	s.closePromptMu.RUnlock()
	if handler == nil {
		apiError(c, http.StatusServiceUnavailable, "desktop_only", "The close prompt is only available in desktop mode.")
		return
	}
	if err := handler(action, input.Remember); err != nil {
		apiError(c, http.StatusInternalServerError, "close_failed", err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"resolved": true})
}
