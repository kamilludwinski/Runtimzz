package logger

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/kamilludwinski/runtimzzz/internal/meta"
)

var defaultLogger *slog.Logger = slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug}))

const logDateLayout = "2006-01-02"

func Init() {
	logDir := meta.LogDir()
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return
	}
	date := time.Now().Format(logDateLayout)
	logPath := filepath.Join(logDir, "runtimz-"+date+".log")
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defaultLogger = slog.New(slog.NewTextHandler(f, &slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: true,
	}))
}

func Debug(msg string, args ...any) {
	defaultLogger.Debug(msg, args...)
}

func Info(msg string, args ...any) {
	defaultLogger.Info(msg, args...)
}

func Warn(msg string, args ...any) {
	defaultLogger.Warn(msg, args...)
}

func Error(msg string, args ...any) {
	defaultLogger.Error(msg, args...)
}

func Logger() *slog.Logger {
	return defaultLogger
}
