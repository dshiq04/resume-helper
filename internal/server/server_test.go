package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"resume-helper/internal/config"
	"resume-helper/internal/database"
	"resume-helper/internal/models"

	"github.com/xuri/excelize/v2"
)

type testApp struct {
	server *Server
	http   *httptest.Server
}

func newTestApp(t *testing.T) *testApp {
	t.Helper()
	dir := t.TempDir()
	cfg := &config.Config{
		Server: config.ServerConfig{Host: "127.0.0.1"},
		Database: config.DatabaseConfig{
			Driver: "sqlite",
			DSN:    filepath.Join(dir, "test.db"),
		},
		SourcePath: filepath.Join(dir, "config.yaml"),
		DataDir:    dir,
	}
	db, err := database.Open(cfg.Database)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})

	server := New(cfg, db, nil)
	httpServer := httptest.NewUnstartedServer(server.engine)
	httpServer.Start()
	t.Cleanup(httpServer.Close)
	return &testApp{
		server: server,
		http:   httpServer,
	}
}

func (a *testApp) request(t *testing.T, method, path string, body any, token string) (*http.Response, []byte) {
	t.Helper()
	var reader io.Reader
	contentType := ""
	switch value := body.(type) {
	case nil:
	case []byte:
		reader = bytes.NewReader(value)
	case *bytes.Buffer:
		reader = value
	case string:
		reader = strings.NewReader(value)
		contentType = "application/json"
	default:
		data, err := json.Marshal(value)
		if err != nil {
			t.Fatalf("marshal request body: %v", err)
		}
		reader = bytes.NewReader(data)
		contentType = "application/json"
	}
	req, err := http.NewRequest(method, a.http.URL+path, reader)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	return resp, data
}

func decodeJSON[T any](t *testing.T, data []byte) T {
	t.Helper()
	var value T
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatalf("decode response %q: %v", string(data), err)
	}
	return value
}

