package kotlin_test

import (
	_ "embed"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/rudderlabs/rudder-iac/cli/internal/typer/generator/core"
	"github.com/rudderlabs/rudder-iac/cli/internal/typer/generator/platforms/kotlin"
	"github.com/rudderlabs/rudder-iac/cli/internal/typer/plan"
	"github.com/rudderlabs/rudder-iac/cli/internal/typer/plan/testutils"
	"github.com/stretchr/testify/assert"
)

//go:embed testdata/validator/src/main/kotlin/com/rudderstack/ruddertyper/Main.kt
var mainContent string

//go:embed testdata/annotations/single_annotation.kt
var singleAnnotationContent string

//go:embed testdata/annotations/multiple_annotations.kt
var multipleAnnotationsContent string

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

func TestGenerateWithAnnotations(t *testing.T) {
	generator := &kotlin.Generator{}
	files, err := generator.Generate(annotationTestPlan(), core.GenerateOptions{
		RudderCLIVersion: "1.0.0",
	}, kotlin.KotlinOptions{
		Annotations: "androidx.compose.runtime.Stable",
	})

	assert.NoError(t, err)
	assert.Len(t, files, 1)

	if diff := cmp.Diff(singleAnnotationContent, files[0].Content); diff != "" {
		t.Errorf("Generated content does not match expected (-expected +actual):\n%s", diff)
	}
}

func TestGenerateWithMultipleAnnotations(t *testing.T) {
	generator := &kotlin.Generator{}
	files, err := generator.Generate(annotationTestPlan(), core.GenerateOptions{
		RudderCLIVersion: "1.0.0",
	}, kotlin.KotlinOptions{
		Annotations: "androidx.compose.runtime.Stable,androidx.compose.runtime.Immutable",
	})

	assert.NoError(t, err)
	assert.Len(t, files, 1)

	if diff := cmp.Diff(multipleAnnotationsContent, files[0].Content); diff != "" {
		t.Errorf("Generated content does not match expected (-expected +actual):\n%s", diff)
	}
}

func TestGenerateWithInvalidAnnotation(t *testing.T) {
	generator := &kotlin.Generator{}
	_, err := generator.Generate(annotationTestPlan(), core.GenerateOptions{
		RudderCLIVersion: "1.0.0",
	}, kotlin.KotlinOptions{
		Annotations: "Stable", // Invalid: not fully qualified
	})

	assert.ErrorContains(t, err, "fully qualified class name")
}

// annotationTestPlan returns a minimal tracking plan with two events (one with nested properties)
// for testing annotation generation
func annotationTestPlan() *plan.TrackingPlan {
	return &plan.TrackingPlan{
		Name: "Test Plan",
		Rules: []plan.EventRule{
			{
				Event: plan.Event{
					EventType: plan.EventTypeTrack,
					Name:      "Button Clicked",
				},
				Section: plan.IdentitySectionProperties,
				Schema: plan.ObjectSchema{
					Properties: map[string]plan.PropertySchema{
						"label": {
							Property: plan.Property{
								Name:  "label",
								Types: []plan.PropertyType{plan.PrimitiveTypeString},
							},
							Required: true,
						},
					},
				},
			},
			{
				Event: plan.Event{
					EventType: plan.EventTypeTrack,
					Name:      "Item Viewed",
				},
				Section: plan.IdentitySectionProperties,
				Schema: plan.ObjectSchema{
					Properties: map[string]plan.PropertySchema{
						"item_id": {
							Property: plan.Property{
								Name:  "item_id",
								Types: []plan.PropertyType{plan.PrimitiveTypeString},
							},
							Required: true,
						},
						"details": {
							Property: plan.Property{
								Name:  "details",
								Types: []plan.PropertyType{plan.PrimitiveTypeObject},
							},
							Required: false,
							Schema: &plan.ObjectSchema{
								Properties: map[string]plan.PropertySchema{
									"color": {
										Property: plan.Property{
											Name:  "color",
											Types: []plan.PropertyType{plan.PrimitiveTypeString},
										},
										Required: true,
									},
								},
							},
						},
					},
				},
			},
		},
		Metadata: plan.PlanMetadata{
			TrackingPlanID:      "tp_123",
			TrackingPlanVersion: 1,
		},
	}
}
