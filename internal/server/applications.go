package server

import (
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"resume-helper/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type listApplicationsParams struct {
	Search   string `form:"search"`
	Category string `form:"category"`
	Status   string `form:"status"`
	Page     int    `form:"page"`
	PageSize int    `form:"pageSize"`
}

type applicationInput struct {
	Category    string `json:"category"`
	CompanyName string `json:"companyName"`
	Link        string `json:"link"`
	Status      string `json:"status"`
	Pinned      bool   `json:"pinned"`
}

type applicationUpdateInput struct {
	Category    *string `json:"category"`
	CompanyName *string `json:"companyName"`
	Link        *string `json:"link"`
	Status      *string `json:"status"`
	Pinned      *bool   `json:"pinned"`
}

type idsInput struct {
	IDs []uint `json:"ids"`
}

type batchStatusInput struct {
	IDs    []uint `json:"ids"`
	Status string `json:"status"`
}

func (s *Server) listApplications(c *gin.Context) {
	var params listApplicationsParams
	if err := c.ShouldBindQuery(&params); err != nil {
		apiError(c, http.StatusBadRequest, "invalid_query", "Invalid query parameters.")
		return
	}

	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 {
		params.PageSize = 10
	}
	if params.PageSize > 100 {
		params.PageSize = 100
	}

	category := models.NormalizeCategory(params.Category)
	status := models.NormalizeStatus(params.Status)
	if category == "all" {
		category = ""
	}
	if status == "all" {
		status = ""
	}
	if category != "" && !models.IsValidCategory(category) {
		apiError(c, http.StatusBadRequest, "invalid_category", "Unknown application category.")
		return
	}
	if status != "" && !models.IsValidStatus(status) {
		apiError(c, http.StatusBadRequest, "invalid_status", "Unknown application status.")
		return
	}

	db := s.database()
	build := func() *gorm.DB {
		query := db.Model(&models.Application{})
		if category != "" {
			query = query.Where("category = ?", category)
		}
		if status != "" {
			query = query.Where("status = ?", status)
		}
		if search := strings.TrimSpace(params.Search); search != "" {
			like := "%" + strings.ToLower(search) + "%"
			query = query.Where("LOWER(company_name) LIKE ? OR LOWER(link) LIKE ?", like, like)
		}
		return query
	}

	var total int64
	if err := build().Count(&total).Error; err != nil {
		apiError(c, http.StatusInternalServerError, "query_failed", "Failed to load applications.")
		return
	}

	var items []models.Application
	if err := build().
		Order("pinned DESC, updated_at DESC, id DESC").
		Offset((params.Page - 1) * params.PageSize).
		Limit(params.PageSize).
		Find(&items).Error; err != nil {
		apiError(c, http.StatusInternalServerError, "query_failed", "Failed to load applications.")
		return
	}
	if items == nil {
		items = []models.Application{}
	}

	c.JSON(http.StatusOK, gin.H{
		"items":    items,
		"total":    total,
		"page":     params.Page,
		"pageSize": params.PageSize,
	})
}

func (s *Server) createApplication(c *gin.Context) {
	var input applicationInput
	if err := c.ShouldBindJSON(&input); err != nil {
		apiError(c, http.StatusBadRequest, "invalid_body", "Invalid request body.")
		return
	}

	record, code, message := validateApplicationInput(input)
	if code != "" {
		apiError(c, http.StatusBadRequest, code, message)
		return
	}
	// The application timestamp always comes from the server clock, never
	// from the client payload.
	record.AppliedAt = time.Now()

	if err := s.database().Create(record).Error; err != nil {
		apiError(c, http.StatusInternalServerError, "create_failed", "Failed to create the application.")
		return
	}
	c.JSON(http.StatusCreated, record)
}

func (s *Server) updateApplication(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	db := s.database()
	var record models.Application
	if err := db.First(&record, id).Error; err != nil {
		if isRecordNotFound(err) {
			apiError(c, http.StatusNotFound, "not_found", "Application not found.")
			return
		}
		apiError(c, http.StatusInternalServerError, "query_failed", "Failed to load the application.")
		return
	}

	var input applicationUpdateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		apiError(c, http.StatusBadRequest, "invalid_body", "Invalid request body.")
		return
	}

	updates := map[string]interface{}{}

	if input.Category != nil {
		category := models.NormalizeCategory(*input.Category)
		if !models.IsValidCategory(category) {
			apiError(c, http.StatusBadRequest, "invalid_category", "Unknown application category.")
			return
		}
		updates["category"] = category
	}
	if input.CompanyName != nil {
		company := strings.TrimSpace(*input.CompanyName)
		if company == "" {
			apiError(c, http.StatusBadRequest, "company_required", "Company name is required.")
			return
		}
		if utf8.RuneCountInString(company) > 255 {
			apiError(c, http.StatusBadRequest, "company_too_long", "Company name is too long.")
			return
		}
		updates["company_name"] = company
	}
	if input.Link != nil {
		link := normalizeLink(*input.Link)
		if len(link) > 2048 {
			apiError(c, http.StatusBadRequest, "link_too_long", "Application link is too long.")
			return
		}
		updates["link"] = link
	}
	if input.Status != nil {
		status := models.NormalizeStatus(*input.Status)
		if !models.IsValidStatus(status) {
			apiError(c, http.StatusBadRequest, "invalid_status", "Unknown application status.")
			return
		}
		updates["status"] = status
	}
	if input.Pinned != nil {
		updates["pinned"] = *input.Pinned
	}

	if len(updates) > 0 {
		if err := db.Model(&record).Updates(updates).Error; err != nil {
			apiError(c, http.StatusInternalServerError, "update_failed", "Failed to update the application.")
			return
		}
	}

	if err := db.First(&record, id).Error; err != nil {
		apiError(c, http.StatusInternalServerError, "query_failed", "Failed to reload the application.")
		return
	}
	c.JSON(http.StatusOK, record)
}

