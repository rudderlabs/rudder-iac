package formatter

import (
	"errors"
)

var (
	ErrUnsupportedExtension = errors.New("unsupported extension")
)

type Formatter interface {
	Format(data any) ([]byte, error)
	Extension() []string
}

type Formatters struct {
	formatters map[string]Formatter
}

func Setup(ip ...Formatter) Formatters {
	formatters := make(map[string]Formatter)
	for _, formatter := range ip {
		for _, ext := range formatter.Extension() {
			formatters[ext] = formatter
		}
	}

	return Formatters{
		formatters: formatters,
	}
}

func (f *Formatters) Format(data any, extension string) ([]byte, error) {
	formatter, ok := f.formatters[extension]
	if !ok {
		return nil, ErrUnsupportedExtension
	}
	return formatter.Format(data)
}
