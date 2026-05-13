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
