package server

import (
	"archive/zip"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"resume-helper/internal/appinfo"
)

func TestExportBackupStartsWithVersion(t *testing.T) {
	before := appinfo.Get()
	t.Cleanup(func() {
		appinfo.Apply(before)
	})
	appinfo.Apply(appinfo.Info{Version: "2.0-beta"})

	app := newTestApp(t)
	target := filepath.Join(t.TempDir(), "version.zip")
	if _, err := app.server.ExportBackup(target); err != nil {
		t.Fatalf("export backup: %v", err)
	}

	data := readManifestBytes(t, target)
	if !strings.HasPrefix(string(data), `{"version":"2.0-beta"`) {
		t.Fatalf("manifest should start with version: %s", data)
	}
}

func TestImportRejectsNewerBackupVersion(t *testing.T) {
	before := appinfo.Get()
	t.Cleanup(func() {
		appinfo.Apply(before)
	})
	appinfo.Apply(appinfo.Info{Version: "1.2"})

	app := newTestApp(t)
	source := filepath.Join(t.TempDir(), "source.zip")
	if _, err := app.server.ExportBackup(source); err != nil {
		t.Fatalf("export backup: %v", err)
	}

	newer := filepath.Join(t.TempDir(), "newer.zip")
	rewriteBackupManifest(t, source, newer, func(manifest map[string]any) {
		manifest["version"] = "1.3"
		manifest["appVersion"] = "1.3"
	})
	if _, err := app.server.InspectBackup(newer); err == nil || !strings.Contains(err.Error(), "newer") {
		t.Fatalf("inspect newer backup error = %v, want a version rejection", err)
	}
	if _, err := app.server.ImportBackup(newer); err == nil || !strings.Contains(err.Error(), "1.3") {
		t.Fatalf("import newer backup error = %v, want a version rejection", err)
	}

	older := filepath.Join(t.TempDir(), "older.zip")
	rewriteBackupManifest(t, source, older, func(manifest map[string]any) {
		manifest["version"] = "1.1"
		manifest["appVersion"] = "1.1"
	})
	if _, err := app.server.ImportBackup(older); err != nil {
		t.Fatalf("import older backup: %v", err)
	}
}

func TestImportAcceptsEqualOrOlderBackupVersion(t *testing.T) {
	before := appinfo.Get()
	t.Cleanup(func() {
		appinfo.Apply(before)
	})
	appinfo.Apply(appinfo.Info{Version: "1.2"})

	app := newTestApp(t)
	source := filepath.Join(t.TempDir(), "source.zip")
	if _, err := app.server.ExportBackup(source); err != nil {
		t.Fatalf("export backup: %v", err)
	}

	for _, version := range []string{"1.2", "1.0"} {
		target := filepath.Join(t.TempDir(), "backup-"+version+".zip")
		rewriteBackupManifest(t, source, target, func(manifest map[string]any) {
			manifest["version"] = version
			manifest["appVersion"] = version
		})
		if _, err := app.server.ImportBackup(target); err != nil {
			t.Fatalf("import backup version %s: %v", version, err)
		}
	}
}

func TestImportRejectsDevOrBetaBackupVersion(t *testing.T) {
	before := appinfo.Get()
	t.Cleanup(func() {
		appinfo.Apply(before)
	})
	appinfo.Apply(appinfo.Info{Version: "2.0"})

	app := newTestApp(t)
	source := filepath.Join(t.TempDir(), "source.zip")
	if _, err := app.server.ExportBackup(source); err != nil {
		t.Fatalf("export backup: %v", err)
	}

	for _, version := range []string{"2.0-beta", "1.9-dev"} {
		target := filepath.Join(t.TempDir(), "backup-"+version+".zip")
		rewriteBackupManifest(t, source, target, func(manifest map[string]any) {
			manifest["version"] = version
			manifest["appVersion"] = version
		})
		if _, err := app.server.InspectBackup(target); err != nil {
			t.Fatalf("inspect backup version %s: %v", version, err)
		}
		if _, err := app.server.ImportBackup(target); err == nil || !strings.Contains(err.Error(), "dev/beta") {
			t.Fatalf("import backup version %s error = %v, want a dev/beta rejection", version, err)
		}
	}
}

