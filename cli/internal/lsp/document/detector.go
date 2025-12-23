// Package document provides rudder file detection logic
package document

import (
	"strings"
)

// IsRudderFile checks if the given content contains the rudder version marker

func IsRudderFile(content []byte) bool {
	return strings.Contains(string(content), "version: rudder/v0.1")
}
