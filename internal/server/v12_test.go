package server

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"resume-helper/internal/models"
)

func TestStatusHistoryAndDeduplication(t *testing.T) {
	app := newTestApp(t)

	_, body := app.request(t, http.MethodPost, "/api/applications", map[string]string{
		"companyName": "History Corp",
		"category":    "autumn",
		"status":      "screening",
	}, "")
	created := decodeJSON[models.Application](t, body)

	resp, body := app.request(t, http.MethodGet, "/api/application/"+uintString(created.ID)+"/history", nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("initial history status = %d, body = %s", resp.StatusCode, body)
	}
	initial := decodeJSON[applicationHistoryResponse](t, body)
	if len(initial.Items) != 1 || initial.Items[0].Status != models.StatusScreening {
		t.Fatalf("unexpected initial history: %+v", initial.Items)
	}

	path := "/api/application/" + uintString(created.ID)
	for _, status := range []string{"written", "interview"} {
		resp, body = app.request(t, http.MethodPut, path, map[string]string{"status": status}, "")
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("update %s status = %d, body = %s", status, resp.StatusCode, body)
		}
	}
	// Going back to an already recorded stage must not create a duplicate.
	resp, body = app.request(t, http.MethodPut, path, map[string]string{"status": "written"}, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("dedupe update status = %d, body = %s", resp.StatusCode, body)
	}

	resp, body = app.request(t, http.MethodGet, path+"/history", nil, "")
	history := decodeJSON[applicationHistoryResponse](t, body)
	if len(history.Items) != 3 {
		t.Fatalf("history length = %d, want 3: %+v", len(history.Items), history.Items)
	}
	if len(history.MissingStatuses) != 0 {
		t.Fatalf("missing statuses = %v, want none", history.MissingStatuses)
	}
}

func TestRecycleBinRestoresOriginalRecord(t *testing.T) {
	app := newTestApp(t)

	_, body := app.request(t, http.MethodPost, "/api/applications", map[string]string{
		"companyName": "Recycle Corp",
		"category":    "spring",
		"status":      "screening",
	}, "")
	created := decodeJSON[models.Application](t, body)
	app.request(t, http.MethodPut, "/api/application/"+uintString(created.ID), map[string]string{"status": "interview"}, "")

	resp, body := app.request(t, http.MethodDelete, "/api/application/"+uintString(created.ID), nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("delete status = %d, body = %s", resp.StatusCode, body)
	}
	deleted := decodeJSON[map[string]any](t, body)
	if deleted["trashId"].(float64) == 0 {
		t.Fatalf("delete did not return a trash id: %+v", deleted)
	}

	resp, body = app.request(t, http.MethodGet, "/api/trash", nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("trash status = %d, body = %s", resp.StatusCode, body)
	}
	trash := decodeJSON[trashListTest](t, body)
	if trash.Total != 1 || trash.Items[0].OriginalID != created.ID {
		t.Fatalf("unexpected trash: %+v", trash)
	}

	resp, body = app.request(t, http.MethodPost, "/api/trash/"+uintString(trash.Items[0].ID)+"/restore", nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("restore status = %d, body = %s", resp.StatusCode, body)
	}
	restored := decodeJSON[models.Application](t, body)
	if restored.ID != created.ID || restored.Status != models.StatusInterview {
		t.Fatalf("unexpected restored record: %+v", restored)
	}

	resp, body = app.request(t, http.MethodGet, "/api/application/"+uintString(created.ID)+"/history", nil, "")
	history := decodeJSON[applicationHistoryResponse](t, body)
	if len(history.Items) != 2 {
		t.Fatalf("restored history length = %d, want 2", len(history.Items))
	}
}

