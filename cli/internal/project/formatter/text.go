package formatter

import "fmt"

// TextFormatter formats string content as raw bytes for text-based code files.
// This is used for transformation and library code files (.js, .py) during import.
type TextFormatter struct{}

// Format converts string content to bytes without any formatting.
// Since the API only returns valid code, no syntax validation is performed.
func (f TextFormatter) Format(data any) ([]byte, error) {
	str, ok := data.(string)
	if !ok {
		return nil, fmt.Errorf("expected string, got %T", data)
	}
	return []byte(str), nil
}

// Extension returns the file extensions this formatter handles.
func (f TextFormatter) Extension() []string {
	return []string{"js", "py"}
}
