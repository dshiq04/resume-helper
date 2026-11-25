package server

import (
	"errors"
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
		changedAt = time.Now().UTC()
	}
	return db.Clauses(clause.OnConflict{DoNothing: true}).Create(&models.ApplicationChange{
		ApplicationID: applicationID,
		Status:        status,
		ChangedAt:     changedAt,
	}).Error
}

// updateStatusChange records a normal status transition. The timestamp of an
// existing milestone is refreshed whenever the user enters that stage again,
// including regressions such as interview → screening. Without this, the
// "days in current status" values would keep counting from the first time the
// stage was ever reached.
func updateStatusChange(db *gorm.DB, applicationID uint, _ string, status string, changedAt time.Time) error {
	if applicationID == 0 || status == "" {
		return nil
	}
	if changedAt.IsZero() {
		changedAt = time.Now().UTC()
	}

	var existing models.ApplicationChange
	err := db.Where("application_id = ? AND status = ?", applicationID, status).First(&existing).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return db.Create(&models.ApplicationChange{
			ApplicationID: applicationID,
			Status:        status,
			ChangedAt:     changedAt,
		}).Error
	case err != nil:
		return err
	}

	if changedAt.After(existing.ChangedAt) {
		return db.Model(&existing).Update("changed_at", changedAt).Error
	}
	return nil
}

// timelineStatusRank returns the fixed position used by the detail timeline.
// Offer and rejected are parallel terminal stages, so they share a rank and
// are ordered by their timestamps instead.
func timelineStatusRank(status string) int {
	if status == models.StatusRejected {
		status = models.StatusOffer
	}
	for index, item := range timelineStatusOrder {
		if item == status {
			return index
		}
	}
	return len(timelineStatusOrder)
}

func isTerminalStatus(status string) bool {
	return status == models.StatusOffer || status == models.StatusRejected
}

func (s *Server) loadChanges(db *gorm.DB, applicationID uint) ([]models.ApplicationChange, error) {
	var items []models.ApplicationChange
	err := db.Where("application_id = ?", applicationID).
		Order("datetime(changed_at) ASC, id ASC").
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
			fail(c, http.StatusNotFound, "not_found", "The record does not exist.")
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

	// A record only needs its history filled in when it looks like it was
	// backfilled: the timeline holds nothing but the current status. As soon as
	// any earlier stage is recorded, the timeline came from real usage and the
	// supplement prompt would only be noise.
	chain := models.StageChainForStatus(record.Status)
	currentRank := timelineStatusRank(record.Status)
	hasEarlierStage := false
	for _, status := range chain {
		if status == record.Status || timelineStatusRank(status) >= currentRank {
			continue
		}
		if _, exists := present[status]; exists {
			hasEarlierStage = true
			break
		}
	}

	missing := make([]string, 0, len(chain))
	if !hasEarlierStage {
		for _, status := range chain {
			if _, exists := present[status]; !exists {
				missing = append(missing, status)
			}
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

	changedAt := time.Now().UTC()
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
			fail(c, http.StatusNotFound, "not_found", "The record does not exist.")
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

// parseFlexibleRangeTime parses a range bound. A bare date used as the upper
// bound means the whole day, so it is extended to the end of that day instead
// of silently excluding every record created after midnight.
func parseFlexibleRangeTime(value string, upperBound bool) (time.Time, error) {
	parsed, err := parseFlexibleTime(value)
	if err != nil {
		return time.Time{}, err
	}
	if upperBound && isDateOnly(value) {
		// AddDate keeps the end of day correct across DST transitions.
		parsed = parsed.AddDate(0, 0, 1).Add(-time.Nanosecond)
	}
	return parsed, nil
}

// isDateOnly reports whether value is exactly a "YYYY-MM-DD" date without a
// time or timezone component.
func isDateOnly(value string) bool {
	value = strings.TrimSpace(value)
	if len(value) != len("2006-01-02") {
		return false
	}
	_, err := time.ParseInLocation("2006-01-02", value, time.Local)
	return err == nil
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
	currentRank := timelineStatusRank(record.Status)

	// The applications table is authoritative. Drop milestones that sit after
	// the current status in the timeline order, then make sure the current
	// status is present so it is always the last node.
	filtered := make([]models.ApplicationChange, 0, len(changes)+1)
	for _, change := range changes {
		// Offer and rejected are mutually exclusive outcomes, so a record can
		// only keep the terminal stage that matches its current status.
		if isTerminalStatus(change.Status) &&
			isTerminalStatus(record.Status) &&
			change.Status != record.Status {
			continue
		}
		if timelineStatusRank(change.Status) <= currentRank {
			filtered = append(filtered, change)
		}
	}

	foundCurrent := false
	for _, change := range filtered {
		if change.Status == record.Status {
			foundCurrent = true
			break
		}
	}
	if !foundCurrent {
		changedAt := record.UpdatedAt
		if changedAt.IsZero() || (len(filtered) > 0 && changedAt.Before(filtered[len(filtered)-1].ChangedAt)) {
			changedAt = time.Now().UTC()
		}
		if changedAt.IsZero() {
			changedAt = record.AppliedAt
		}
		filtered = append(filtered, models.ApplicationChange{
			ApplicationID: record.ID,
			Status:        record.Status,
			ChangedAt:     changedAt,
		})
	}

	sortChangesForTimeline(filtered)
	return filtered
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
			Order("datetime(changed_at) ASC, id ASC").
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

// timelineStatusOrder is the display order used by the detail timeline. It is
// intentionally separate from models.Statuses so the list filters and select
// options keep their existing order.
var timelineStatusOrder = []string{
	models.StatusPending,
	models.StatusScreening,
	models.StatusWritten,
	models.StatusInterview,
	models.StatusOffer,
	models.StatusRejected,
}

// sortChangesForTimeline follows the fixed stage order shown in the UI instead
// of the chronological order of the change records. Offer and rejected share a
// rank, so parallel terminal stages fall back to their timestamps.
func sortChangesForTimeline(items []models.ApplicationChange) {
	sort.SliceStable(items, func(left, right int) bool {
		leftRank := timelineStatusRank(items[left].Status)
		rightRank := timelineStatusRank(items[right].Status)
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
