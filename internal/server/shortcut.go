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
