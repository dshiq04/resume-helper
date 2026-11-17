package server

import (
	"net/http"
	"sort"
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
	From     string `form:"from"`
	To       string `form:"to"`
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

type batchCategoryInput struct {
	IDs      []uint `json:"ids"`
	Category string `json:"category"`
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

	var from, to *time.Time
	if strings.TrimSpace(params.From) != "" {
		parsed, err := parseFlexibleTime(params.From)
		if err != nil {
			apiError(c, http.StatusBadRequest, "invalid_range", "Invalid date range.")
			return
		}
		from = &parsed
	}
	if strings.TrimSpace(params.To) != "" {
		parsed, err := parseFlexibleTime(params.To)
		if err != nil {
			apiError(c, http.StatusBadRequest, "invalid_range", "Invalid date range.")
			return
		}
		to = &parsed
	}
	if from != nil && to != nil && to.Before(*from) {
		apiError(c, http.StatusBadRequest, "invalid_range", "The end date must be after the start date.")
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
		if from != nil {
			query = query.Where("applied_at >= ?", *from)
		}
		if to != nil {
			query = query.Where("applied_at <= ?", *to)
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
		Order("pinned DESC, applied_at DESC, id DESC").
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
		"items":    s.attachStatusDays(db, items),
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

	db := s.database()
	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(record).Error; err != nil {
			return err
		}
		return recordStatusChange(tx, record.ID, record.Status, record.AppliedAt)
	}); err != nil {
		apiError(c, http.StatusInternalServerError, "create_failed", "Failed to create the application.")
		return
	}
	c.JSON(http.StatusCreated, s.attachStatusDays(db, []models.Application{*record})[0])
}

func (s *Server) getApplication(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	var record models.Application
	if err := s.database().First(&record, id).Error; err != nil {
		if isRecordNotFound(err) {
			fail(c, http.StatusNotFound, "not_found", "原内容不存在")
			return
		}
		apiError(c, http.StatusInternalServerError, "query_failed", "Failed to load the application.")
		return
	}

	c.JSON(http.StatusOK, s.attachStatusDays(s.database(), []models.Application{record})[0])
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
			fail(c, http.StatusNotFound, "not_found", "原内容不存在")
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

	previousStatus := record.Status
	newStatus := ""
	if input.Status != nil {
		newStatus = models.NormalizeStatus(*input.Status)
	}

	if len(updates) > 0 {
		if err := db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Model(&record).Updates(updates).Error; err != nil {
				return err
			}
			if newStatus != "" && newStatus != previousStatus {
				return recordStatusChange(tx, record.ID, newStatus, time.Now())
			}
			return nil
		}); err != nil {
			apiError(c, http.StatusInternalServerError, "update_failed", "Failed to update the application.")
			return
		}
	}

	if err := db.First(&record, id).Error; err != nil {
		apiError(c, http.StatusInternalServerError, "query_failed", "Failed to reload the application.")
		return
	}
	c.JSON(http.StatusOK, s.attachStatusDays(db, []models.Application{record})[0])
}

