package logger

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

// Writing a wrapper over the slog
// Levels as caller shouldn't know about the library used
// for implementation.
type Level int

type Attr struct {
	Key   string
	Value string
}

var logFile *os.File
var levelVar = new(slog.LevelVar)

func init() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("Error getting home directory: %v\n", err)
		os.Exit(1)
	}

	logPath := filepath.Join(homeDir, ".rudder", "cli.log")
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		fmt.Printf("Error creating log directory: %v\n", err)
		os.Exit(1)
	}

	lf, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Error opening log file: %v\n", err)
		os.Exit(1)
	}

	logFile = lf
}

type Logger struct {
	*slog.Logger
}

func New(pkgName string, attrs ...Attr) *Logger {
	h := slog.NewTextHandler(logFile, &slog.HandlerOptions{
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
