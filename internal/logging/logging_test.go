package logging

import (
	"os"
	"strings"
	"testing"

	"go.uber.org/zap"
)

func TestSetupCreatesLogFile(t *testing.T) {
	dir := t.TempDir()
	logger, path, closeLog, err := Setup(dir)
	if err != nil {
		t.Fatalf("setup logging: %v", err)
	}
	defer closeLog()
	logger.Info("hello from test", zap.String("source", "unit-test"))
	if err := logger.Sync(); err != nil && !strings.Contains(err.Error(), "invalid argument") {
		t.Fatalf("sync logger: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, `"msg":"hello from test"`) || !strings.Contains(content, `"source":"unit-test"`) {
		t.Fatalf("unexpected log content: %s", content)
	}
}
