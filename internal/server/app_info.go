package server

import (
	"net/http"

	"resume-helper/internal/appinfo"

	"github.com/gin-gonic/gin"
)

// SetAppInfoHandler registers a callback used to refresh desktop UI that was
// created before the frontend reported its metadata.
func (s *Server) SetAppInfoHandler(handler func()) {
	s.appInfoMu.Lock()
	s.appInfoHandler = handler
	s.appInfoMu.Unlock()
}

func (s *Server) updateAppInfo(c *gin.Context) {
	var input appinfo.Info
	if err := c.ShouldBindJSON(&input); err != nil {
		apiError(c, http.StatusBadRequest, "invalid_body", "Invalid request body.")
		return
	}

	appinfo.Apply(input)

	s.appInfoMu.RLock()
	handler := s.appInfoHandler
	s.appInfoMu.RUnlock()
	if handler != nil {
		handler()
	}

	c.JSON(http.StatusOK, gin.H{"applied": true})
}
