package converter

import (
	"crypto/md5"
	"fmt"
	"strings"
)

// StringUtils handles string manipulation utilities
type StringUtils struct{}

// sanitize provides unified string sanitization with different modes
func (su *StringUtils) sanitize(input string, mode SanitizationMode) string {
	if mode == SanitizationModeEvent {
		return su.sanitizeEventID(input)
	}
	return su.sanitizeBasic(input)
}

// sanitizeBasic performs basic string sanitization
func (su *StringUtils) sanitizeBasic(input string) string {
	// Replace non-alphanumeric characters with underscores
	result := strings.ReplaceAll(input, ".", "_")
	result = strings.ReplaceAll(result, "-", "_")
	result = strings.ReplaceAll(result, " ", "_")
	result = strings.ToLower(result)
	return result
}

// sanitizeEventID performs event-specific sanitization with more comprehensive character replacement
func (su *StringUtils) sanitizeEventID(input string) string {
	// Remove or replace problematic characters
	result := input

	// Replace common problematic characters
	replacements := map[string]string{
		"/": "_", "\\": "_", "?": "_", "<": "_", ">": "_", "\"": "_",
		"'": "_", "(": "_", ")": "_", "[": "_", "]": "_", "{": "_",
		"}": "_", "|": "_", "#": "_", "%": "_", "&": "_", "*": "_",
		"+": "_", "=": "_", "@": "_", "!": "_", " ": "_", "\t": "_",
		"\n": "_", "\r": "_",
	}

	for old, new := range replacements {
		result = strings.ReplaceAll(result, old, new)
	}

	// Remove any remaining non-alphanumeric characters except underscores and hyphens
	var clean strings.Builder
	for _, char := range result {
		if (char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '_' || char == '-' {
			clean.WriteRune(char)
		}
	}

	result = clean.String()

	// Remove consecutive underscores
	for strings.Contains(result, "__") {
		result = strings.ReplaceAll(result, "__", "_")
	}

	// Trim underscores from start and end
	result = strings.Trim(result, "_")

	// Ensure lowercase for consistency
	result = strings.ToLower(result)

	return result
}

// generateHash creates consistent hashes with optional prefix
func (su *StringUtils) generateHash(content, prefix string) string {
	var hashInput string
	if prefix != "" {
		hashInput = prefix + "_" + content
	} else {
		hashInput = content
	}

	hash := md5.Sum([]byte(hashInput))
	return fmt.Sprintf("%x", hash)[:8]
}

// ensureUnique resolves naming conflicts using the specified strategy
func (su *StringUtils) ensureUnique(baseName string, usedNames map[string]bool, strategy UniquenessStrategy, maxLength int) string {
	if !usedNames[baseName] {
		usedNames[baseName] = true
		return baseName
	}

	switch strategy {
	case UniquenessStrategyLetterSuffix:
		return su.ensureUniqueWithLetterSuffix(baseName, usedNames, maxLength)
	default:
		return su.ensureUniqueWithCounter(baseName, usedNames)
	}
}

// ensureUniqueWithCounter adds numeric suffixes for uniqueness
func (su *StringUtils) ensureUniqueWithCounter(baseName string, usedNames map[string]bool) string {
	// Check if base name is available first
	if !usedNames[baseName] {
		usedNames[baseName] = true
		return baseName
	}

	counter := 1
	for {
		candidateName := fmt.Sprintf("%s_%d", baseName, counter)
		if !usedNames[candidateName] {
			usedNames[candidateName] = true
			return candidateName
		}
		counter++
	}
}

// ensureUniqueWithLetterSuffix adds letter suffixes for uniqueness
func (su *StringUtils) ensureUniqueWithLetterSuffix(baseName string, usedNames map[string]bool, maxLength int) string {
	letterSuffixes := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M", "N", "O", "P", "Q", "R", "S", "T", "U", "V", "W", "X", "Y", "Z"}

	// Try single letters first
	for _, suffix := range letterSuffixes {
		candidateName := baseName + suffix
		if maxLength > 0 && len(candidateName) > maxLength {
			maxBaseLen := maxLength - len(suffix)
			if maxBaseLen > 3 {
				candidateName = baseName[:maxBaseLen] + suffix
			} else {
				candidateName = "GenType" + suffix
			}
		}

		if !usedNames[candidateName] {
			usedNames[candidateName] = true
			return candidateName
		}
	}

	// Try double letters if needed
	for _, suffix1 := range letterSuffixes {
		for _, suffix2 := range letterSuffixes {
			suffix := suffix1 + suffix2
			candidateName := baseName + suffix
			if maxLength > 0 && len(candidateName) > maxLength {
				maxBaseLen := maxLength - len(suffix)
				if maxBaseLen > 3 {
					candidateName = baseName[:maxBaseLen] + suffix
				} else {
					candidateName = "GenType" + suffix
				}
			}

			if !usedNames[candidateName] {
				usedNames[candidateName] = true
				return candidateName
			}
		}
	}

	// Fallback
	return "UniqueGenType"
}