func (s *Server) deleteApplication(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	db := s.database()
	var trashID uint
	err := db.Transaction(func(tx *gorm.DB) error {
		var record models.Application
		if err := tx.First(&record, id).Error; err != nil {
			return err
		}
		trash, err := moveApplicationToTrash(tx, record)
		if err != nil {
			return err
		}
		trashID = trash.ID
		return nil
	})
	if err != nil {
		if isRecordNotFound(err) {
			apiError(c, http.StatusNotFound, "not_found", "Application not found.")
			return
		}
		apiError(c, http.StatusInternalServerError, "delete_failed", "Failed to delete the application.")
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": 1, "trashId": trashID})
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

	trashIDs := make([]uint, 0, len(ids))
	err := s.database().Transaction(func(tx *gorm.DB) error {
		var records []models.Application
		if err := tx.Where("id IN ?", ids).Find(&records).Error; err != nil {
			return err
		}
		for _, record := range records {
			trash, err := moveApplicationToTrash(tx, record)
			if err != nil {
				return err
			}
			trashIDs = append(trashIDs, trash.ID)
		}
		return nil
	})
	if err != nil {
		apiError(c, http.StatusInternalServerError, "delete_failed", "Failed to delete the selected applications.")
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": len(trashIDs), "trashIds": trashIDs})
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

	db := s.database()
	changed := make([]models.Application, 0, len(ids))
	err := db.Transaction(func(tx *gorm.DB) error {
		var records []models.Application
		if err := tx.Where("id IN ?", ids).Find(&records).Error; err != nil {
			return err
		}
		for _, record := range records {
			if record.Status == status {
				continue
			}
			if err := tx.Model(&models.Application{}).
				Where("id = ?", record.ID).
				Update("status", status).Error; err != nil {
				return err
			}
			if err := recordStatusChange(tx, record.ID, status, time.Now()); err != nil {
				return err
			}
			changed = append(changed, record)
		}
		return nil
	})
	if err != nil {
		apiError(c, http.StatusInternalServerError, "update_failed", "Failed to update the selected applications.")
		return
	}
	c.JSON(http.StatusOK, gin.H{"updated": len(changed)})
}

func (s *Server) batchUpdateCategory(c *gin.Context) {
	var input batchCategoryInput
	if err := c.ShouldBindJSON(&input); err != nil {
		apiError(c, http.StatusBadRequest, "invalid_body", "Invalid request body.")
		return
	}
	ids := uniqueIDs(input.IDs)
	if len(ids) == 0 {
		apiError(c, http.StatusBadRequest, "ids_required", "Select at least one application.")
		return
	}
	category := models.NormalizeCategory(input.Category)
	if !models.IsValidCategory(category) {
		apiError(c, http.StatusBadRequest, "invalid_category", "Unknown application category.")
		return
	}

	result := s.database().Model(&models.Application{}).
		Where("id IN ?", ids).
		Update("category", category)
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
	from, to, ok := parseStatsRange(c)
	if !ok {
		return
	}
	db := s.database()
	base := func() *gorm.DB {
		query := db.Model(&models.Application{})
		if from != nil {
			query = query.Where("applied_at >= ?", *from)
		}
		if to != nil {
			query = query.Where("applied_at <= ?", *to)
		}
		return query
	}

	type groupRow struct {
		Category string
		Status   string
		Count    int64
	}
	var rows []groupRow
	if err := base().
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

	var resumeCount int64
	if err := s.database().Model(&models.Resume{}).Count(&resumeCount).Error; err != nil {
		apiError(c, http.StatusInternalServerError, "query_failed", "Failed to load statistics.")
		return
	}

	var screeningItems []models.Application
	if err := base().Where("status = ?", models.StatusScreening).Find(&screeningItems).Error; err != nil {
		apiError(c, http.StatusInternalServerError, "query_failed", "Failed to load statistics.")
		return
	}
	views := s.attachStatusDays(db, screeningItems)
	daysSum := 0
	maxDays := 0
	for _, view := range views {
		if view.ScreeningDays == nil {
			continue
		}
		daysSum += *view.ScreeningDays
		if *view.ScreeningDays > maxDays {
			maxDays = *view.ScreeningDays
		}
	}
	averageDays := 0
	if len(views) > 0 {
		averageDays = daysSum / len(views)
	}

	c.JSON(http.StatusOK, gin.H{
		"total":            total,
		"resumes":          resumeCount,
		"byCategory":       byCategory,
		"byStatus":         byStatus,
		"byCategoryStatus": byCategoryStatus,
		"stuckScreening": gin.H{
			"count":       len(views),
			"averageDays": averageDays,
			"maxDays":     maxDays,
		},
	})
}

type dailyStat struct {
	Date     string `json:"date"`
	Category string `json:"category"`
	Count    int64  `json:"count"`
}

// dailyStats returns per-day application counts grouped by category. The
// aggregation happens in Go so the query stays portable across SQLite, MySQL
// and PostgreSQL, which all format dates differently.
func (s *Server) dailyStats(c *gin.Context) {
	from, to, ok := parseStatsRange(c)
	if !ok {
		return
	}
	query := s.database().Model(&models.Application{}).Select("category, applied_at")
	if from != nil {
		query = query.Where("applied_at >= ?", *from)
	}
	if to != nil {
		query = query.Where("applied_at <= ?", *to)
	}
	var rows []struct {
		Category  string
		AppliedAt time.Time
	}
	if err := query.Scan(&rows).Error; err != nil {
		apiError(c, http.StatusInternalServerError, "query_failed", "Failed to load statistics.")
		return
	}

	counts := make(map[string]*dailyStat, len(rows))
	keys := make([]string, 0, len(rows))
	for _, row := range rows {
		if row.AppliedAt.IsZero() {
			continue
		}
		category := models.NormalizeCategory(row.Category)
		if !models.IsValidCategory(category) {
			continue
		}
		date := row.AppliedAt.Local().Format("2006-01-02")
		key := date + "|" + category
		item, exists := counts[key]
		if !exists {
			item = &dailyStat{Date: date, Category: category}
			counts[key] = item
			keys = append(keys, key)
		}
		item.Count++
	}

	sort.Strings(keys)
	daily := make([]dailyStat, 0, len(keys))
	for _, key := range keys {
		daily = append(daily, *counts[key])
	}

	c.JSON(http.StatusOK, gin.H{"daily": daily})
}
