package server

import (
	"archive/zip"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"resume-helper/internal/models"
)

func TestAPIRoutesRequireProcessToken(t *testing.T) {
	app := newTestApp(t)

	request := func(path string, token bool) *http.Response {
		t.Helper()
		req, err := http.NewRequest(http.MethodGet, app.http.URL+path, nil)
		if err != nil {
			t.Fatalf("create request: %v", err)
		}
		if token {
			req.Header.Set("X-ResumeTrail-Token", app.server.APIToken())
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("perform request: %v", err)
		}
		t.Cleanup(func() { _ = resp.Body.Close() })
		return resp
	}

	if resp := request("/api/health", false); resp.StatusCode != http.StatusOK {
		t.Fatalf("health without token status = %d, want 200", resp.StatusCode)
	}
	if resp := request("/api/applications", false); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("api without token status = %d, want 403", resp.StatusCode)
	}
	if resp := request("/api/applications", true); resp.StatusCode != http.StatusOK {
		t.Fatalf("api with token status = %d, want 200", resp.StatusCode)
	}
}

func TestAPIRejectsUnknownOrigin(t *testing.T) {
	app := newTestApp(t)
	req, err := http.NewRequest(http.MethodGet, app.http.URL+"/api/applications", nil)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	req.Header.Set("Origin", "https://evil.example")
	req.Header.Set("X-ResumeTrail-Token", app.server.APIToken())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("unknown origin status = %d, want 403", resp.StatusCode)
	}
	if origin := resp.Header.Get("Access-Control-Allow-Origin"); origin != "" {
		t.Fatalf("unknown origin received ACAO %q", origin)
	}
}

func TestLoginIsThrottledAfterRepeatedFailures(t *testing.T) {
	app := newTestApp(t)
	resp, body := app.request(t, http.MethodPost, "/api/auth/setup", map[string]string{"password": "secret123"}, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("setup status = %d, body = %s", resp.StatusCode, body)
	}
	for attempt := 0; attempt < maxLoginFailures; attempt++ {
		app.request(t, http.MethodPost, "/api/auth/login", map[string]string{"password": "wrong"}, "")
	}
	resp, _ = app.request(t, http.MethodPost, "/api/auth/login", map[string]string{"password": "secret123"}, "")
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("locked login status = %d, want 429", resp.StatusCode)
	}
}

func TestFunnelDoesNotCountRejectedAsInterview(t *testing.T) {
	app := newTestApp(t)
	resp, body := app.request(t, http.MethodPost, "/api/applications", map[string]string{
		"companyName": "Rejected Corp",
		"category":    "autumn",
		"status":      string(models.StatusRejected),
	}, "")
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", resp.StatusCode, body)
	}

	resp, body = app.request(t, http.MethodGet, "/api/stats/funnel", nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("funnel status = %d, body = %s", resp.StatusCode, body)
	}
	funnel := decodeJSON[funnelResponse](t, body)
	if funnel.InterviewCount != 0 {
		t.Fatalf("interview count = %d, want 0 for a screening rejection", funnel.InterviewCount)
	}
	if funnel.OfferCount != 0 {
		t.Fatalf("offer count = %d, want 0", funnel.OfferCount)
	}
	for _, stage := range funnel.Stages {
		if stage.Key == models.StatusInterview && stage.Count != 0 {
			t.Fatalf("interview stage = %d, want 0", stage.Count)
		}
	}
}

func TestFunnelIgnoresRolledBackOffer(t *testing.T) {
	app := newTestApp(t)
	_, body := app.request(t, http.MethodPost, "/api/applications", map[string]string{
		"companyName": "Rollback Corp",
		"category":    "spring",
		"status":      string(models.StatusScreening),
	}, "")
	created := decodeJSON[applicationStatusView](t, body).Application
	path := "/api/application/" + uintString(created.ID)

	for _, status := range []string{models.StatusOffer, models.StatusScreening} {
		resp, body := app.request(t, http.MethodPut, path, map[string]string{"status": status}, "")
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("update to %s status = %d, body = %s", status, resp.StatusCode, body)
		}
	}

	resp, body := app.request(t, http.MethodGet, "/api/stats/funnel", nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("funnel status = %d, body = %s", resp.StatusCode, body)
	}
	funnel := decodeJSON[funnelResponse](t, body)
	if funnel.OfferCount != 0 {
		t.Fatalf("offer count = %d, want 0 after rolling back to screening", funnel.OfferCount)
	}
}

func TestStatusDaysRestartAfterRollingBack(t *testing.T) {
	app := newTestApp(t)
	_, body := app.request(t, http.MethodPost, "/api/applications", map[string]string{
		"companyName": "Days Corp",
		"category":    "internship",
		"status":      string(models.StatusScreening),
	}, "")
	created := decodeJSON[applicationStatusView](t, body).Application
	path := "/api/application/" + uintString(created.ID)

	for _, status := range []string{models.StatusInterview, models.StatusScreening} {
		resp, body := app.request(t, http.MethodPut, path, map[string]string{"status": status}, "")
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("update to %s status = %d, body = %s", status, resp.StatusCode, body)
		}
	}

	resp, body := app.request(t, http.MethodGet, path+"/history", nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("history status = %d, body = %s", resp.StatusCode, body)
	}
	history := decodeJSON[applicationHistoryResponse](t, body)
	if history.ScreeningDays != 0 {
		t.Fatalf("screening days = %d, want 0 right after re-entering screening", history.ScreeningDays)
	}
	if len(history.Items) == 0 || history.Items[len(history.Items)-1].Status != models.StatusScreening {
		t.Fatalf("timeline should end at screening: %+v", history.Items)
	}
}

