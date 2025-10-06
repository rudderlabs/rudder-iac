package formatter

import (
	"errors"
	"fmt"
)

var (
	YAML = YAMLFormatter{}
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
			formatters[ext] = formatter
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

	formatter, ok := f.formatters[extension]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedExtension, extension)
	}
	return formatter.Format(data)
}
