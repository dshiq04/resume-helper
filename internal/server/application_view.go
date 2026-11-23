package server

import (
	"net/http"

	"resume-helper/internal/models"

	"github.com/gin-gonic/gin"
)

// openApplicationView opens the read-only detail window. The window loads the
// record and its status timeline through the regular authenticated API.
func (s *Server) openApplicationView(c *gin.Context) {
	var input struct {
		ID uint `json:"id"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || input.ID == 0 {
		apiError(c, http.StatusBadRequest, "invalid_id", "Invalid record id.")
		return
	}

	var record models.Application
	if err := s.database().Select("id").First(&record, input.ID).Error; err != nil {
		if isRecordNotFound(err) {
			fail(c, http.StatusNotFound, "not_found", "原内容不存在")
			return
		}
		apiError(c, http.StatusInternalServerError, "query_failed", "Failed to load the application.")
		return
	}

	s.applicationViewMu.RLock()
	opener := s.applicationViewOpener
	s.applicationViewMu.RUnlock()
	if opener == nil {
		apiError(c, http.StatusServiceUnavailable, "desktop_only", "The detail window is only available in desktop mode.")
		return
	}
	opener(input.ID)
	c.JSON(http.StatusOK, gin.H{"opened": true})
}
