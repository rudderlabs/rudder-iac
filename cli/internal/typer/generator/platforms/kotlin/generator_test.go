package kotlin_test

import (
	_ "embed"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/rudderlabs/rudder-iac/cli/internal/typer/generator/core"
	"github.com/rudderlabs/rudder-iac/cli/internal/typer/generator/platforms/kotlin"
	"github.com/rudderlabs/rudder-iac/cli/internal/typer/plan/testutils"
	"github.com/stretchr/testify/assert"
)

//go:embed testdata/validator/src/main/kotlin/com/rudderstack/ruddertyper/Main.kt
var mainContent string

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
