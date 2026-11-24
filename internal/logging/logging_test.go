package logging

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestDailyFileRotatesWhenDateChanges(t *testing.T) {
	dir := t.TempDir()
	yesterday := time.Now().AddDate(0, 0, -1)
	stalePath := filepath.Join(dir, "resume-helper-"+yesterday.Format("20060102")+".log")
	file, err := os.OpenFile(stalePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		t.Fatalf("open stale log file: %v", err)
	}
	writer := &dailyFile{
		dir:    dir,
		prefix: "resume-helper-",
		day:    yesterday.Format("20060102"),
		path:   stalePath,
		file:   file,
	}
	defer writer.Close()

	// The next write must notice that the local date changed and reopen the
	// file for today instead of appending to yesterday's log.
	if _, err := writer.Write([]byte("after\n")); err != nil {
		t.Fatalf("write after midnight: %v", err)
	}
	todayPath := filepath.Join(dir, "resume-helper-"+time.Now().Format("20060102")+".log")
	if writer.Path() != todayPath {
		t.Fatalf("log file = %s, want %s", writer.Path(), todayPath)
	}
	data, err := os.ReadFile(todayPath)
	if err != nil {
		t.Fatalf("read rotated file: %v", err)
	}
	if !strings.Contains(string(data), "after") {
		t.Fatalf("rotated file does not contain the new entry: %s", data)
	}
}

func TestSetupCreatesLogFile(t *testing.T) {
	dir := t.TempDir()
	logger, path, closeLog, err := Setup(dir)
	if err != nil {
		t.Fatalf("setup logging: %v", err)
	}
	defer closeLog()
	logger.Info("hello from test", zap.String("source", "unit-test"))
	logger.Debug("debug from test")
	if err := logger.Sync(); err != nil && !strings.Contains(err.Error(), "invalid argument") {
		t.Fatalf("sync logger: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, `"level":"INFO"`) ||
		!strings.Contains(content, `"level":"DEBUG"`) ||
		!strings.Contains(content, `"msg":"hello from test"`) ||
		!strings.Contains(content, `"msg":"debug from test"`) ||
		!strings.Contains(content, `"source":"unit-test"`) {
		t.Fatalf("unexpected log content: %s", content)
	}
}
