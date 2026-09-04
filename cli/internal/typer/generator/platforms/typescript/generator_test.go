package typescript_test

import (
	_ "embed"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/rudderlabs/rudder-iac/cli/internal/typer/generator/core"
	"github.com/rudderlabs/rudder-iac/cli/internal/typer/generator/platforms/typescript"
	"github.com/rudderlabs/rudder-iac/cli/internal/typer/plan"
	"github.com/rudderlabs/rudder-iac/cli/internal/typer/plan/testutils"
	"github.com/stretchr/testify/assert"
)

//go:embed testdata/RudderTyper.ts
var rudderTyperTS string

//go:embed testdata/IdentitySections.ts
var identitySectionsTS string

//go:embed testdata/EmptyIdentity.ts
var emptyIdentityTS string

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

// TestGenerateAuxiliaryGoldens keeps the clients the wire-contract suite runs
// against in step with the generator. They are generated from their own plans
// because identify and group are singletons — one shape each per plan — so the
// identity-section matrix and the empty-schema path cannot live in the main
// reference plan.
func TestGenerateAuxiliaryGoldens(t *testing.T) {
	for _, tc := range []struct {
		name       string
		plan       *plan.TrackingPlan
		outputFile string
		want       string
	}{
		{"identity sections", testutils.GetIdentitySectionsPlan(), "IdentitySections.ts", identitySectionsTS},
		{"empty identity", testutils.GetEmptyIdentityPlan(), "EmptyIdentity.ts", emptyIdentityTS},
	} {
		t.Run(tc.name, func(t *testing.T) {
			generator := &typescript.Generator{}
			files, err := generator.Generate(tc.plan, core.GenerateOptions{
				RudderCLIVersion: "1.0.0",
			}, typescript.TypeScriptOptions{OutputFileName: tc.outputFile})

			assert.NoError(t, err)
			assert.Len(t, files, 1)

			if diff := cmp.Diff(tc.want, files[0].Content); diff != "" {
				t.Errorf("generated content does not match testdata/%s (-want +got):\n%s\nRun 'make typer-typescript-update-testdata' to update the golden files.", tc.outputFile, diff)
			}
		})
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
	assert.Contains(t, file.Content, "        this.analytics.group(\n            groupId,\n            undefined,\n            this.withRudderTyperContext(options),\n            callback,\n        );")
	assert.NotContains(t, file.Content, "if (typeof", "single branch must not emit if/else")
}

// TestGenerateResolvesAnalyticsLazily guards against regressing to eager instance
// capture. The class must accept a resolver-only constructor and re-resolve the
// instance on every call (via the `analytics` getter) so a lazily-loaded browser
// SDK is picked up instead of the preloader being captured at construction and
// events dropped. The instance form is intentionally not accepted so the unsafe
// call cannot compile.
func TestGenerateResolvesAnalyticsLazily(t *testing.T) {
	trackingPlan := testutils.GetReferenceTrackingPlan()

	generator := &typescript.Generator{}
	files, err := generator.Generate(trackingPlan, core.GenerateOptions{RudderCLIVersion: "1.0.0"}, typescript.TypeScriptOptions{})
	assert.NoError(t, err)
	assert.Len(t, files, 1)
	content := files[0].Content

	assert.Contains(t, content, "constructor(resolveAnalytics: () => RudderAnalytics)")
	assert.Contains(t, content, "this.resolveAnalytics = resolveAnalytics;")
	assert.Contains(t, content, "private get analytics(): RudderAnalytics {")
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