func TestLegacyNumericBackupVersionFallsBackToAppVersion(t *testing.T) {
	manifest := backupManifest{}
	if err := json.Unmarshal([]byte(`{"version":1,"appVersion":"1.0","format":"resumetrail-backup"}`), &manifest); err != nil {
		t.Fatalf("decode legacy manifest: %v", err)
	}
	if manifest.Version != "" || manifest.AppVersion != "1.0" {
		t.Fatalf("legacy manifest = version %q, appVersion %q", manifest.Version, manifest.AppVersion)
	}
}

func TestCompareAppVersions(t *testing.T) {
	cases := []struct {
		left, right string
		want        int
	}{
		{"1.2", "1.2", 0},
		{"1.2.0", "1.2", 0},
		{"v1.2", "1.2", 0},
		{"1.10", "1.9", 1},
		{"1.1", "1.2", -1},
		{"1.2", "1.2-beta", 1},
		{"1.2-beta", "1.2", -1},
		{"1.2-beta", "1.2-alpha", 1},
		{"1.2.0+build", "1.2.0", 0},
	}
	for _, tc := range cases {
		left, ok := parseAppVersion(tc.left)
		if !ok {
			t.Fatalf("parse %q failed", tc.left)
		}
		right, ok := parseAppVersion(tc.right)
		if !ok {
			t.Fatalf("parse %q failed", tc.right)
		}
		if got := compareAppVersions(left, right); got != tc.want {
			t.Fatalf("compare %q to %q = %d, want %d", tc.left, tc.right, got, tc.want)
		}
	}
}

func readManifestBytes(t *testing.T, path string) []byte {
	t.Helper()
	reader, err := zip.OpenReader(path)
	if err != nil {
		t.Fatalf("open backup: %v", err)
	}
	defer reader.Close()
	for _, file := range reader.File {
		if file.Name != backupManifestFile {
			continue
		}
		handle, err := file.Open()
		if err != nil {
			t.Fatalf("open manifest: %v", err)
		}
		data, err := io.ReadAll(handle)
		_ = handle.Close()
		if err != nil {
			t.Fatalf("read manifest: %v", err)
		}
		return data
	}
	t.Fatal("manifest.json is missing")
	return nil
}

func rewriteBackupManifest(t *testing.T, source, target string, mutate func(map[string]any)) {
	t.Helper()
	reader, err := zip.OpenReader(source)
	if err != nil {
		t.Fatalf("open source backup: %v", err)
	}
	defer reader.Close()

	file, err := os.Create(target)
	if err != nil {
		t.Fatalf("create target backup: %v", err)
	}
	writer := zip.NewWriter(file)
	for _, entry := range reader.File {
		handle, err := entry.Open()
		if err != nil {
			t.Fatalf("open backup entry %s: %v", entry.Name, err)
		}
		data, err := io.ReadAll(handle)
		_ = handle.Close()
		if err != nil {
			t.Fatalf("read backup entry %s: %v", entry.Name, err)
		}
		if entry.Name == backupManifestFile {
			manifest := map[string]any{}
			if err := json.Unmarshal(data, &manifest); err != nil {
				t.Fatalf("decode manifest: %v", err)
			}
			mutate(manifest)
			data, err = json.Marshal(manifest)
			if err != nil {
				t.Fatalf("encode manifest: %v", err)
			}
		}
		header := entry.FileHeader
		output, err := writer.CreateHeader(&header)
		if err != nil {
			t.Fatalf("create backup entry %s: %v", entry.Name, err)
		}
		if _, err := output.Write(data); err != nil {
			t.Fatalf("write backup entry %s: %v", entry.Name, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("finish target backup: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close target backup: %v", err)
	}
}