func (s *Server) deleteApplication(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	result := s.database().Delete(&models.Application{}, id)
	if result.Error != nil {
		apiError(c, http.StatusInternalServerError, "delete_failed", "Failed to delete the application.")
		return
	}
	if result.RowsAffected == 0 {
		apiError(c, http.StatusNotFound, "not_found", "Application not found.")
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": result.RowsAffected})
}

func (s *Server) batchDeleteApplications(c *gin.Context) {
	var input idsInput
	if err := c.ShouldBindJSON(&input); err != nil {
		apiError(c, http.StatusBadRequest, "invalid_body", "Invalid request body.")
		return
	}
	ids := uniqueIDs(input.IDs)
	if len(ids) == 0 {
		apiError(c, http.StatusBadRequest, "ids_required", "Select at least one application.")
		return
	}
	if len(ids) > 2000 {
		apiError(c, http.StatusBadRequest, "too_many_ids", "Too many applications selected.")
		return
	}

	result := s.database().Where("id IN ?", ids).Delete(&models.Application{})
	if result.Error != nil {
		apiError(c, http.StatusInternalServerError, "delete_failed", "Failed to delete the selected applications.")
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": result.RowsAffected})
}

func (s *Server) batchUpdateStatus(c *gin.Context) {
	var input batchStatusInput
	if err := c.ShouldBindJSON(&input); err != nil {
		apiError(c, http.StatusBadRequest, "invalid_body", "Invalid request body.")
		return
	}
	ids := uniqueIDs(input.IDs)
	if len(ids) == 0 {
		apiError(c, http.StatusBadRequest, "ids_required", "Select at least one application.")
		return
	}
	status := models.NormalizeStatus(input.Status)
	if !models.IsValidStatus(status) {
		apiError(c, http.StatusBadRequest, "invalid_status", "Unknown application status.")
		return
	}

	result := s.database().Model(&models.Application{}).
		Where("id IN ?", ids).
		Update("status", status)
	if result.Error != nil {
		apiError(c, http.StatusInternalServerError, "update_failed", "Failed to update the selected applications.")
		return
	}
	c.JSON(http.StatusOK, gin.H{"updated": result.RowsAffected})
}

func validateApplicationInput(input applicationInput) (*models.Application, string, string) {
	category := models.NormalizeCategory(input.Category)
	if category == "" {
		category = models.CategorySpring
	}
	if !models.IsValidCategory(category) {
		return nil, "invalid_category", "Unknown application category."
	}

	status := models.NormalizeStatus(input.Status)
	if status == "" {
		status = models.StatusScreening
	}
	if !models.IsValidStatus(status) {
		return nil, "invalid_status", "Unknown application status."
	}

	company := strings.TrimSpace(input.CompanyName)
	if company == "" {
		return nil, "company_required", "Company name is required."
	}
	if utf8.RuneCountInString(company) > 255 {
		return nil, "company_too_long", "Company name is too long."
	}

	link := normalizeLink(input.Link)
	if len(link) > 2048 {
		return nil, "link_too_long", "Application link is too long."
	}

	return &models.Application{
		Category:    category,
		CompanyName: company,
		Link:        link,
		Status:      status,
		Pinned:      input.Pinned,
	}, "", ""
}

// normalizeLink intentionally only trims whitespace. Users may store any
// identifier or partial URL, and the app must never silently rewrite links by
// adding a protocol prefix.
func normalizeLink(raw string) string {
	return strings.TrimSpace(raw)
}

func uniqueIDs(ids []uint) []uint {
	seen := make(map[uint]struct{}, len(ids))
	result := make([]uint, 0, len(ids))
	for _, id := range ids {
		if id == 0 {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result
}

func (s *Server) stats(c *gin.Context) {
	type groupRow struct {
		Category string
		Status   string
		Count    int64
	}
	var rows []groupRow
	if err := s.database().Model(&models.Application{}).
		Select("category, status, COUNT(*) AS count").
		Group("category, status").
		Scan(&rows).Error; err != nil {
		apiError(c, http.StatusInternalServerError, "query_failed", "Failed to load statistics.")
		return
	}

	byCategory := map[string]int64{}
	byStatus := map[string]int64{}
	byCategoryStatus := map[string]map[string]int64{}
	for _, category := range models.Categories {
		byCategory[category] = 0
		byCategoryStatus[category] = map[string]int64{}
		for _, status := range models.Statuses {
			byCategoryStatus[category][status] = 0
		}
	}
	for _, status := range models.Statuses {
		byStatus[status] = 0
	}

	var total int64
	for _, row := range rows {
		total += row.Count
		if _, ok := byCategory[row.Category]; ok {
			byCategory[row.Category] += row.Count
		}
		if _, ok := byStatus[row.Status]; ok {
			byStatus[row.Status] += row.Count
		}
		if _, ok := byCategoryStatus[row.Category]; ok {
			byCategoryStatus[row.Category][row.Status] += row.Count
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"total":            total,
		"byCategory":       byCategory,
		"byStatus":         byStatus,
		"byCategoryStatus": byCategoryStatus,
	})
}
