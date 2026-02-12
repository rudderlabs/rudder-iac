package config

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsValidExperimentalFlag_InvalidFlags(t *testing.T) {
	t.Parallel()
	assert.False(t, IsValidExperimentalFlag("invalidFlag"), "should return false for an invalid flag")

}

func TestGetEnvironmentVariableName(t *testing.T) {
	t.Parallel()

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
			t.Parallel()
			got := GetEnvironmentVariableName(tt.flagName)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExperimentalConfigStructInvariants(t *testing.T) {
	t.Parallel()

	expType := reflect.TypeOf(ExperimentalConfig{})

	assert.Greater(t, expType.NumField(), 0, "should have at least one experimental flag")

	for i := 0; i < expType.NumField(); i++ {
		field := expType.Field(i)

		t.Run(field.Name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, reflect.Bool, field.Type.Kind(),
				"experimental flag %s must be a bool", field.Name)

			tag := field.Tag.Get("mapstructure")
			assert.NotEmpty(t, tag,
				"experimental flag %s must have a mapstructure tag", field.Name)

			assert.True(t, IsValidExperimentalFlag(tag),
				"experimental flag %s with tag %q must be recognized as valid", field.Name, tag)
		})
	}
}

func TestGetAvailableExperimentalFlags(t *testing.T) {
	t.Parallel()

	flags := getAvailableExperimentalFlags()
	expType := reflect.TypeOf(ExperimentalConfig{})

	assert.Len(t, flags, expType.NumField(),
		"every ExperimentalConfig field should produce a flag")
}
