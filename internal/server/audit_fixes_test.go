package server

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"resume-helper/internal/models"

	"github.com/xuri/excelize/v2"
)

func TestDateRangeUsesLocalDayBounds(t *testing.T) {
	app := newTestApp(t)

	now := time.Now()
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	// Two records are on the same local day but span the hours that used to be
	// dropped by the text comparison (late evening) and the start of the day.
	late := dayStart.Add(23*time.Hour + 30*time.Minute)
	early := dayStart.Add(30 * time.Minute)
	records := []models.Application{
		{Category: models.CategorySpring, CompanyName: "Early Co", Status: models.StatusScreening, AppliedAt: early},
		{Category: models.CategorySpring, CompanyName: "Late Co", Status: models.StatusScreening, AppliedAt: late},
		// Same instant as Late Co but stored with a different timezone offset,
		// like data exported on another device or created before UTC storage.
		{Category: models.CategorySpring, CompanyName: "Offset Co", Status: models.StatusScreening, AppliedAt: late.In(time.FixedZone("offset", -5*3600))},
	}
	for _, record := range records {
		if err := app.server.database().Create(&record).Error; err != nil {
			t.Fatalf("insert record %q: %v", record.CompanyName, err)
		}
	}

	from := dayStart.UTC().Format(time.RFC3339Nano)
	to := dayStart.AddDate(0, 0, 1).Add(-time.Nanosecond).UTC().Format(time.RFC3339Nano)
	rangeQuery := "from=" + url.QueryEscape(from) + "&to=" + url.QueryEscape(to)

	resp, body := app.request(t, http.MethodGet, "/api/applications?page=1&pageSize=10&"+rangeQuery, nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list status = %d, body = %s", resp.StatusCode, body)
	}
	list := decodeJSON[ApplicationList](t, body)
	if list.Total != 3 {
		t.Fatalf("local-day list total = %d, want 3 (timezone comparison bug)", list.Total)
	}
	if len(list.Items) != 3 || list.Items[len(list.Items)-1].CompanyName != "Early Co" {
		t.Fatalf("list should be ordered by the real instant, got %+v", list.Items)
	}

	resp, body = app.request(t, http.MethodGet, "/api/stats?"+rangeQuery, nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("stats status = %d, body = %s", resp.StatusCode, body)
	}
	stats := decodeJSON[Stats](t, body)
	if stats.Total != 3 {
		t.Fatalf("local-day stats total = %d, want 3", stats.Total)
	}

	resp, body = app.request(t, http.MethodGet, "/api/stats/daily?"+rangeQuery, nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("daily status = %d, body = %s", resp.StatusCode, body)
	}
	daily := decodeJSON[DailyStats](t, body)
	if len(daily.Daily) != 1 || daily.Daily[0].Count != 3 || daily.Daily[0].Date != now.Format("2006-01-02") {
		t.Fatalf("local-day daily stats = %+v, want one bucket with 3 records", daily.Daily)
	}

	// A bare date used as the upper bound must include the whole day.
	today := now.Format("2006-01-02")
	dateOnlyQuery := "from=" + url.QueryEscape(today) + "&to=" + url.QueryEscape(today)
	resp, body = app.request(t, http.MethodGet, "/api/applications?page=1&pageSize=10&"+dateOnlyQuery, nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("date-only list status = %d, body = %s", resp.StatusCode, body)
	}
	dateOnly := decodeJSON[ApplicationList](t, body)
	if dateOnly.Total != 3 {
		t.Fatalf("date-only same-day total = %d, want 3", dateOnly.Total)
	}

	resp, body = app.request(t, http.MethodGet, "/api/stats?"+dateOnlyQuery, nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("date-only stats status = %d, body = %s", resp.StatusCode, body)
	}
	if stats = decodeJSON[Stats](t, body); stats.Total != 3 {
		t.Fatalf("date-only stats total = %d, want 3", stats.Total)
	}
}

func TestBatchEndpointsRejectTooManyIDs(t *testing.T) {
	app := newTestApp(t)
	ids := make([]uint, 0, 2001)
	for value := uint(1); value <= 2001; value++ {
		ids = append(ids, value)
	}

	for _, tc := range []struct {
		method string
		path   string
		body   map[string]any
	}{
		{http.MethodPatch, "/api/applications/batch-status", map[string]any{"ids": ids, "status": models.StatusScreening}},
		{http.MethodPatch, "/api/applications/batch-category", map[string]any{"ids": ids, "category": models.CategorySpring}},
	} {
		resp, body := app.request(t, tc.method, tc.path, tc.body, "")
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("%s status = %d, want 400; body = %s", tc.path, resp.StatusCode, body)
		}
		var payload struct {
			Code string `json:"code"`
		}
		payload = decodeJSON[struct {
			Code string `json:"code"`
		}](t, body)
		if payload.Code != "too_many_ids" {
			t.Fatalf("%s code = %q, want too_many_ids", tc.path, payload.Code)
		}
	}
}

