package telemetry

import (
	"encoding/json"
	"os"

	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/telemetry"
)

const (
	CommandExecutedEvent = "CLI Command Executed"
)

type KV struct {
	K string
	V interface{}
}

// isRunningInCI detects whether the CLI is running in a CI/CD environment.
// It checks for common CI/CD environment variables across multiple platforms.
func isRunningInCI() bool {
	return os.Getenv("CI") == "true" || os.Getenv("TF_BUILD") == "true"
}

// getCIPlatform returns the name of the CI/CD platform if running in CI.
func getCIPlatform() string {
	ciPlatforms := map[string]string{
		"GITHUB_ACTIONS":         "github-actions",
		"GITLAB_CI":              "gitlab-ci",
		"CIRCLECI":               "circle-ci",
		"TF_BUILD":               "azure-pipelines",
		"JENKINS_URL":            "jenkins",
		"TRAVIS":                 "travis-ci",
	}

	for envVar, platform := range ciPlatforms {
		if os.Getenv(envVar) != "" {
			return platform
		}
	}

	return "unknown"
}

func TrackCommand(command string, err error, extras ...KV) {

	props := map[string]interface{}{
		"command": command,
		"errored": err != nil,
	}

	for _, extra := range extras {
		props[extra.K] = extra.V
	}

	// Automatically add experimental flags
	cfg := config.GetConfig()
	experimentalData, _ := json.Marshal(cfg.ExperimentalFlags)
	var experimental map[string]interface{}
	json.Unmarshal(experimentalData, &experimental)
	props["experimental"] = experimental

	// Automatically add execution context (CI)
	if isRunningInCI() {
		props["is_ci"] = true
		props["ci_platform"] = getCIPlatform()
	}

	if err := telemetry.TrackEvent(CommandExecutedEvent, props); err != nil {
		log.Error("failed to track command", "error", err)
	}
}
