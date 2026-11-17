package server

import (
	"net/http"
	"sort"
	"strings"
	"time"

	"resume-helper/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type applicationChangeView struct {
	models.ApplicationChange
	DaysSincePrevious *int `json:"daysSincePrevious,omitempty"`
}

type applicationHistoryResponse struct {
	Items           []applicationChangeView `json:"items"`
	MissingStatuses []string                `json:"missingStatuses"`
	ScreeningDays   int                     `json:"screeningDays"`
	AppliedAt       time.Time               `json:"appliedAt"`
}

type applicationStatusView struct {
	models.Application
	ScreeningDays *int `json:"screeningDays,omitempty"`
}

// recordStatusChange appends a milestone the first time an application
// reaches a status. The unique index on (application_id, status) makes this
// safe against concurrent requests as well.
func recordStatusChange(db *gorm.DB, applicationID uint, status string, changedAt time.Time) error {
	if applicationID == 0 || status == "" {
		return nil
	}
	if changedAt.IsZero() {
		changedAt = time.Now()
	}
	return db.Clauses(clause.OnConflict{DoNothing: true}).Create(&models.ApplicationChange{
		ApplicationID: applicationID,
		Status:        status,
		ChangedAt:     changedAt,
	}).Error
}

func (s *Server) loadChanges(db *gorm.DB, applicationID uint) ([]models.ApplicationChange, error) {
	var items []models.ApplicationChange
	err := db.Where("application_id = ?", applicationID).
		Order("changed_at ASC, id ASC").
		Find(&items).Error
	return items, err
}

func (s *Server) getApplicationHistory(c *gin.Context) {
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

	changes, err := s.loadChanges(db, id)
	if err != nil {
		apiError(c, http.StatusInternalServerError, "query_failed", "Failed to load the status history.")
		return
	}

	present := make(map[string]struct{}, len(changes))
	for _, change := range changes {
		present[change.Status] = struct{}{}
	}
	missing := make([]string, 0, len(models.StageOrder))
	for _, status := range models.StageChainForStatus(record.Status) {
		if _, exists := present[status]; !exists {
			missing = append(missing, status)
		}
	}

	views := make([]applicationChangeView, 0, len(changes))
	var previous time.Time
	for _, change := range changes {
		view := applicationChangeView{ApplicationChange: change}
		if !previous.IsZero() {
			days := int(change.ChangedAt.Sub(previous).Hours() / 24)
			if days < 0 {
				days = 0
			}
			view.DaysSincePrevious = &days
		}
		views = append(views, view)
		previous = change.ChangedAt
	}

	screeningDays := 0
	if record.Status == models.StatusScreening {
		since := record.AppliedAt
		for _, change := range changes {
			if change.Status == models.StatusScreening {
				since = change.ChangedAt
				break
			}
		}
		screeningDays = daysSince(since)
	}

	c.JSON(http.StatusOK, applicationHistoryResponse{
		Items:           views,
		MissingStatuses: missing,
		ScreeningDays:   screeningDays,
		AppliedAt:       record.AppliedAt,
	})
}

func (s *Server) createApplicationHistory(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	var input struct {
		Status    string `json:"status"`
		ChangedAt string `json:"changedAt"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		apiError(c, http.StatusBadRequest, "invalid_body", "Invalid request body.")
		return
	}

	status := models.NormalizeStatus(input.Status)
	if !models.IsValidStatus(status) {
		apiError(c, http.StatusBadRequest, "invalid_status", "Unknown application status.")
		return
	}

	changedAt := time.Now()
	if strings.TrimSpace(input.ChangedAt) != "" {
		parsed, err := parseFlexibleTime(input.ChangedAt)
		if err != nil {
			apiError(c, http.StatusBadRequest, "invalid_date", "Invalid history date.")
			return
		}
		changedAt = parsed
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

	if err := recordStatusChange(db, id, status, changedAt); err != nil {
		apiError(c, http.StatusInternalServerError, "create_failed", "Failed to save the status history.")
		return
	}

	changes, err := s.loadChanges(db, id)
	if err != nil {
		apiError(c, http.StatusInternalServerError, "query_failed", "Failed to load the status history.")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"items": changes})
}

func parseFlexibleTime(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	var lastErr error
	for _, layout := range layouts {
		parsed, err := time.ParseInLocation(layout, value, time.Local)
		if err == nil {
			return parsed, nil
		}
		lastErr = err
	}
	return time.Time{}, lastErr
}

func daysSince(value time.Time) int {
	if value.IsZero() {
		return 0
	}
	days := int(time.Since(value).Hours() / 24)
	if days < 0 {
		return 0
	}
	return days
}

// attachScreeningDays adds the "stuck in screening" age to application rows.
// The query is batched so listing a page never performs one lookup per row.
func (s *Server) attachScreeningDays(db *gorm.DB, items []models.Application) []applicationStatusView {
	views := make([]applicationStatusView, 0, len(items))
	ids := make([]uint, 0, len(items))
	for _, item := range items {
		if item.Status == models.StatusScreening {
			ids = append(ids, item.ID)
		}
	}

	screeningStart := make(map[uint]time.Time, len(ids))
	if len(ids) > 0 {
		var changes []models.ApplicationChange
		if err := db.Where("application_id IN ? AND status = ?", ids, models.StatusScreening).
			Order("changed_at ASC, id ASC").
			Find(&changes).Error; err == nil {
			for _, change := range changes {
				if _, exists := screeningStart[change.ApplicationID]; !exists {
					screeningStart[change.ApplicationID] = change.ChangedAt
				}
			}
		}
	}

	for _, item := range items {
		view := applicationStatusView{Application: item}
		if item.Status == models.StatusScreening {
			since, exists := screeningStart[item.ID]
			if !exists {
				since = item.AppliedAt
			}
			days := daysSince(since)
			view.ScreeningDays = &days
		}
		views = append(views, view)
	}
	return views
}

func sortChanges(items []models.ApplicationChange) {
	sort.SliceStable(items, func(left, right int) bool {
		if items[left].ChangedAt.Equal(items[right].ChangedAt) {
			return items[left].ID < items[right].ID
		}
		return items[left].ChangedAt.Before(items[right].ChangedAt)
	})
}
