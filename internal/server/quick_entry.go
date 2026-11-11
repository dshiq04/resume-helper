package server

import (
	"net/http"
	"strings"

	"resume-helper/internal/models"

	"github.com/gin-gonic/gin"
)

func (s *Server) openQuickEntry(c *gin.Context) {
	var input struct {
		Category string `json:"category"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		apiError(c, http.StatusBadRequest, "invalid_body", "Invalid request body.")
		return
	}

	category := strings.ToLower(strings.TrimSpace(input.Category))
	if category == "" {
		category = "all"
	}
	if category != "all" {
		category = models.NormalizeCategory(category)
		if !models.IsValidCategory(category) {
			apiError(c, http.StatusBadRequest, "invalid_category", "Unknown application category.")
			return
		}
	}

	s.quickEntryMu.RLock()
	opener := s.quickEntryOpener
	s.quickEntryMu.RUnlock()
	if opener == nil {
		apiError(c, http.StatusServiceUnavailable, "desktop_only", "Quick entry window is only available in desktop mode.")
		return
	}

	opener(category)
	c.JSON(http.StatusOK, gin.H{"opened": true})
}
