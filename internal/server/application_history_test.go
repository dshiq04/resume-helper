package server

import (
	"net/http"
	"reflect"
	"testing"
	"time"

	"resume-helper/internal/models"
)

func TestHistoryMissingStatusesRequireBackfilledRecord(t *testing.T) {
	app := newTestApp(t)
	_, body := app.request(t, http.MethodPost, "/api/applications", map[string]string{
		"companyName": "Terminal Corp",
		"category":    "autumn",
		"status":      models.StatusRejected,
	}, "")
	created := decodeJSON[applicationStatusView](t, body)

	// Simulate a record created before status history existed: only the current
	// status was backfilled, so every earlier stage still needs a date.
	db := app.server.database()
	if err := db.Where("application_id = ? AND status <> ?", created.ID, models.StatusRejected).
		Delete(&models.ApplicationChange{}).Error; err != nil {
		t.Fatalf("clear backfilled history: %v", err)
	}

	_, body = app.request(t, http.MethodGet, "/api/application/"+uintString(created.ID)+"/history", nil, "")
	history := decodeJSON[applicationHistoryResponse](t, body)
	want := []string{models.StatusPending, models.StatusScreening}
	if !reflect.DeepEqual(history.MissingStatuses, want) {
		t.Fatalf("missing statuses = %v, want %v", history.MissingStatuses, want)
	}

	// Recording any earlier stage means the timeline came from real usage, so
	// the supplement prompt is no longer needed.
	if err := db.Create(&models.ApplicationChange{
		ApplicationID: created.ID,
		Status:        models.StatusScreening,
		ChangedAt:     time.Now(),
	}).Error; err != nil {
		t.Fatalf("record earlier stage: %v", err)
	}
	_, body = app.request(t, http.MethodGet, "/api/application/"+uintString(created.ID)+"/history", nil, "")
	history = decodeJSON[applicationHistoryResponse](t, body)
	if len(history.MissingStatuses) != 0 {
		t.Fatalf("missing statuses = %v, want none", history.MissingStatuses)
	}
}

func TestReconcileApplicationChangesKeepsMatchingTerminal(t *testing.T) {
	start := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	offer := models.ApplicationChange{
		ID:        1,
		Status:    models.StatusOffer,
		ChangedAt: start.Add(48 * time.Hour),
	}
	rejected := models.ApplicationChange{
		ID:        2,
		Status:    models.StatusRejected,
		ChangedAt: start.Add(72 * time.Hour),
	}

	offerRecord := models.Application{ID: 1, Status: models.StatusOffer}
	got := reconcileApplicationChanges(offerRecord, []models.ApplicationChange{rejected, offer})
	if len(got) != 1 || got[0].Status != models.StatusOffer {
		t.Fatalf("offer record should keep only the offer change, got %+v", got)
	}

	rejectedRecord := models.Application{ID: 1, Status: models.StatusRejected}
	got = reconcileApplicationChanges(rejectedRecord, []models.ApplicationChange{offer, rejected})
	if len(got) != 1 || got[0].Status != models.StatusRejected {
		t.Fatalf("rejected record should keep only the rejected change, got %+v", got)
	}
}

func TestSortChangesForTimelineTreatsTerminalStagesAsParallel(t *testing.T) {
	start := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	items := []models.ApplicationChange{
		{Status: models.StatusOffer, ChangedAt: start.Add(48 * time.Hour)},
		{Status: models.StatusInterview, ChangedAt: start.Add(24 * time.Hour)},
		{Status: models.StatusRejected, ChangedAt: start.Add(72 * time.Hour)},
	}

	sortChangesForTimeline(items)

	want := []string{models.StatusInterview, models.StatusOffer, models.StatusRejected}
	for index, change := range items {
		if change.Status != want[index] {
			t.Fatalf("item %d = %s, want %s", index, change.Status, want[index])
		}
	}
}