func TestStatsCategoryFilterAppliesToKPIAndFunnel(t *testing.T) {
	app := newTestApp(t)
	for _, entry := range []struct {
		company  string
		category string
		status   string
	}{
		{"Spring Corp", "spring", string(models.StatusScreening)},
		{"Autumn Corp", "autumn", string(models.StatusRejected)},
	} {
		resp, body := app.request(t, http.MethodPost, "/api/applications", map[string]string{
			"companyName": entry.company,
			"category":    entry.category,
			"status":      entry.status,
		}, "")
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("create %s status = %d, body = %s", entry.company, resp.StatusCode, body)
		}
	}

	resp, body := app.request(t, http.MethodGet, "/api/stats?category=spring", nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("stats status = %d, body = %s", resp.StatusCode, body)
	}
	var stats struct {
		Total int64 `json:"total"`
	}
	stats = decodeJSON[struct {
		Total int64 `json:"total"`
	}](t, body)
	if stats.Total != 1 {
		t.Fatalf("category-scoped total = %d, want 1", stats.Total)
	}

	resp, body = app.request(t, http.MethodGet, "/api/stats/funnel?category=spring", nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("funnel status = %d, body = %s", resp.StatusCode, body)
	}
	funnel := decodeJSON[funnelResponse](t, body)
	if funnel.Total != 1 {
		t.Fatalf("category-scoped funnel total = %d, want 1", funnel.Total)
	}
}

func TestImportBackupFindsNestedManifest(t *testing.T) {
	app := newTestApp(t)
	app.request(t, http.MethodPost, "/api/applications", map[string]string{
		"companyName": "Nested Corp",
		"category":    "social",
	}, "")

	source := filepath.Join(t.TempDir(), "source.zip")
	if _, err := app.server.ExportBackup(source); err != nil {
		t.Fatalf("export backup: %v", err)
	}

	nested := filepath.Join(t.TempDir(), "nested.zip")
	reader, err := zip.OpenReader(source)
	if err != nil {
		t.Fatalf("open source backup: %v", err)
	}
	defer reader.Close()
	file, err := os.Create(nested)
	if err != nil {
		t.Fatalf("create nested backup: %v", err)
	}
	writer := zip.NewWriter(file)
	for _, entry := range reader.File {
		handle, err := entry.Open()
		if err != nil {
			t.Fatalf("open entry %s: %v", entry.Name, err)
		}
		content, err := io.ReadAll(handle)
		_ = handle.Close()
		if err != nil {
			t.Fatalf("read entry %s: %v", entry.Name, err)
		}
		header := entry.FileHeader
		header.Name = "backup/" + entry.Name
		target, err := writer.CreateHeader(&header)
		if err != nil {
			t.Fatalf("create nested entry %s: %v", header.Name, err)
		}
		if _, err := target.Write(content); err != nil {
			t.Fatalf("write nested entry %s: %v", header.Name, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close nested backup: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close nested backup file: %v", err)
	}

	if _, err := app.server.ImportBackup(nested); err != nil {
		t.Fatalf("import nested backup: %v", err)
	}
}

func TestFindImportHeaderPrefersExactCompanyColumn(t *testing.T) {
	rows := [][]string{{"Company Size", "Company Name", "Link", "Status"}}
	rowIndex, columns := findImportHeader(rows)
	if rowIndex != 0 {
		t.Fatalf("header row = %d, want 0", rowIndex)
	}
	if columns.Company != 1 {
		t.Fatalf("company column = %d, want 1 (Company Name)", columns.Company)
	}
	if columns.Link != 2 || columns.Status != 3 {
		t.Fatalf("unexpected columns: %+v", columns)
	}
}

func TestParseCategoryAndStatusDoNotUseArbitraryPrefixes(t *testing.T) {
	if _, ok := parseCategory("international"); ok {
		t.Fatal("international must not be parsed as internship")
	}
	if _, ok := parseStatus("interviewed"); ok {
		t.Fatal("interviewed must not be parsed as interview")
	}
	if category, ok := parseCategory("Spring 2026"); !ok || category != models.CategorySpring {
		t.Fatalf("Spring 2026 = %q, %v", category, ok)
	}
	if category, ok := parseCategory("internship2026"); !ok || category != models.CategoryInternship {
		t.Fatalf("internship2026 = %q, %v", category, ok)
	}
	if status, ok := parseStatus("初筛"); !ok || status != models.StatusScreening {
		t.Fatalf("初筛 = %q, %v", status, ok)
	}
}

func TestBackupPathWithTopLevelFolderHasNoDoubleSeparators(t *testing.T) {
	// A regression guard for the nested-manifest test: normalized entry names
	// must not keep a leading slash after prefixing.
	if got := normalizeBackupEntryName("backup\\manifest.json"); got != "backup/manifest.json" {
		t.Fatalf("normalized name = %q", got)
	}
	if !strings.HasSuffix(normalizeBackupEntryName("/backup/manifest.json"), "backup/manifest.json") {
		t.Fatal("leading slash was not removed")
	}
}
