package server

import (
	"fmt"
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
	DaysInStatus      *int `json:"daysInStatus,omitempty"`
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
	StatusDays    *int `json:"statusDays,omitempty"`
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
	changes = reconcileApplicationChanges(record, changes)
	sortChangesForTimeline(changes)

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
	now := time.Now()
	for index, change := range changes {
		view := applicationChangeView{ApplicationChange: change}
		if !previous.IsZero() {
			days := int(change.ChangedAt.Sub(previous).Hours() / 24)
			if days < 0 {
				days = 0
			}
			view.DaysSincePrevious = &days
		}
		next := now
		if index+1 < len(changes) {
			next = changes[index+1].ChangedAt
		}
		daysInStatus := int(next.Sub(change.ChangedAt).Hours() / 24)
		if daysInStatus < 0 {
			daysInStatus = 0
		}
		view.DaysInStatus = &daysInStatus
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

// reconcileApplicationChanges makes the applications table authoritative when
// its current status and the change history disagree. If the current status is
// already recorded, later milestones are discarded; otherwise the current
// status is appended so the timeline always ends at the real current stage.
func reconcileApplicationChanges(record models.Application, changes []models.ApplicationChange) []models.ApplicationChange {
	sortChanges(changes)
	if len(changes) == 0 {
		return []models.ApplicationChange{{
			ApplicationID: record.ID,
			Status:        record.Status,
			ChangedAt:     record.AppliedAt,
		}}
	}

	lastMatch := -1
	for index, change := range changes {
		if change.Status == record.Status {
			lastMatch = index
		}
	}
	if lastMatch >= 0 {
		return changes[:lastMatch+1]
	}

	changedAt := record.UpdatedAt
	if changedAt.IsZero() || changedAt.Before(changes[len(changes)-1].ChangedAt) {
		changedAt = time.Now()
	}
	return append(changes, models.ApplicationChange{
		ApplicationID: record.ID,
		Status:        record.Status,
		ChangedAt:     changedAt,
	})
}

// attachStatusDays adds the age of the current status to application rows.
// The query is batched so listing a page never performs one lookup per row.
func (s *Server) attachStatusDays(db *gorm.DB, items []models.Application) []applicationStatusView {
	views := make([]applicationStatusView, 0, len(items))
	ids := make([]uint, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}

	statusStart := make(map[string]time.Time, len(ids))
	if len(ids) > 0 {
		var changes []models.ApplicationChange
		if err := db.Where("application_id IN ?", ids).
			Order("changed_at ASC, id ASC").
			Find(&changes).Error; err == nil {
			for _, change := range changes {
				key := statusKey(change.ApplicationID, change.Status)
				if _, exists := statusStart[key]; !exists {
					statusStart[key] = change.ChangedAt
				}
			}
		}
	}

	for _, item := range items {
		view := applicationStatusView{Application: item}
		since, exists := statusStart[statusKey(item.ID, item.Status)]
		if !exists {
			since = item.AppliedAt
		}
		days := daysSince(since)
		view.StatusDays = &days
		if item.Status == models.StatusScreening {
			view.ScreeningDays = &days
		}
		views = append(views, view)
	}
	return views
}

func statusKey(applicationID uint, status string) string {
	return fmt.Sprintf("%d|%s", applicationID, status)
}

// sortChangesForTimeline follows the fixed stage order shown in the UI
// (screening → written → interview → pending → offer → rejected) instead of
// the chronological order of the change records.
func sortChangesForTimeline(items []models.ApplicationChange) {
	rank := make(map[string]int, len(models.Statuses))
	for index, status := range models.Statuses {
		rank[status] = index
	}
	sort.SliceStable(items, func(left, right int) bool {
		leftRank, leftOk := rank[items[left].Status]
		rightRank, rightOk := rank[items[right].Status]
		if !leftOk {
			leftRank = len(models.Statuses)
		}
		if !rightOk {
			rightRank = len(models.Statuses)
		}
		if leftRank == rightRank {
			return items[left].ChangedAt.Before(items[right].ChangedAt)
		}
		return leftRank < rightRank
	})
}

func sortChanges(items []models.ApplicationChange) {
	sort.SliceStable(items, func(left, right int) bool {
		if items[left].ChangedAt.Equal(items[right].ChangedAt) {
			return items[left].ID < items[right].ID
		}
		return items[left].ChangedAt.Before(items[right].ChangedAt)
	})
}