func TestApplicationLifecycle(t *testing.T) {
	app := newTestApp(t)

	resp, body := app.request(t, http.MethodPost, "/api/applications", map[string]string{
		"category":    "autumn",
		"companyName": "Example Corp",
		"link":        "jobs.example.com/1",
		"status":      "screening",
	}, "")
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", resp.StatusCode, body)
	}
	created := decodeJSON[models.Application](t, body)
	if created.ID == 0 || created.Link != "jobs.example.com/1" {
		t.Fatalf("unexpected created application: %+v", created)
	}

	resp, body = app.request(t, http.MethodGet, "/api/applications?search=jobs.example&page=1&pageSize=10", nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list status = %d, body = %s", resp.StatusCode, body)
	}
	list := decodeJSON[ApplicationList](t, body)
	if list.Total != 1 || len(list.Items) != 1 || list.Items[0].CompanyName != "Example Corp" {
		t.Fatalf("unexpected search result: %+v", list)
	}

	resp, body = app.request(t, http.MethodPut, fmt.Sprintf("/api/application/%d", created.ID), map[string]string{
		"status": "offer",
	}, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("update status = %d, body = %s", resp.StatusCode, body)
	}
	updated := decodeJSON[models.Application](t, body)
	if updated.Status != models.StatusOffer {
		t.Fatalf("status = %q, want offer", updated.Status)
	}

	resp, body = app.request(t, http.MethodGet, "/api/stats", nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("stats status = %d, body = %s", resp.StatusCode, body)
	}
	stats := decodeJSON[Stats](t, body)
	if stats.Total != 1 || stats.ByStatus[models.StatusOffer] != 1 || stats.ByCategory[models.CategoryAutumn] != 1 {
		t.Fatalf("unexpected stats: %+v", stats)
	}

	resp, body = app.request(t, http.MethodGet, "/api/stats/daily", nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("daily stats status = %d, body = %s", resp.StatusCode, body)
	}
	daily := decodeJSON[DailyStats](t, body)
	if len(daily.Daily) != 1 {
		t.Fatalf("unexpected daily stats: %+v", daily)
	}
	entry := daily.Daily[0]
	if entry.Category != models.CategoryAutumn || entry.Count != 1 || entry.Date != created.AppliedAt.Local().Format("2006-01-02") {
		t.Fatalf("unexpected daily entry: %+v", entry)
	}

	resp, body = app.request(t, http.MethodPost, "/api/applications", map[string]string{
		"companyName": "Second Corp",
		"category":    "spring",
	}, "")
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create second status = %d, body = %s", resp.StatusCode, body)
	}
	second := decodeJSON[models.Application](t, body)
	if second.Status != models.StatusScreening || second.Category != models.CategorySpring {
		t.Fatalf("expected defaults, got %+v", second)
	}

	resp, body = app.request(t, http.MethodPut, fmt.Sprintf("/api/application/%d", created.ID), map[string]bool{
		"pinned": true,
	}, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("pin status = %d, body = %s", resp.StatusCode, body)
	}
	if pinned := decodeJSON[models.Application](t, body); !pinned.Pinned {
		t.Fatal("application should be pinned")
	}
	resp, body = app.request(t, http.MethodGet, "/api/applications?page=1&pageSize=10", nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("pinned list status = %d, body = %s", resp.StatusCode, body)
	}
	pinnedList := decodeJSON[ApplicationList](t, body)
	if len(pinnedList.Items) < 2 || pinnedList.Items[0].ID != created.ID {
		t.Fatalf("pinned application should be ordered first: %+v", pinnedList.Items)
	}

	resp, body = app.request(t, http.MethodPatch, "/api/applications/batch-status", map[string]any{
		"ids":    []uint{created.ID, second.ID},
		"status": "interview",
	}, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("batch status = %d, body = %s", resp.StatusCode, body)
	}
	batchResult := decodeJSON[map[string]int64](t, body)
	if batchResult["updated"] != 2 {
		t.Fatalf("batch updated = %d, want 2", batchResult["updated"])
	}

	resp, body = app.request(t, http.MethodDelete, fmt.Sprintf("/api/application/%d", second.ID), nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("delete status = %d, body = %s", resp.StatusCode, body)
	}

	resp, body = app.request(t, http.MethodGet, "/api/applications", nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("final list status = %d, body = %s", resp.StatusCode, body)
	}
	list = decodeJSON[ApplicationList](t, body)
	if list.Total != 1 {
		t.Fatalf("final total = %d, want 1", list.Total)
	}
}

func TestOptionalPassword(t *testing.T) {
	app := newTestApp(t)

	resp, body := app.request(t, http.MethodGet, "/api/auth/status", nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("auth status = %d, body = %s", resp.StatusCode, body)
	}
	status := decodeJSON[struct {
		Enabled       bool `json:"enabled"`
		Authenticated bool `json:"authenticated"`
	}](t, body)
	if status.Enabled || !status.Authenticated {
		t.Fatalf("unexpected initial auth status: %+v", status)
	}

	resp, body = app.request(t, http.MethodPost, "/api/auth/setup", map[string]string{"password": "123"}, "")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("short password status = %d, body = %s", resp.StatusCode, body)
	}

	resp, body = app.request(t, http.MethodPost, "/api/auth/setup", map[string]string{"password": "secret123"}, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("setup status = %d, body = %s", resp.StatusCode, body)
	}
	session := decodeJSON[struct {
		Token string `json:"token"`
	}](t, body)
	if session.Token == "" {
		t.Fatal("setup did not return a token")
	}

	resp, body = app.request(t, http.MethodGet, "/api/applications", nil, "")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status = %d, body = %s", resp.StatusCode, body)
	}

	resp, body = app.request(t, http.MethodGet, "/api/applications", nil, session.Token)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("authenticated status = %d, body = %s", resp.StatusCode, body)
	}

	resp, _ = app.request(t, http.MethodPost, "/api/auth/login", map[string]string{"password": "wrong"}, "")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("wrong login status = %d", resp.StatusCode)
	}
	resp, body = app.request(t, http.MethodPost, "/api/auth/login", map[string]string{"password": "secret123"}, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login status = %d, body = %s", resp.StatusCode, body)
	}
	loginSession := decodeJSON[struct {
		Token string `json:"token"`
	}](t, body)

	resp, body = app.request(t, http.MethodPost, "/api/auth/change-password", map[string]string{
		"currentPassword": "secret123",
		"newPassword":     "newsecret",
	}, loginSession.Token)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("change password status = %d, body = %s", resp.StatusCode, body)
	}
	changed := decodeJSON[struct {
		Token string `json:"token"`
	}](t, body)
	if changed.Token == "" || changed.Token == loginSession.Token {
		t.Fatal("change password should rotate the session token")
	}
	resp, _ = app.request(t, http.MethodGet, "/api/applications", nil, loginSession.Token)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("old token status = %d, want 401", resp.StatusCode)
	}
	resp, _ = app.request(t, http.MethodGet, "/api/applications", nil, changed.Token)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("new token status = %d, want 200", resp.StatusCode)
	}

	resp, body = app.request(t, http.MethodPost, "/api/auth/disable", map[string]string{
		"password": "newsecret",
	}, changed.Token)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("disable status = %d, body = %s", resp.StatusCode, body)
	}
	resp, body = app.request(t, http.MethodGet, "/api/applications", nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("post-disable status = %d, body = %s", resp.StatusCode, body)
	}
}

