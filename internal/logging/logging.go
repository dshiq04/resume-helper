// Package logging wires zap into the application. Logs are written as JSON to
// "<data dir>/logs/resume-helper-YYYYMMDD.log" and mirrored to stdout in a
// human readable console format. The file is reopened automatically when the
// local date changes so long-running sessions keep writing to the current day.
package logging

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// dailyFile is an io.Writer that appends to a date-named file and rotates at
// midnight without restarting the application.
type dailyFile struct {
	mu     sync.Mutex
	dir    string
	prefix string
	day    string
	path   string
	file   *os.File
}

func newDailyFile(dir, prefix string) (*dailyFile, error) {
	writer := &dailyFile{dir: dir, prefix: prefix}
	if err := writer.rotate(time.Now()); err != nil {
		return nil, err
	}
	return writer, nil
}

func (w *dailyFile) rotate(now time.Time) error {
	day := now.Format("20060102")
	if w.file != nil && day == w.day {
		return nil
	}
	path := filepath.Join(w.dir, w.prefix+day+".log")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	if w.file != nil {
		_ = w.file.Close()
	}
	w.file = file
	w.path = path
	w.day = day
	return nil
}

func (w *dailyFile) Write(payload []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.rotate(time.Now()); err != nil {
		return 0, err
	}
	return w.file.Write(payload)
}

func (w *dailyFile) Sync() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		return nil
	}
	return w.file.Sync()
}

func (w *dailyFile) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		return nil
	}
	err := w.file.Close()
	w.file = nil
	return err
}

func (w *dailyFile) Path() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.path
}

// Setup creates the logs folder and returns a logger, the active file path and
// a cleanup function that flushes and closes the file. Both console and file
// output stay enabled so the same information is available interactively and
// for later inspection.
func Setup(dataDir string) (*zap.Logger, string, func(), error) {
	dir := filepath.Join(dataDir, "logs")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, "", nil, fmt.Errorf("create logs directory %s: %w", dir, err)
	}

	file, err := newDailyFile(dir, "resume-helper-")
	if err != nil {
		return nil, "", nil, fmt.Errorf("open log file: %w", err)
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
	return logger, file.Path(), cleanup, nil
}