func TestBackupRoundTripPreservesDataAndDeviceID(t *testing.T) {
	app := newTestApp(t)
	deviceID, err := app.server.ensureDeviceID(false)
	if err != nil {
		t.Fatalf("ensure device id: %v", err)
	}

	app.request(t, http.MethodPost, "/api/applications", map[string]string{
		"companyName": "Backup Corp",
		"category":    "social",
	}, "")
	app.request(t, http.MethodPut, "/api/system/preferences", map[string]any{
		"closeAction":     "tray",
		"idleLockEnabled": true,
		"idleLockMinutes": 30,
	}, "")

	target := filepath.Join(t.TempDir(), "backup.zip")
	exported, err := app.server.ExportBackup(target)
	if err != nil {
		t.Fatalf("export backup: %v", err)
	}
	if exported.Counts.Applications != 1 {
		t.Fatalf("exported applications = %d, want 1", exported.Counts.Applications)
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("backup file missing: %v", err)
	}

	inspected, err := app.server.InspectBackup(target)
	if err != nil {
		t.Fatalf("inspect backup: %v", err)
	}
	if inspected.Counts.Applications != 1 {
		t.Fatalf("inspected applications = %d, want 1", inspected.Counts.Applications)
	}

	// Add data that must disappear after restoring the backup.
	app.request(t, http.MethodPost, "/api/applications", map[string]string{
		"companyName": "Later Corp",
		"category":    "spring",
	}, "")

	imported, err := app.server.ImportBackup(target)
	if err != nil {
		t.Fatalf("import backup: %v", err)
	}
	if imported.Counts.Applications != 1 {
		t.Fatalf("imported applications = %d, want 1", imported.Counts.Applications)
	}

	var items []models.Application
	if err := app.server.database().Find(&items).Error; err != nil {
		t.Fatalf("load imported applications: %v", err)
	}
	if len(items) != 1 || items[0].CompanyName != "Backup Corp" {
		t.Fatalf("unexpected imported applications: %+v", items)
	}

	after, err := app.server.ensureDeviceID(true)
	if err != nil {
		t.Fatalf("reload device id: %v", err)
	}
	if after != deviceID {
		t.Fatalf("device id changed after import: %q -> %q", deviceID, after)
	}

	preferences := app.server.preferences()
	if preferences.CloseAction != CloseActionTray || !preferences.IdleLockEnabled || preferences.IdleLockMinutes != 30 {
		t.Fatalf("preferences were not restored: %+v", preferences)
	}
}

func TestBackupReEncryptsResumeFiles(t *testing.T) {
	app := newTestApp(t)

	source := filepath.Join(t.TempDir(), "resume.md")
	content := "# Hello resume\n\nBackup round trip."
	if err := os.WriteFile(source, []byte(content), 0o644); err != nil {
		t.Fatalf("write source resume: %v", err)
	}
	created, err := app.server.ImportResumeFromPath(source, "Round trip", []string{"test"})
	if err != nil {
		t.Fatalf("import resume: %v", err)
	}

	before, err := app.server.readResumeFile(*created)
	if err != nil || string(before) != content {
		t.Fatalf("read original resume: %v, content = %q", err, string(before))
	}

	target := filepath.Join(t.TempDir(), "with-resume.zip")
	if _, err := app.server.ExportBackup(target); err != nil {
		t.Fatalf("export backup with resume: %v", err)
	}
	if _, err := app.server.ImportBackup(target); err != nil {
		t.Fatalf("import backup with resume: %v", err)
	}

	var resumes []models.Resume
	if err := app.server.database().Find(&resumes).Error; err != nil {
		t.Fatalf("load imported resumes: %v", err)
	}
	if len(resumes) != 1 {
		t.Fatalf("resume count = %d, want 1", len(resumes))
	}
	if !resumes[0].Encrypted {
		t.Fatal("imported resume should be encrypted")
	}
	if resumes[0].StoredName == created.StoredName {
		t.Fatal("import should write resume files with a new stored name")
	}
	after, err := app.server.readResumeFile(resumes[0])
	if err != nil || string(after) != content {
		t.Fatalf("read imported resume: %v, content = %q", err, string(after))
	}
}

func TestPreferencesAndCompanySuggestions(t *testing.T) {
	app := newTestApp(t)

	resp, body := app.request(t, http.MethodPut, "/api/system/preferences", map[string]any{
		"closeAction":     "quit",
		"idleLockEnabled": true,
		"idleLockMinutes": 45,
	}, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("update preferences status = %d, body = %s", resp.StatusCode, body)
	}
	prefs := decodeJSON[preferencesResponse](t, body)
	if prefs.CloseAction != CloseActionQuit || !prefs.IdleLockEnabled || prefs.IdleLockMinutes != 45 {
		t.Fatalf("unexpected preferences: %+v", prefs)
	}

	for _, company := range []string{"Acme", "Acme", "Beta"} {
		app.request(t, http.MethodPost, "/api/applications", map[string]string{
			"companyName": company,
			"category":    "spring",
		}, "")
	}
	resp, body = app.request(t, http.MethodGet, "/api/applications/companies", nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("company suggestions status = %d, body = %s", resp.StatusCode, body)
	}
	suggestions := decodeJSON[companySuggestionsTest](t, body)
	if len(suggestions.Items) != 2 {
		t.Fatalf("company suggestions = %v, want 2 distinct values", suggestions.Items)
	}
}

type trashListTest struct {
	Items []models.DeletedApplication `json:"items"`
	Total int                         `json:"total"`
}

type companySuggestionsTest struct {
	Items []string `json:"items"`
}

func uintString(value uint) string {
	return fmt.Sprintf("%d", value)
}
