// Package tests provides helper utilities for CLI E2E/integration tests.
// This file implements a builder that produces a CLI config.json file at a
// caller-specified location (or a temporary file when no location is given).
// The shape of the JSON mirrors the fields consumed by cli/internal/config.
package tests

import (
	"encoding/json"
	"fmt"
	"os"
)

// ConfigBuilder helps construct a Rudder CLI configuration file programmatically
// for test cases.
//
// Usage example:
//
//	cfg := tests.NewConfigBuilder().
//	        WithAccessToken("tok_123").
//	        WithTelemetryEnabled("write-key", "https://telemetry.example.com").
//	        Build()
//
// The Build method returns the path to the generated config file, a cleanup
// function, and an error if something went wrong.
//
// All methods mutate the builder in-place and return the same instance,
// allowing chaining.
// NOTE: Only the fields relevant to current tests are implemented. Extend as
// needed to cover additional configuration keys.
type ConfigBuilder struct {
	debug              bool
	verbose            bool
	apiURL             string
	accessToken        string
	telemetryDisabled  bool
	telemetryWriteKey  string
	telemetryDataplane string
	telemetryAnonymous string

	filePath string
}

func NewConfigBuilder(path string) *ConfigBuilder {
	apiURL := os.Getenv("CONFIG_BACKEND_URL")
	accessToken := os.Getenv("ACCESS_TOKEN")

	return &ConfigBuilder{
		apiURL:      apiURL,
		accessToken: accessToken,
		filePath:    path,
	}
}

func (b *ConfigBuilder) WithDebug(v bool) *ConfigBuilder { b.debug = v; return b }

func (b *ConfigBuilder) WithVerbose(v bool) *ConfigBuilder { b.verbose = v; return b }

func (b *ConfigBuilder) WithAPIURL(url string) *ConfigBuilder { b.apiURL = url; return b }

func (b *ConfigBuilder) WithAccessToken(token string) *ConfigBuilder {
	b.accessToken = token
	return b
}

func (b *ConfigBuilder) WithTelemetryEnabled(writeKey, dataplaneURL string) *ConfigBuilder {
	b.telemetryDisabled = false
	b.telemetryWriteKey = writeKey
	b.telemetryDataplane = dataplaneURL
	return b
}

func (b *ConfigBuilder) WithTelemetryDisabled() *ConfigBuilder {
	b.telemetryDisabled = true
	return b
}

func (b *ConfigBuilder) WithAnonymousID(id string) *ConfigBuilder {
	b.telemetryAnonymous = id
	return b
}

// Build writes the config.json to the chosen location (or a temp file) and
// returns (path, cleanup, error).
func (b *ConfigBuilder) Build() (string, func() error, error) {

	cfg := map[string]any{
		"debug":   b.debug,
		"verbose": b.verbose,
		"apiURL":  b.apiURL,
		"auth": map[string]any{
			"accessToken": b.accessToken,
		},
		"telemetry": map[string]any{
			"disabled":     b.telemetryDisabled,
			"writeKey":     b.telemetryWriteKey,
			"dataplaneURL": b.telemetryDataplane,
			"anonymousId":  b.telemetryAnonymous,
		},
	}

	content, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return "", nil, fmt.Errorf("marshal config: %w", err)
	}

	path := b.filePath
	if path == "" {
		f, err := os.CreateTemp("", "rudder-cli-config-*.json")
		if err != nil {
			return "", nil, fmt.Errorf("create temp config: %w", err)
		}
		path = f.Name()
		f.Close()
	}

	if err := os.WriteFile(path, content, 0o644); err != nil {
		return "", nil, fmt.Errorf("write config: %w", err)
	}

	cleanup := func() error {
		return os.Remove(path)
	}

	return path, cleanup, nil
}
