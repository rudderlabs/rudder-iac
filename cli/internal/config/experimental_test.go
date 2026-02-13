package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsValidExperimentalFlag(t *testing.T) {
	tests := []struct {
		name     string
		flagName string
		want     bool
	}{
		{
			name:     "valid flag - concurrentSyncs",
			flagName: "concurrentSyncs",
			want:     true,
		},
		{
			name:     "valid flag - v1SpecSupport",
			flagName: "v1SpecSupport",
			want:     true,
		},
		{
			name:     "valid flag - validationFramework",
			flagName: "validationFramework",
			want:     true,
		},
		{
			name:     "valid flag - transformations",
			flagName: "transformations",
			want:     true,
		},
		{
			name:     "invalid flag - nonexistent",
			flagName: "nonexistent",
			want:     false,
		},
		{
			name:     "invalid flag - empty string",
			flagName: "",
			want:     false,
		},
		{
			name:     "invalid flag - wrong case",
			flagName: "Transformations",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidExperimentalFlag(tt.flagName)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetEnvironmentVariableName(t *testing.T) {
	tests := []struct {
		name     string
		flagName string
		want     string
	}{
		{
			name:     "concurrentSyncs",
			flagName: "concurrentSyncs",
			want:     "RUDDERSTACK_X_CONCURRENT_SYNCS",
		},
		{
			name:     "v1SpecSupport",
			flagName: "v1SpecSupport",
			want:     "RUDDERSTACK_X_V1_SPEC_SUPPORT",
		},
		{
			name:     "validationFramework",
			flagName: "validationFramework",
			want:     "RUDDERSTACK_X_VALIDATION_FRAMEWORK",
		},
		{
			name:     "transformations",
			flagName: "transformations",
			want:     "RUDDERSTACK_X_TRANSFORMATIONS",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetEnvironmentVariableName(tt.flagName)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetAvailableExperimentalFlags(t *testing.T) {
	flags := getAvailableExperimentalFlags()

	expectedFlags := []string{
		"concurrentSyncs",
		"v1SpecSupport",
		"validationFramework",
		"transformations",
	}

	assert.ElementsMatch(t, expectedFlags, flags, "should return all available experimental flags")
	assert.Len(t, flags, 4, "should have exactly 4 experimental flags")
}
