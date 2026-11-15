package kotlin_test

import (
	_ "embed"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/rudderlabs/rudder-iac/typer/generator/core"
	"github.com/rudderlabs/rudder-iac/typer/generator/platforms/kotlin"
	"github.com/rudderlabs/rudder-iac/typer/plan/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/validator/src/main/kotlin/com/rudderstack/ruddertyper/Main.kt
var mainContent string

//go:embed testdata/validator/src/main/kotlin/com/rudderstack/ruddertyper/composeimmutable/Main.kt
var composeImmutableContent string

//go:embed testdata/validator/src/main/kotlin/com/rudderstack/ruddertyper/common/Main.kt
var commonContent string

func TestGenerate(t *testing.T) {
	// Create a tracking plan with primitive custom types
	trackingPlan := testutils.GetReferenceTrackingPlan()

	generator := &kotlin.Generator{}
	files, err := generator.Generate(trackingPlan, core.GenerateOptions{
		RudderCLIVersion: "1.0.0",
	}, nil)

	assert.NoError(t, err)
	assert.Len(t, files, 1)
	assert.NotNil(t, files[0])
	assert.Equal(t, "Main.kt", files[0].Path)

	// Use go-cmp to provide a nice diff if the content doesn't match
	if diff := cmp.Diff(mainContent, files[0].Content); diff != "" {
		t.Errorf("Generated content does not match expected (-expected +actual):\n%s", diff)
	}
}

func TestGenerateWithCustomPackageName(t *testing.T) {
	trackingPlan := testutils.GetReferenceTrackingPlan()

	generator := &kotlin.Generator{}
	files, err := generator.Generate(trackingPlan, core.GenerateOptions{
		RudderCLIVersion: "1.0.0",
	}, kotlin.KotlinOptions{
		PackageName: "com.example.analytics",
	})

	assert.NoError(t, err)
	assert.Len(t, files, 1)
	assert.NotNil(t, files[0])
	assert.Equal(t, "Main.kt", files[0].Path)

	// Verify that the generated code has the custom package name
	assert.Contains(t, files[0].Content, "package com.example.analytics", "generated code should contain custom package name")
}

func TestGenerateWithInvalidPackageName(t *testing.T) {
	trackingPlan := testutils.GetReferenceTrackingPlan()

	generator := &kotlin.Generator{}
	_, err := generator.Generate(trackingPlan, core.GenerateOptions{
		RudderCLIVersion: "1.0.0",
	}, kotlin.KotlinOptions{
		PackageName: "Com.Example.Analytics", // Invalid: starts with uppercase letters
	})

	assert.Error(t, err, "should fail with invalid package name")
}

func TestGenerateWithCustomOutputFileName(t *testing.T) {
	trackingPlan := testutils.GetReferenceTrackingPlan()

	generator := &kotlin.Generator{}
	files, err := generator.Generate(trackingPlan, core.GenerateOptions{
		RudderCLIVersion: "1.0.0",
	}, kotlin.KotlinOptions{
		OutputFileName: "Events.kt",
	})

	assert.NoError(t, err)
	assert.Len(t, files, 1)
	assert.NotNil(t, files[0])
	assert.Equal(t, "Events.kt", files[0].Path)
}

// Snapshot is committed under the validator project so `make typer-kotlin-validate`
// compiles it against the stub androidx.compose.runtime.Immutable annotation.
func TestGenerateWithComposeImmutable(t *testing.T) {
	trackingPlan := testutils.GetReferenceTrackingPlan()

	generator := &kotlin.Generator{}
	files, err := generator.Generate(trackingPlan, core.GenerateOptions{
		RudderCLIVersion: "1.0.0",
	}, kotlin.KotlinOptions{
		PackageName:      "com.rudderstack.ruddertyper.composeimmutable",
		ComposeImmutable: true,
	})

	assert.NoError(t, err)
	assert.Len(t, files, 1)

	if diff := cmp.Diff(composeImmutableContent, files[0].Content); diff != "" {
		t.Errorf("Generated content does not match expected (-expected +actual):\n%s", diff)
	}
}

// Snapshot is committed under the validator project so `make typer-kotlin-validate` compiles it
// without the RudderStack SDK on the classpath, which is what the common emit mode promises.
func TestGenerateWithEmitCommon(t *testing.T) {
	trackingPlan := testutils.GetReferenceTrackingPlan()

	generator := &kotlin.Generator{}
	files, err := generator.Generate(trackingPlan, core.GenerateOptions{
		RudderCLIVersion: "1.0.0",
	}, kotlin.KotlinOptions{
		PackageName: "com.rudderstack.ruddertyper.common",
		Emit:        kotlin.EmitCommon,
	})

	assert.NoError(t, err)
	assert.Len(t, files, 1)

	if diff := cmp.Diff(commonContent, files[0].Content); diff != "" {
		t.Errorf("Generated content does not match expected (-expected +actual):\n%s", diff)
	}
}

func TestGenerateWithEmitCommonHasNoSDKReferences(t *testing.T) {
	trackingPlan := testutils.GetReferenceTrackingPlan()

	generator := &kotlin.Generator{}
	files, err := generator.Generate(trackingPlan, core.GenerateOptions{
		RudderCLIVersion: "1.0.0",
	}, kotlin.KotlinOptions{Emit: kotlin.EmitCommon})
	require.NoError(t, err)

	content := files[0].Content
	assert.NotContains(t, content, "com.rudderstack.sdk")
	assert.NotContains(t, content, "RudderOption")
	assert.Contains(t, content, "import kotlin.jvm.JvmName")
	assert.Contains(t, content, "class RudderTyper(")
	assert.NotContains(t, content, "class RudderAnalytics(")
}

func TestGenerateWithInvalidEmit(t *testing.T) {
	trackingPlan := testutils.GetReferenceTrackingPlan()

	generator := &kotlin.Generator{}
	_, err := generator.Generate(trackingPlan, core.GenerateOptions{
		RudderCLIVersion: "1.0.0",
	}, kotlin.KotlinOptions{Emit: "models"})

	assert.ErrorContains(t, err, `invalid emit value "models"`)
}
