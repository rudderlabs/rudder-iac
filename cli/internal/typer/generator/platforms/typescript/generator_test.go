package typescript_test

import (
	_ "embed"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/rudderlabs/rudder-iac/cli/internal/typer/generator/core"
	"github.com/rudderlabs/rudder-iac/cli/internal/typer/generator/platforms/typescript"
	"github.com/rudderlabs/rudder-iac/cli/internal/typer/plan/testutils"
	"github.com/stretchr/testify/assert"
)

//go:embed testdata/RudderTyper.ts
var rudderTyperTS string

func TestGenerate(t *testing.T) {
	trackingPlan := testutils.GetReferenceTrackingPlan()

	generator := &typescript.Generator{}
	files, err := generator.Generate(trackingPlan, core.GenerateOptions{
		RudderCLIVersion: "1.0.0",
	}, typescript.TypeScriptOptions{})

	assert.NoError(t, err)
	assert.Len(t, files, 1)
	assert.NotNil(t, files[0])
	assert.Equal(t, "RudderTyper.ts", files[0].Path)

	if diff := cmp.Diff(rudderTyperTS, files[0].Content); diff != "" {
		t.Errorf("generated content does not match testdata/RudderTyper.ts (-want +got):\n%s\nRun 'make typer-typescript-update-testdata' to update the golden file.", diff)
	}
}

func TestGenerateSingleBranchDispatcher(t *testing.T) {
	ctx := &typescript.TSContext{
		EventContext:     map[string]string{"platform": `"typescript"`},
		UsesApiCallback:  true,
		TrackingPlanName: "test",
		TrackingPlanID:   "tp_1",
		AnalyticsMethods: []typescript.TSAnalyticsMethod{{
			Name:          "group",
			Comment:       "test",
			SDKMethodName: "group",
			Overloads: []typescript.TSOverloadSignature{{
				Arguments: []typescript.TSMethodArgument{
					{Name: "groupId", Type: "string"},
					{Name: "options", Type: "ApiOptions", Optional: true},
					{Name: "callback", Type: "ApiCallback", Optional: true},
				},
			}},
			MethodArguments: []typescript.TSMethodArgument{
				{Name: "groupId", Type: "string"},
				{Name: "options", Type: "ApiOptions", Optional: true},
				{Name: "callback", Type: "ApiCallback", Optional: true},
			},
			DispatcherBranches: []typescript.TSDispatcherBranch{{
				SDKArguments: []typescript.TSSDKArgument{
					{Value: "groupId"},
					{Value: "undefined"},
					{Value: "this.withRudderTyperContext(options)"},
					{Value: "callback"},
				},
			}},
		}},
	}

	file, err := typescript.GenerateFile("test.ts", ctx)
	assert.NoError(t, err)
	assert.Contains(t, file.Content, "        const analytics = this.resolveAnalytics();\n        if (analytics === null || analytics === undefined) {\n            return;\n        }\n        analytics.group(\n            groupId,\n            undefined,\n            this.withRudderTyperContext(options),\n            callback,\n        );")
	assert.NotContains(t, file.Content, "this.analytics.group", "must dispatch through the per-call local analytics variable")
	assert.NotContains(t, file.Content, "if (typeof", "single branch must not emit if/else")
}

// TestGenerateResolvesAnalyticsLazily guards against regressing to eager instance
// capture. The class must accept a resolver-only constructor and re-resolve the
// instance inside each generated method so a lazily-loaded browser SDK is picked
// up instead of the preloader being captured at construction and events dropped.
// The instance form is intentionally not accepted so the unsafe call cannot
// compile.
func TestGenerateResolvesAnalyticsLazily(t *testing.T) {
	trackingPlan := testutils.GetReferenceTrackingPlan()

	generator := &typescript.Generator{}
	files, err := generator.Generate(trackingPlan, core.GenerateOptions{RudderCLIVersion: "1.0.0"}, typescript.TypeScriptOptions{})
	assert.NoError(t, err)
	assert.Len(t, files, 1)
	content := files[0].Content

	assert.Contains(t, content, "constructor(resolveAnalytics: () => RudderAnalytics | null | undefined)")
	assert.Contains(t, content, "private readonly resolveAnalytics: () => RudderAnalytics | null | undefined;")
	assert.Contains(t, content, "this.resolveAnalytics = resolveAnalytics;")
	assert.Contains(t, content, "const analytics = this.resolveAnalytics();\n        if (analytics === null || analytics === undefined) {\n            return;\n        }")
	assert.Contains(t, content, "analytics.track(\n            \"User Signed Up\",")
	assert.Contains(t, content, "analytics.identify(")
	assert.NotContains(t, content, "private get analytics", "must not store the resolved SDK beyond a method call")
	assert.NotContains(t, content, "this.analytics = analytics;", "must not capture the instance eagerly")
	assert.NotContains(t, content, "RudderAnalytics | (() => RudderAnalytics)", "must not accept the unsafe instance form")
}

func TestGenerateWithCustomOutputFileName(t *testing.T) {
	trackingPlan := testutils.GetReferenceTrackingPlan()

	generator := &typescript.Generator{}
	files, err := generator.Generate(trackingPlan, core.GenerateOptions{
		RudderCLIVersion: "1.0.0",
	}, typescript.TypeScriptOptions{
		OutputFileName: "Analytics.ts",
	})

	assert.NoError(t, err)
	assert.Len(t, files, 1)
	assert.Equal(t, "Analytics.ts", files[0].Path)
}
