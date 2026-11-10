package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

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
	if created.ID == 0 || created.Link != "https://jobs.example.com/1" {
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
	result := decodeJSON[importResult](t, importBody)
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

// These types mirror the JSON response shapes used by the tests.
type ApplicationList struct {
	Items []models.Application `json:"items"`
	Total int64                `json:"total"`
}

type Stats struct {
	Total      int64                       `json:"total"`
	ByCategory map[string]int64            `json:"byCategory"`
	ByStatus   map[string]int64            `json:"byStatus"`
	Matrix     map[string]map[string]int64 `json:"byCategoryStatus"`
}
