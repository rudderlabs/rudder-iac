package logger

import (
	"io"
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

var logFile io.Writer
var levelVar = new(slog.LevelVar)

func init() {
	homeDir, _ := os.UserHomeDir()
	logFile = openLog(homeDir)
}

// openLog opens ~/.rudder/cli.log for append. When HOME is unset or not
// writable it returns io.Discard: help and usage errors must still work.
func openLog(homeDir string) io.Writer {
	if homeDir == "" {
		return io.Discard
	}
	logPath := filepath.Join(homeDir, ".rudder", "cli.log")
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		return io.Discard
	}
	lf, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return io.Discard
	}
	return lf
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
