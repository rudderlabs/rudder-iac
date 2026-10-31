package telemetry

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// clearCIEnv empties every variable the CLI reads for the CI context, so the
// test sees the same environment on a laptop and in the repository's own CI.
func clearCIEnv(t *testing.T) {
	t.Helper()
	t.Setenv("RUDDERSTACK_CLI_WORKFLOW_VERSION", "")
	t.Setenv("RUDDERSTACK_CLI_CI_PLATFORM", "")
	t.Setenv("CI", "")
	for _, ci := range ciPlatforms {
		t.Setenv(ci.envVar, "")
	}
}

func TestGetCIExecutionContext(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want map[string]interface{}
	}{
		{
			name: "outside CI",
			env:  map[string]string{},
			want: map[string]interface{}{},
		},
		{
			name: "the official workflow sets both variables",
			env: map[string]string{
				"RUDDERSTACK_CLI_WORKFLOW_VERSION": "v1.2.0",
				"RUDDERSTACK_CLI_CI_PLATFORM":      "github-actions",
				"GITHUB_ACTIONS":                   "true",
			},
			want: map[string]interface{}{"cli_workflow_version": "v1.2.0", "ci_platform": "github-actions"},
		},
		{
			name: "the variable wins over detection",
			env:  map[string]string{"RUDDERSTACK_CLI_CI_PLATFORM": "custom", "GITLAB_CI": "true"},
			want: map[string]interface{}{"ci_platform": "custom"},
		},
		{
			name: "a GitHub Actions job without the official workflow",
			env:  map[string]string{"GITHUB_ACTIONS": "true"},
			want: map[string]interface{}{"ci_platform": "github-actions"},
		},
		{
			name: "a GitLab job",
			env:  map[string]string{"GITLAB_CI": "true", "CI": "true"},
			want: map[string]interface{}{"ci_platform": "gitlab-ci"},
		},
		{
			name: "Jenkins sets a URL, not a flag",
			env:  map[string]string{"JENKINS_URL": "https://jenkins.example.com/"},
			want: map[string]interface{}{"ci_platform": "jenkins"},
		},
		{
			name: "CI alone is not taken as a CI service",
			env:  map[string]string{"CI": "true"},
			want: map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearCIEnv(t)
			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			assert.Equal(t, tt.want, getCIExecutionContext())
		})
	}
}
