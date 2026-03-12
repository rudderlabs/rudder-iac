package rules

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"

	vrules "github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

const (
	LanguageJavaScript = "javascript"
	LanguagePython     = "python"

	ExtJavaScript = ".js"
	ExtPython     = ".py"
)

func GetExpectedExtension(language string) string {
	switch language {
	case LanguageJavaScript:
		return ExtJavaScript
	case LanguagePython:
		return ExtPython
	default:
		return ""
	}
}

func ResolveSpecRelativePath(specFilePath, targetPath string) (string, error) {
	if filepath.IsAbs(targetPath) {
		return "", errors.New("path must be relative to the spec file directory")
	}

	if slices.Contains(splitPathSegments(targetPath), "..") {
		return "", errors.New("path must not contain '..' segments")
	}

	return filepath.Join(filepath.Dir(specFilePath), targetPath), nil
}

func splitPathSegments(path string) []string {
	return strings.FieldsFunc(path, func(r rune) bool {
		return r == '/' || r == '\\'
	})
}

func ValidateSpecFile(resolvedPath string) []vrules.ValidationResult {
	info, err := os.Stat(resolvedPath)
	if err != nil {
		return []vrules.ValidationResult{{
			Reference: "/file",
			Message:   "path does not exist or is not accessible",
		}}
	}

	if info.IsDir() {
		return []vrules.ValidationResult{{
			Reference: "/file",
			Message:   "path must be a file",
		}}
	}

	return nil
}
