package swift_test

import (
	_ "embed"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/rudderlabs/rudder-iac/cli/internal/typer/generator/core"
	"github.com/rudderlabs/rudder-iac/cli/internal/typer/generator/platforms/swift"
	"github.com/rudderlabs/rudder-iac/cli/internal/typer/plan/testutils"
	"github.com/stretchr/testify/assert"
)

//go:embed testdata/RudderTyper.swift
var rudderTyperSwift string

func TestGenerate(t *testing.T) {
	trackingPlan := testutils.GetReferenceTrackingPlan()

	generator := &swift.Generator{}
	files, err := generator.Generate(trackingPlan, core.GenerateOptions{
		RudderCLIVersion: "1.0.0",
	}, swift.SwiftOptions{})

	assert.NoError(t, err)
	assert.Len(t, files, 1)
	assert.NotNil(t, files[0])
	assert.Equal(t, "RudderTyper.swift", files[0].Path)

	if diff := cmp.Diff(rudderTyperSwift, files[0].Content); diff != "" {
		t.Errorf("generated content does not match testdata/RudderTyper.swift (-want +got):\n%s\nRun 'make typer-swift-update-testdata' to update the golden file.", diff)
	}
}

func TestGenerateWithCustomOutputFileName(t *testing.T) {
	trackingPlan := testutils.GetReferenceTrackingPlan()

	generator := &swift.Generator{}
	files, err := generator.Generate(trackingPlan, core.GenerateOptions{
		RudderCLIVersion: "1.0.0",
	}, swift.SwiftOptions{
		OutputFileName: "Analytics.swift",
	})

	assert.NoError(t, err)
	assert.Len(t, files, 1)
	assert.Equal(t, "Analytics.swift", files[0].Path)
}
