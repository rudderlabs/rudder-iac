package formatter

import (
	"errors"
	"fmt"
	"strings"
)

var (
	DefaultYAML = YAMLFormatter{}
	DefaultText = TextFormatter{}
)

var (
	ErrUnsupportedExtension = errors.New("unsupported extension")
	ErrEmptyExtension       = errors.New("empty extension")
)

type Formatter interface {
	Format(data any) ([]byte, error)
	Extension() []string
}

type Formatters struct {
	formatters map[string]Formatter
}

func Setup(inputs ...Formatter) Formatters {
	formatters := make(map[string]Formatter)

	for _, formatter := range inputs {
		if formatter == nil {
			continue
		}
		for _, ext := range formatter.Extension() {
			formatters[normalizeExtension(ext)] = formatter
		}
	}

	return Formatters{
		formatters: formatters,
	}
}

func (f *Formatters) Format(data any, extension string) ([]byte, error) {
	if extension == "" {
		return nil, fmt.Errorf("%w: extension cannot be empty", ErrEmptyExtension)
	}

	formatter, ok := f.formatters[normalizeExtension(extension)]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedExtension, extension)
	}
	return formatter.Format(data)
}

func normalizeExtension(extension string) string {
	return strings.TrimLeftFunc(strings.ToLower(extension), func(r rune) bool {
		return r == '.'
	})
}
