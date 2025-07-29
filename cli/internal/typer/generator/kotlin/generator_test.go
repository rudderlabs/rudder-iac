package kotlin_test

import (
	_ "embed"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/typer/generator/kotlin"
	"github.com/rudderlabs/rudder-iac/cli/internal/typer/plan"
	"github.com/stretchr/testify/assert"
)

//go:embed testdata/Main.kt
var mainContents string

func TestGenerate(t *testing.T) {
	plan := &plan.TrackingPlan{
		Name: "Test Plan",
	}
	files, err := kotlin.Generate(plan)
	assert.Len(t, files, 1)
	assert.NoError(t, err)
	assert.NotNil(t, files[0])
	assert.Equal(t, "Main.kt", files[0].Path)
	assert.Contains(t, files[0].Content, mainContents)
}
