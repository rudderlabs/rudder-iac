package logger

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
)

// Writing a wrapper over the slog
// Levels as caller shouldn't know about the library used
// for implementation.
type Level int

type Attr struct {
	Key   string
	Value string
}

var (
	logFile  *os.File
	levelVar = new(slog.LevelVar)
	initOnce sync.Once
	initErr  error
)

// initializeLogger initializes the logger with proper error handling
// This replaces the problematic init() function
func initializeLogger() error {
	initOnce.Do(func() {
		// Try to get home directory, but don't fail if it doesn't work
		homeDir, err := os.UserHomeDir()
		if err != nil {
			// In CI environments or when home directory is not available,
			// we'll use a fallback (discard output or current directory)
			logFile = nil
			initErr = nil // Don't treat this as an error
			return
		}

		// Try to create log directory and file
		logPath := filepath.Join(homeDir, ".rudder", "cli.log")
		if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
			// If we can't create the directory, fall back to discard
			logFile = nil
			initErr = nil
			return
		}

		lf, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			// If we can't open the log file, fall back to discard
			logFile = nil
			initErr = nil
			return
		}

		logFile = lf
	})
	return initErr
}

// getLogWriter returns the appropriate writer for logging
func getLogWriter() io.Writer {
	// Initialize logger if not already done
	_ = initializeLogger()

	if logFile != nil {
		return logFile
	}

	// Fallback: in test environments or when file operations fail,
	// we discard logs to avoid breaking functionality
	return io.Discard
}

type Logger struct {
	*slog.Logger
}

func New(pkgName string, attrs ...Attr) *Logger {
	h := slog.NewTextHandler(getLogWriter(), &slog.HandlerOptions{
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Anything other than time key
			// return
			if a.Key != slog.TimeKey {
				return a
			}
			return slog.String(
				slog.TimeKey,
				a.Value.Time().Format("2006-01-02T15:04:05.000Z"),
			)
		},
		Level: levelVar,
	})

	slogAttrs := []slog.Attr{
		{
			Key:   "pkg",
			Value: slog.StringValue(pkgName),
		},
	}
	for _, attr := range attrs {
		slogAttrs = append(slogAttrs, slog.Attr{
			Key:   attr.Key,
			Value: slog.StringValue(attr.Value),
		})
	}

	return &Logger{slog.New(h.WithAttrs(slogAttrs))}
}

func SetLogLevel(l slog.Level) {
	levelVar.Set(l)
}

// InitializeLogging can be called explicitly to ensure logging is set up
// This is safe to call multiple times
func InitializeLogging() error {
	return initializeLogger()
}
