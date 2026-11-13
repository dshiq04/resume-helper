// Package logging wires zap into the application. Logs are written as JSON to
// "<data dir>/logs/resume-helper-YYYYMMDD.log" and mirrored to stdout in a
// human readable console format.
package logging

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Setup creates the logs folder and returns a logger, the active file path and
// a cleanup function that flushes and closes the file. Both console and file
// output stay enabled so the same information is available interactively and
// for later inspection.
func Setup(dataDir string) (*zap.Logger, string, func(), error) {
	dir := filepath.Join(dataDir, "logs")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, "", nil, fmt.Errorf("create logs directory %s: %w", dir, err)
	}

	filePath := filepath.Join(dir, "resume-helper-"+time.Now().Format("20060102")+".log")
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, "", nil, fmt.Errorf("open log file %s: %w", filePath, err)
	}

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "time"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	encoderConfig.EncodeCaller = zapcore.ShortCallerEncoder

	fileCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(file),
		zap.DebugLevel,
	)
	consoleCore := zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderConfig),
		zapcore.AddSync(os.Stdout),
		zap.DebugLevel,
	)

	logger := zap.New(zapcore.NewTee(fileCore, consoleCore), zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
	cleanup := func() {
		_ = logger.Sync()
		_ = file.Close()
	}
	return logger, filePath, cleanup, nil
}
