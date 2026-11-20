package source

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSourceTypesReadTheSpecTypeRule(t *testing.T) {
	t.Parallel()

	assert.Equal(t, []string{
		"java", "dotnet", "php", "flutter", "cordova", "rust", "react_native", "python", "ios",
		"android", "javascript", "go", "node", "ruby", "unity", "ios_swift", "android_kotlin",
	}, SourceTypes())
}
