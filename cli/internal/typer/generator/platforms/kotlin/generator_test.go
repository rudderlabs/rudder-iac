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

//go:embed testdata/compose_stable/compose_stable.kt
var composeStableContent string

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

func TestGenerateWithComposeStable(t *testing.T) {
	generator := &kotlin.Generator{}
	files, err := generator.Generate(composeStableTestPlan(), core.GenerateOptions{
		RudderCLIVersion: "1.0.0",
	}, kotlin.KotlinOptions{
		ComposeStable: true,
	})

	assert.NoError(t, err)
	assert.Len(t, files, 1)

	if diff := cmp.Diff(composeStableContent, files[0].Content); diff != "" {
		t.Errorf("Generated content does not match expected (-expected +actual):\n%s", diff)
	}
}

// composeStableTestPlan returns a tracking plan with data classes, nested data classes,
// and a variant event (sealed class) for testing composeStable annotation generation
func composeStableTestPlan() *plan.TrackingPlan {
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
			{
				Event: plan.Event{
					EventType: plan.EventTypeTrack,
					Name:      "Page Viewed",
				},
				Section: plan.IdentitySectionProperties,
				Schema: plan.ObjectSchema{
					Properties: map[string]plan.PropertySchema{
						"device_type": {
							Property: plan.Property{
								Name:  "device_type",
								Types: []plan.PropertyType{plan.PrimitiveTypeString},
							},
							Required: true,
						},
					},
				},
				Variants: []plan.Variant{
					{
						Type:          "discriminator",
						Discriminator: "device_type",
						Cases: []plan.VariantCase{
							{
								DisplayName: "Mobile",
								Match:       []any{"mobile"},
								Schema: plan.ObjectSchema{
									Properties: map[string]plan.PropertySchema{
										"screen_size": {
											Property: plan.Property{
												Name:  "screen_size",
												Types: []plan.PropertyType{plan.PrimitiveTypeString},
											},
										},
									},
								},
							},
							{
								DisplayName: "Desktop",
								Match:       []any{"desktop"},
								Schema: plan.ObjectSchema{
									Properties: map[string]plan.PropertySchema{
										"browser": {
											Property: plan.Property{
												Name:  "browser",
												Types: []plan.PropertyType{plan.PrimitiveTypeString},
											},
										},
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