func TestDatabaseSwitchInvalidatesSessions(t *testing.T) {
	app := newTestApp(t)

	resp, body := app.request(t, http.MethodPost, "/api/auth/setup", map[string]string{"password": "secret123"}, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("setup status = %d, body = %s", resp.StatusCode, body)
	}
	session := decodeJSON[struct {
		Token string `json:"token"`
	}](t, body)
	if !app.server.auth.validateSession(session.Token) {
		t.Fatal("session should be valid after setup")
	}

	resp, body = app.request(t, http.MethodPut, "/api/system/database", map[string]any{
		"driver":     "sqlite",
		"sqlitePath": "second.db",
	}, session.Token)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("database switch status = %d, body = %s", resp.StatusCode, body)
	}
	if app.server.auth.validateSession(session.Token) {
		t.Fatal("switching databases must invalidate the previous session")
	}

	if sqlDB, err := app.server.database().DB(); err == nil {
		_ = sqlDB.Close()
	}
	// Let the background close of the previous pool finish before TempDir is
	// removed (Windows cannot unlink an open SQLite file).
	time.Sleep(1200 * time.Millisecond)
}

func TestCleanupDeletedResumeFiles(t *testing.T) {
	app := newTestApp(t)
	dir := app.server.resumeDirectory()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create resume dir: %v", err)
	}
	keep := filepath.Join(dir, "keep.reb")
	remove := filepath.Join(dir, "gone.reb.deleting")
	if err := os.WriteFile(keep, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(remove, []byte("remove"), 0o644); err != nil {
		t.Fatal(err)
	}

	app.server.cleanupDeletedResumeFiles()

	if _, err := os.Stat(keep); err != nil {
		t.Fatalf("normal resume file should be kept: %v", err)
	}
	if _, err := os.Stat(remove); !os.IsNotExist(err) {
		t.Fatalf("temporary .deleting file should be removed, stat err = %v", err)
	}
}

func TestExcelImportRejectsTooManyRowsWhileStreaming(t *testing.T) {
	app := newTestApp(t)
	file := excelize.NewFile()
	sheet := file.GetSheetName(0)
	header := []string{"Category", "Company", "Link", "Status"}
	for index, value := range header {
		cell, _ := excelize.CoordinatesToCellName(index+1, 1)
		if err := file.SetCellValue(sheet, cell, value); err != nil {
			t.Fatal(err)
		}
	}
	for row := 2; row <= maxExcelRows+2; row++ {
		cell, _ := excelize.CoordinatesToCellName(2, row)
		if err := file.SetCellValue(sheet, cell, fmt.Sprintf("Company %d", row)); err != nil {
			t.Fatal(err)
		}
	}
	defer func() { _ = file.Close() }()

	result, err := app.server.parseAndImportWorkbook(file)
	if err == nil {
		t.Fatalf("expected the row limit to be enforced, got result %+v", result)
	}
	if !strings.Contains(err.Error(), "more than") {
		t.Fatalf("error = %v, want a row-limit error", err)
	}
	var count int64
	if err := app.server.database().Model(&models.Application{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("imported %d applications despite exceeding the limit", count)
	}
}

func TestImportRejectsInvalidRecycleBinItem(t *testing.T) {
	app := newTestApp(t)
	resp, body := app.request(t, http.MethodPost, "/api/applications", map[string]string{
		"companyName": "Trash Validation Corp",
		"category":    "spring",
	}, "")
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", resp.StatusCode, body)
	}
	created := decodeJSON[models.Application](t, body)
	resp, body = app.request(t, http.MethodDelete, "/api/application/"+fmt.Sprintf("%d", created.ID), nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("delete status = %d, body = %s", resp.StatusCode, body)
	}

	source := filepath.Join(t.TempDir(), "valid.zip")
	if _, err := app.server.ExportBackup(source); err != nil {
		t.Fatalf("export backup: %v", err)
	}
	target := filepath.Join(t.TempDir(), "invalid-trash.zip")
	rewriteBackupManifest(t, source, target, func(manifest map[string]any) {
		trash, ok := manifest["trash"].([]any)
		if !ok || len(trash) == 0 {
			t.Fatalf("exported backup has no trash items: %+v", manifest["trash"])
		}
		item, ok := trash[0].(map[string]any)
		if !ok {
			t.Fatalf("unexpected trash item: %+v", trash[0])
		}
		item["status"] = "not-a-status"
	})

	if _, err := app.server.ImportBackup(target); err == nil || !strings.Contains(err.Error(), "recycle bin") {
		t.Fatalf("import error = %v, want a recycle-bin validation error", err)
	}
}
