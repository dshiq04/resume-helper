package server

import (
	"net/http"
	"strings"

	"resume-helper/internal/models"

	"github.com/gin-gonic/gin"
)

// companySuggestions returns distinct historical company names. GROUP BY is
// used so the database performs the de-duplication instead of the client.
func (s *Server) companySuggestions(c *gin.Context) {
	search := strings.ToLower(strings.TrimSpace(c.Query("search")))
	query := s.database().Model(&models.Application{}).
		Select("company_name").
		Group("company_name").
		Order("company_name ASC").
		Limit(20)
	if search != "" {
		query = query.Where("LOWER(company_name) LIKE ?", "%"+search+"%")
	}

	names := []string{}
	if err := query.Pluck("company_name", &names).Error; err != nil {
		apiError(c, http.StatusInternalServerError, "query_failed", "Failed to load company suggestions.")
		return
	}
	if names == nil {
		names = []string{}
	}
	c.JSON(http.StatusOK, gin.H{"items": names})
}
