package logger

import (
	"log/slog"
	"os"
)

// Writing a wrapper over the slog
// Levels as caller shouldn't know about the library used
// for implementation.
type Level int

const (
	Debug Level = -4
	Info  Level = 0
	Warn  Level = 4
	Error Level = 8
)

func GetLogger(pkgName string) *slog.Logger {
	return GetLoggerWithLevel(pkgName, Warn)
}

func GetLoggerWithLevel(pkgName string, l Level) *slog.Logger {
	h := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
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
		Level: slog.Level(l),
	})

	return slog.New(h.WithAttrs([]slog.Attr{
		{
			Key:   "pkg",
			Value: slog.StringValue(pkgName),
		},
	}))
}