func TestExcelTemplateAndImport(t *testing.T) {
	app := newTestApp(t)

	resp, body := app.request(t, http.MethodGet, "/api/applications/template", nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("template status = %d, body = %s", resp.StatusCode, body)
	}
	if len(body) == 0 || resp.Header.Get("Content-Disposition") == "" {
		t.Fatal("template response is missing content")
	}
	file, err := excelize.OpenReader(bytes.NewReader(body))
	if err != nil {
		t.Fatalf("open downloaded template: %v", err)
	}
	if len(file.GetSheetList()) < 2 {
		t.Fatalf("expected data + instruction sheets, got %v", file.GetSheetList())
	}
	_ = file.Close()

	book := excelize.NewFile()
	rows := [][]interface{}{
		{"Category", "Company", "Link", "Status"},
		{"春招", "腾讯", "careers.tencent.com", "面试"},
		{"autumn", "Google", "google.com/careers", "offer"},
		{"实习", "无效状态公司", "", "nope"},
	}
	for rowIndex, row := range rows {
		for colIndex, value := range row {
			cell, _ := excelize.CoordinatesToCellName(colIndex+1, rowIndex+1)
			_ = book.SetCellValue("Sheet1", cell, value)
		}
	}
	buffer, err := book.WriteToBuffer()
	if err != nil {
		t.Fatalf("write test workbook: %v", err)
	}
	_ = book.Close()

	payload := &bytes.Buffer{}
	writer := multipart.NewWriter(payload)
	part, err := writer.CreateFormFile("file", "applications.xlsx")
	if err != nil {
		t.Fatalf("create multipart: %v", err)
	}
	if _, err := part.Write(buffer.Bytes()); err != nil {
		t.Fatalf("write multipart: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, app.http.URL+"/api/applications/import", payload)
	if err != nil {
		t.Fatalf("create import request: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	importResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("perform import: %v", err)
	}
	defer importResp.Body.Close()
	importBody, _ := io.ReadAll(importResp.Body)
	if importResp.StatusCode != http.StatusOK {
		t.Fatalf("import status = %d, body = %s", importResp.StatusCode, importBody)
	}
	result := decodeJSON[ImportResult](t, importBody)
	if result.Imported != 2 || result.Skipped != 1 || result.Total != 3 || len(result.Errors) != 1 {
		t.Fatalf("unexpected import result: %+v", result)
	}

	resp, body = app.request(t, http.MethodGet, "/api/applications?page=1&pageSize=10", nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list after import = %d, body = %s", resp.StatusCode, body)
	}
	list := decodeJSON[ApplicationList](t, body)
	if list.Total != 2 {
		t.Fatalf("imported total = %d, want 2", list.Total)
	}
}

func TestExcelPathImport(t *testing.T) {
	app := newTestApp(t)

	book := excelize.NewFile()
	rows := [][]interface{}{
		{"Category", "Company", "Link", "Status"},
		{"秋招", "路径导入公司", "https://example.com/drop", "笔试"},
	}
	for rowIndex, row := range rows {
		for colIndex, value := range row {
			cell, _ := excelize.CoordinatesToCellName(colIndex+1, rowIndex+1)
			_ = book.SetCellValue("Sheet1", cell, value)
		}
	}
	buffer, err := book.WriteToBuffer()
	if err != nil {
		t.Fatalf("write workbook: %v", err)
	}
	_ = book.Close()

	path := filepath.Join(t.TempDir(), "path-import.xlsx")
	if err := os.WriteFile(path, buffer.Bytes(), 0o644); err != nil {
		t.Fatalf("write workbook file: %v", err)
	}

	info, err := app.server.ProbeExcelFile(path)
	if err != nil {
		t.Fatalf("probe excel: %v", err)
	}
	if info.FileName != "path-import.xlsx" || info.FileSize != int64(len(buffer.Bytes())) {
		t.Fatalf("unexpected excel info: %+v", info)
	}

	result, err := app.server.ImportExcelFromPath(path)
	if err != nil {
		t.Fatalf("import excel path: %v", err)
	}
	if result.Imported != 1 || result.Skipped != 0 || result.Total != 1 {
		t.Fatalf("unexpected path import result: %+v", result)
	}

	invalidPath := filepath.Join(t.TempDir(), "broken.xlsx")
	if err := os.WriteFile(invalidPath, []byte("not an excel workbook"), 0o644); err != nil {
		t.Fatalf("write invalid workbook: %v", err)
	}
	if _, err := app.server.ImportExcelFromPath(invalidPath); err == nil {
		t.Fatalf("invalid workbook should be rejected")
	}
}

func TestShortcutPromptSetting(t *testing.T) {
	app := newTestApp(t)

	if !app.server.ShortcutPromptEnabled() {
		t.Fatal("shortcut prompt should be enabled by default")
	}
	if err := app.server.SetShortcutPromptEnabled(false); err != nil {
		t.Fatalf("disable shortcut prompt: %v", err)
	}
	if app.server.ShortcutPromptEnabled() {
		t.Fatal("shortcut prompt should be disabled")
	}
	if err := app.server.SetShortcutPromptEnabled(true); err != nil {
		t.Fatalf("enable shortcut prompt: %v", err)
	}
	if !app.server.ShortcutPromptEnabled() {
		t.Fatal("shortcut prompt should be enabled again")
	}
}

func TestCreateApplicationUsesServerTimeAndPinned(t *testing.T) {
	app := newTestApp(t)
	before := time.Now().Add(-2 * time.Second)

	resp, body := app.request(t, http.MethodPost, "/api/applications", map[string]any{
		"companyName": "Pinned Corp",
		"category":    "spring",
		"link":        "pinned.example/job",
		"pinned":      true,
	}, "")
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", resp.StatusCode, body)
	}
	created := decodeJSON[models.Application](t, body)
	if !created.Pinned {
		t.Fatal("created application should be pinned")
	}
	if created.AppliedAt.IsZero() ||
		created.AppliedAt.Before(before) ||
		created.AppliedAt.After(time.Now().Add(2*time.Second)) {
		t.Fatalf("appliedAt = %v, want server time near now", created.AppliedAt)
	}

	resp, body = app.request(t, http.MethodPost, "/api/applications", map[string]any{
		"companyName": "Normal Corp",
		"category":    "spring",
	}, "")
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create second status = %d, body = %s", resp.StatusCode, body)
	}

	resp, body = app.request(t, http.MethodGet, "/api/applications?page=1&pageSize=10", nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list status = %d, body = %s", resp.StatusCode, body)
	}
	list := decodeJSON[ApplicationList](t, body)
	if len(list.Items) != 2 || list.Items[0].ID != created.ID {
		t.Fatalf("pinned application should be first: %+v", list.Items)
	}
}

func TestQuickEntryDefaultsFollowServerClock(t *testing.T) {
	app := newTestApp(t)

	resp, body := app.request(t, http.MethodGet, "/api/system/quick-entry/defaults", nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("defaults status = %d, body = %s", resp.StatusCode, body)
	}
	defaults := decodeJSON[struct {
		DefaultCategory string `json:"defaultCategory"`
		ServerTime      string `json:"serverTime"`
	}](t, body)
	if !models.IsValidCategory(defaults.DefaultCategory) {
		t.Fatalf("defaultCategory = %q, want a known category", defaults.DefaultCategory)
	}
	if want := models.DefaultCategoryForTime(time.Now()); defaults.DefaultCategory != want {
		t.Fatalf("defaultCategory = %q, want %q from server clock", defaults.DefaultCategory, want)
	}
	if _, err := time.Parse(time.RFC3339, defaults.ServerTime); err != nil {
		t.Fatalf("serverTime = %q: %v", defaults.ServerTime, err)
	}
}

func TestDefaultCategoryForTime(t *testing.T) {
	cases := []struct {
		month time.Month
		want  string
	}{
		{time.January, models.CategoryInternship},
		{time.February, models.CategorySpring},
		{time.May, models.CategorySpring},
		{time.June, models.CategoryInternship},
		{time.July, models.CategoryInternship},
		{time.August, models.CategoryAutumn},
		{time.November, models.CategoryAutumn},
		{time.December, models.CategoryInternship},
	}
	for _, tc := range cases {
		at := time.Date(2026, tc.month, 15, 12, 0, 0, 0, time.Local)
		if got := models.DefaultCategoryForTime(at); got != tc.want {
			t.Fatalf("DefaultCategoryForTime(%s) = %q, want %q", tc.month, got, tc.want)
		}
	}
}

func TestDatabaseSettingsAndHotSwap(t *testing.T) {
	app := newTestApp(t)
	defer func() {
		if sqlDB, err := app.server.database().DB(); err == nil {
			_ = sqlDB.Close()
		}
	}()

	resp, body := app.request(t, http.MethodGet, "/api/system/database", nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("database settings status = %d, body = %s", resp.StatusCode, body)
	}
	initial := decodeJSON[struct {
		Settings struct {
			Driver     string `json:"driver"`
			SQLitePath string `json:"sqlitePath"`
		} `json:"settings"`
		ConfigPath string `json:"configPath"`
	}](t, body)
	if initial.Settings.Driver != "sqlite" || initial.Settings.SQLitePath != "test.db" {
		t.Fatalf("unexpected initial settings: %+v", initial)
	}
	if initial.ConfigPath == "" {
		t.Fatal("config path should be returned")
	}

	resp, body = app.request(t, http.MethodPost, "/api/system/database/test", map[string]any{
		"driver":     "sqlite",
		"sqlitePath": "test.db",
		"useDsn":     false,
	}, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("test connection status = %d, body = %s", resp.StatusCode, body)
	}

	resp, body = app.request(t, http.MethodPut, "/api/system/database", map[string]any{
		"driver":     "sqlite",
		"sqlitePath": "second.db",
		"useDsn":     false,
	}, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("update database status = %d, body = %s", resp.StatusCode, body)
	}

	raw, err := os.ReadFile(initial.ConfigPath)
	if err != nil {
		t.Fatalf("read config after update: %v", err)
	}
	if !strings.Contains(string(raw), "second.db") {
		t.Fatalf("config should contain the relative sqlite path:\n%s", raw)
	}
	if strings.Contains(string(raw), app.server.cfg.DataDir) {
		t.Fatalf("config must not embed the absolute data directory:\n%s", raw)
	}

	resp, body = app.request(t, http.MethodPost, "/api/applications", map[string]string{
		"companyName": "After Swap",
		"category":    "spring",
		"link":        "no-protocol.example/job",
	}, "")
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create after swap status = %d, body = %s", resp.StatusCode, body)
	}
	created := decodeJSON[models.Application](t, body)
	if created.Link != "no-protocol.example/job" {
		t.Fatalf("link = %q, want the exact entered value", created.Link)
	}

	// Let the previous pool close before the temporary directory is removed.
	time.Sleep(1200 * time.Millisecond)
}

