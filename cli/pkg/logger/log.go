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

type Attr struct {
	Key   string
	Value string
}

func New(pkgName string, attrs ...Attr) *slog.Logger {
	return NewWithLevel(pkgName, Info, attrs...)
}

func NewWithLevel(pkgName string, l Level, attrs ...Attr) *slog.Logger {
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

	return slog.New(h.WithAttrs(slogAttrs))
}