// These types mirror the JSON response shapes used by the tests.
type ApplicationList struct {
	Items []models.Application `json:"items"`
	Total int64                `json:"total"`
}

type Stats struct {
	Total      int64                       `json:"total"`
	Resumes    int64                       `json:"resumes"`
	ByCategory map[string]int64            `json:"byCategory"`
	ByStatus   map[string]int64            `json:"byStatus"`
	Matrix     map[string]map[string]int64 `json:"byCategoryStatus"`
}

type DailyStats struct {
	Daily []struct {
		Date     string `json:"date"`
		Category string `json:"category"`
		Count    int64  `json:"count"`
	} `json:"daily"`
}

func TestResumeLifecycle(t *testing.T) {
	app := newTestApp(t)

	resp, body := app.uploadResume(t, "Backend Resume", `["backend","go"]`, "Backend Resume.PDF", []byte("%PDF-1.4 test"))
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create resume status = %d, body = %s", resp.StatusCode, body)
	}
	created := decodeJSON[models.Resume](t, body)
	if created.ID == 0 || created.Name != "Backend Resume" || created.FileType != "pdf" {
		t.Fatalf("unexpected created resume: %+v", created)
	}
	if len(created.Tags) != 2 || created.Tags[0] != "backend" || created.Tags[1] != "go" {
		t.Fatalf("unexpected resume tags: %+v", created.Tags)
	}
	var stored models.Resume
	if err := app.server.database().First(&stored, created.ID).Error; err != nil {
		t.Fatalf("load stored resume: %v", err)
	}
	if stored.StoredName == "" {
		t.Fatalf("stored name should not be empty")
	}
	storedPath := filepath.Join(app.server.resumeDirectory(), stored.StoredName)
	if _, err := os.Stat(storedPath); err != nil {
		t.Fatalf("resume file was not stored: %v", err)
	}

	resp, body = app.request(t, http.MethodGet, fmt.Sprintf("/api/resume/%d/file", created.ID), nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("file status = %d, body = %s", resp.StatusCode, body)
	}
	if string(body) != "%PDF-1.4 test" {
		t.Fatalf("file content = %q", string(body))
	}
	if contentType := resp.Header.Get("Content-Type"); !strings.HasPrefix(contentType, "application/pdf") {
		t.Fatalf("content type = %q", contentType)
	}

	resp, body = app.request(t, http.MethodPut, fmt.Sprintf("/api/resume/%d", created.ID), map[string]any{
		"pinned": true,
	}, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("pin status = %d, body = %s", resp.StatusCode, body)
	}
	pinned := decodeJSON[models.Resume](t, body)
	if !pinned.Pinned {
		t.Fatalf("resume should be pinned")
	}

	resp, body = app.request(t, http.MethodPut, fmt.Sprintf("/api/resume/%d", created.ID), map[string]any{
		"tags": []string{"senior", "remote"},
	}, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("update tags status = %d, body = %s", resp.StatusCode, body)
	}
	retagged := decodeJSON[models.Resume](t, body)
	if len(retagged.Tags) != 2 || retagged.Tags[0] != "senior" || retagged.Tags[1] != "remote" {
		t.Fatalf("unexpected updated tags: %+v", retagged.Tags)
	}

	resp, body = app.request(t, http.MethodGet, "/api/resumes", nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list status = %d, body = %s", resp.StatusCode, body)
	}
	list := decodeJSON[ResumeList](t, body)
	if list.Total != 1 || len(list.Items) != 1 || list.Items[0].ID != created.ID {
		t.Fatalf("unexpected resume list: %+v", list)
	}

	resp, body = app.request(t, http.MethodGet, "/api/stats", nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("stats status = %d, body = %s", resp.StatusCode, body)
	}
	stats := decodeJSON[Stats](t, body)
	if stats.Resumes != 1 {
		t.Fatalf("resume count = %d, want 1", stats.Resumes)
	}

	resp, body = app.uploadResume(t, "", "", "empty.pdf", nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("empty file status = %d, body = %s", resp.StatusCode, body)
	}

	resp, body = app.uploadResume(t, "", "", "notes.txt", []byte("not allowed"))
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("unsupported type status = %d, body = %s", resp.StatusCode, body)
	}

	resp, body = app.request(t, http.MethodDelete, fmt.Sprintf("/api/resume/%d", created.ID), nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("delete status = %d, body = %s", resp.StatusCode, body)
	}
	if _, err := os.Stat(storedPath); !os.IsNotExist(err) {
		t.Fatalf("resume file should be removed, err = %v", err)
	}
}

func (a *testApp) uploadResume(t *testing.T, name, tags, fileName string, content []byte) (*http.Response, []byte) {
	t.Helper()
	payload := new(bytes.Buffer)
	writer := multipart.NewWriter(payload)
	if name != "" {
		_ = writer.WriteField("name", name)
	}
	if tags != "" {
		_ = writer.WriteField("tags", tags)
	}
	part, err := writer.CreateFormFile("file", fileName)
	if err != nil {
		t.Fatalf("create multipart file: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("write multipart file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, a.http.URL+"/api/resumes", payload)
	if err != nil {
		t.Fatalf("create resume request: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("perform resume request: %v", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read resume response: %v", err)
	}
	return resp, data
}

type ResumeList struct {
	Items []models.Resume `json:"items"`
	Total int64           `json:"total"`
}

func TestResumePathImport(t *testing.T) {
	app := newTestApp(t)

	resumePath := filepath.Join(t.TempDir(), "Drop Resume.pdf")
	if err := os.WriteFile(resumePath, []byte("%PDF-1.4 drop"), 0o644); err != nil {
		t.Fatalf("write drop resume: %v", err)
	}

	info, err := app.server.ProbeResumeFile(resumePath)
	if err != nil {
		t.Fatalf("probe resume: %v", err)
	}
	if info.FileName != "Drop Resume.pdf" || info.FileType != "pdf" || info.FileSize != int64(len("%PDF-1.4 drop")) {
		t.Fatalf("unexpected probe info: %+v", info)
	}

	created, err := app.server.ImportResumeFromPath(resumePath, "", []string{"drop"})
	if err != nil {
		t.Fatalf("import resume: %v", err)
	}
	if created.Name != "Drop Resume" || created.FileType != "pdf" || len(created.Tags) != 1 || created.Tags[0] != "drop" {
		t.Fatalf("unexpected imported resume: %+v", created)
	}
	storedPath := filepath.Join(app.server.resumeDirectory(), created.StoredName)
	data, err := os.ReadFile(storedPath)
	if err != nil || string(data) != "%PDF-1.4 drop" {
		t.Fatalf("imported file mismatch: err = %v, data = %q", err, string(data))
	}

	var count int64
	if err := app.server.database().Model(&models.Resume{}).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("resume count = %d, err = %v", count, err)
	}

	emptyPath := filepath.Join(t.TempDir(), "empty.pdf")
	if err := os.WriteFile(emptyPath, nil, 0o644); err != nil {
		t.Fatalf("write empty resume: %v", err)
	}
	if _, err := app.server.ProbeResumeFile(emptyPath); err == nil {
		t.Fatalf("empty file should be rejected")
	}

	txtPath := filepath.Join(t.TempDir(), "notes.txt")
	if err := os.WriteFile(txtPath, []byte("nope"), 0o644); err != nil {
		t.Fatalf("write txt resume: %v", err)
	}
	if _, err := app.server.ImportResumeFromPath(txtPath, "Bad", nil); err == nil {
		t.Fatalf("unsupported extension should be rejected")
	}
}
