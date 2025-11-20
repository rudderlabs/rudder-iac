package differ_test

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/differ"
	"github.com/stretchr/testify/assert"
)

func TestCompareData(t *testing.T) {
	data1 := resources.ResourceData{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
		// PropertyRef pointer - same URN and Property as data2
		"refPtr1": &resources.PropertyRef{
			URN:      "test:urn1",
			Property: "prop1",
		},
		// PropertyRef pointer - different URN from data2
		"refPtr2": &resources.PropertyRef{
			URN:      "test:urn2",
			Property: "prop2",
		},
		// PropertyRef value - same URN and Property as data2
		"refValue1": resources.PropertyRef{
			URN:      "test:urn3",
			Property: "prop3",
		},
		// PropertyRef value - different URN from data2
		"refValue2": resources.PropertyRef{
			URN:      "test:urn4",
			Property: "prop4",
		},
	}

	data2 := resources.ResourceData{
		"key1": "value1",
		"key2": "value3",
		"key4": "value4",
		// PropertyRef pointer - same URN and Property as data1
		"refPtr1": &resources.PropertyRef{
			URN:      "test:urn1",
			Property: "prop1",
		},
		// PropertyRef pointer - different URN from data1
		"refPtr2": &resources.PropertyRef{
			URN:      "test:different-urn",
			Property: "prop2",
		},
		// PropertyRef value - same URN and Property as data1
		"refValue1": resources.PropertyRef{
			URN:      "test:urn3",
			Property: "prop3",
		},
		// PropertyRef value - different URN from data1
		"refValue2": resources.PropertyRef{
			URN:      "test:different-urn",
			Property: "prop4",
		},
	}

	diffs := differ.CompareData(data1, data2)

	// Should have diffs for: key2, key3, key4, refPtr2, refValue2
	// Should NOT have diffs for: key1, refPtr1, refValue1
	assert.Len(t, diffs, 5)

	assert.Contains(t, diffs, "key2")
	assert.Contains(t, diffs, "key3")
	assert.Contains(t, diffs, "key4")
	assert.Contains(t, diffs, "refPtr2")
	assert.Contains(t, diffs, "refValue2")

	assert.Equal(t, diffs["key2"].SourceValue, "value2")
	assert.Equal(t, diffs["key2"].TargetValue, "value3")

	assert.Equal(t, diffs["key3"].SourceValue, "value3")
	assert.Nil(t, diffs["key3"].TargetValue)

	assert.Nil(t, diffs["key4"].SourceValue)
	assert.Equal(t, diffs["key4"].TargetValue, "value4")

	// Verify PropertyRefs with same URN/Property are not in diffs
	assert.NotContains(t, diffs, "refPtr1")
	assert.NotContains(t, diffs, "refValue1")
}

func TestCompareRawData(t *testing.T) {
	type TestStruct struct {
		Name        string                 `diff:"name"`
		Value       int                    `diff:"value"`
		Ref         *resources.PropertyRef `diff:"ref"`
		Tags        []string               `diff:"tags"`
		NestedRef   *resources.PropertyRef `diff:"nested_ref"`
		Description *string                `diff:"description"`
		Internal    string                 // No diff tag
	}

	desc1 := "description1"
	desc2 := "description2"

	t.Run("identical structs", func(t *testing.T) {
		raw1 := &TestStruct{
			Name:  "test",
			Value: 42,
			Ref: &resources.PropertyRef{
				URN:      "test:urn",
				Property: "prop",
			},
			Tags: []string{"tag1", "tag2"},
			NestedRef: &resources.PropertyRef{
				URN:      "test:nested",
				Property: "nested-prop",
			},
			Description: &desc1,
		}
		raw2 := &TestStruct{
			Name:  "test",
			Value: 42,
			Ref: &resources.PropertyRef{
				URN:      "test:urn",
				Property: "prop",
			},
			Tags: []string{"tag1", "tag2"},
			NestedRef: &resources.PropertyRef{
				URN:      "test:nested",
				Property: "nested-prop",
			},
			Description: &desc1,
		}

		diffs := differ.CompareRawData(raw1, raw2)
		assert.Len(t, diffs, 0)
	})

	t.Run("different field values", func(t *testing.T) {
		raw1 := &TestStruct{
			Name:  "test1",
			Value: 42,
		}
		raw2 := &TestStruct{
			Name:  "test2",
			Value: 99,
		}

		diffs := differ.CompareRawData(raw1, raw2)
		assert.Len(t, diffs, 2)
		// Should use diff tag names
		assert.Contains(t, diffs, "name")
		assert.Contains(t, diffs, "value")
		assert.Equal(t, "test1", diffs["name"].SourceValue)
		assert.Equal(t, "test2", diffs["name"].TargetValue)
	})

	t.Run("different PropertyRefs", func(t *testing.T) {
		raw1 := &TestStruct{
			Name: "test",
			Ref: &resources.PropertyRef{
				URN:      "test:urn1",
				Property: "prop",
			},
		}
		raw2 := &TestStruct{
			Name: "test",
			Ref: &resources.PropertyRef{
				URN:      "test:urn2",
				Property: "prop",
			},
		}

		diffs := differ.CompareRawData(raw1, raw2)
		assert.Len(t, diffs, 1)
		// Should use diff tag name
		assert.Contains(t, diffs, "ref")
	})

	t.Run("different pointer values", func(t *testing.T) {
		raw1 := &TestStruct{
			Name:        "test",
			Description: &desc1,
		}
		raw2 := &TestStruct{
			Name:        "test",
			Description: &desc2,
		}

		diffs := differ.CompareRawData(raw1, raw2)
		assert.Len(t, diffs, 1)
		// Should use diff tag name
		assert.Contains(t, diffs, "description")
	})

	t.Run("different field without diff tag", func(t *testing.T) {
		raw1 := &TestStruct{
			Name:     "test",
			Internal: "internal1",
		}
		raw2 := &TestStruct{
			Name:     "test",
			Internal: "internal2",
		}

		diffs := differ.CompareRawData(raw1, raw2)
		// Should NOT report Internal field since it has no diff tag
		assert.Len(t, diffs, 0)
	})

	t.Run("nested structs with diff tags", func(t *testing.T) {
		type NestedStruct struct {
			Field1 string `diff:"field1"`
			Field2 int    `diff:"field2"`
		}
		type ParentStruct struct {
			Name   string        `diff:"name"`
			Nested *NestedStruct `diff:"nested"`
		}

		raw1 := &ParentStruct{
			Name: "parent",
			Nested: &NestedStruct{
				Field1: "value1",
				Field2: 10,
			},
		}
		raw2 := &ParentStruct{
			Name: "parent",
			Nested: &NestedStruct{
				Field1: "value2",
				Field2: 10,
			},
		}

		diffs := differ.CompareRawData(raw1, raw2)
		assert.Len(t, diffs, 1)
		// Should report nested field with full path
		assert.Contains(t, diffs, "nested.field1")
		assert.Equal(t, "value1", diffs["nested.field1"].SourceValue)
		assert.Equal(t, "value2", diffs["nested.field1"].TargetValue)
	})

	t.Run("nested struct without diff tag on parent", func(t *testing.T) {
		type NestedStruct struct {
			Field1 string `diff:"field1"`
			Field2 int    `diff:"field2"`
		}
		type ParentStruct struct {
			Name   string        `diff:"name"`
			Nested *NestedStruct // No diff tag
		}

		raw1 := &ParentStruct{
			Name: "parent",
			Nested: &NestedStruct{
				Field1: "value1",
				Field2: 10,
			},
		}
		raw2 := &ParentStruct{
			Name: "parent",
			Nested: &NestedStruct{
				Field1: "value2",
				Field2: 10,
			},
		}

		diffs := differ.CompareRawData(raw1, raw2)
		assert.Len(t, diffs, 1)
		// Should report nested field without parent in path since parent has no diff tag
		assert.Contains(t, diffs, "field1")
		assert.Equal(t, "value1", diffs["field1"].SourceValue)
		assert.Equal(t, "value2", diffs["field1"].TargetValue)
	})

	t.Run("deeply nested structs with pointer parent", func(t *testing.T) {
		type DeepNested struct {
			Value string `diff:"value"`
		}
		type MiddleStruct struct {
			Deep *DeepNested `diff:"deep"`
		}
		type ParentStruct struct {
			Middle *MiddleStruct `diff:"middle"`
		}

		raw1 := ParentStruct{
			Middle: &MiddleStruct{
				Deep: &DeepNested{
					Value: "old",
				},
			},
		}
		raw2 := ParentStruct{
			Middle: &MiddleStruct{
				Deep: &DeepNested{
					Value: "new",
				},
			},
		}

		diffs := differ.CompareRawData(raw1, raw2)
		assert.Len(t, diffs, 1)
		// Should report deeply nested field with full path
		assert.Contains(t, diffs, "middle.deep.value")
		assert.Equal(t, "old", diffs["middle.deep.value"].SourceValue)
		assert.Equal(t, "new", diffs["middle.deep.value"].TargetValue)
	})

	t.Run("nil vs non-nil nested struct", func(t *testing.T) {
		type NestedStruct struct {
			Field1 string `diff:"field1"`
		}
		type ParentStruct struct {
			Name   string        `diff:"name"`
			Nested *NestedStruct `diff:"nested"`
		}

		raw1 := &ParentStruct{
			Name:   "test",
			Nested: nil, // nil nested
		}
		raw2 := &ParentStruct{
			Name: "test",
			Nested: &NestedStruct{
				Field1: "value",
			},
		}

		diffs := differ.CompareRawData(raw1, raw2)
		assert.Len(t, diffs, 1)
		// Should report nested fields with full path
		assert.Contains(t, diffs, "nested.field1")
		assert.Nil(t, diffs["nested.field1"].SourceValue)
		assert.Equal(t, "value", diffs["nested.field1"].TargetValue)
	})

	t.Run("nil vs non-nil deeply nested struct", func(t *testing.T) {
		type DeepNested struct {
			Value string `diff:"value"`
		}
		type MiddleStruct struct {
			Deep  *DeepNested `diff:"deep"`
			Field string      `diff:"field"`
		}
		type ParentStruct struct {
			Name   string        `diff:"name"`
			Middle *MiddleStruct `diff:"middle"`
		}

		raw1 := &ParentStruct{
			Name:   "test",
			Middle: nil, // nil middle
		}
		raw2 := &ParentStruct{
			Name: "test",
			Middle: &MiddleStruct{
				Deep: &DeepNested{
					Value: "deep-value",
				},
				Field: "middle-field",
			},
		}

		diffs := differ.CompareRawData(raw1, raw2)
		assert.Len(t, diffs, 2)
		// Should report all nested fields with full paths
		assert.Contains(t, diffs, "middle.field")
		assert.Contains(t, diffs, "middle.deep.value")
		assert.Nil(t, diffs["middle.field"].SourceValue)
		assert.Equal(t, "middle-field", diffs["middle.field"].TargetValue)
		assert.Nil(t, diffs["middle.deep.value"].SourceValue)
		assert.Equal(t, "deep-value", diffs["middle.deep.value"].TargetValue)
	})

	t.Run("embedded struct fields", func(t *testing.T) {
		type EmbeddedConfig struct {
			PropagateViolations     *bool `diff:"propagate_violations"`
			DropUnplannedProperties *bool `diff:"drop_unplanned_properties"`
		}

		type TrackConfig struct {
			*EmbeddedConfig
			DropUnplannedEvents *bool `diff:"drop_unplanned_events"`
		}

		type Config struct {
			Track  *TrackConfig `diff:"track"`
			Screen *EmbeddedConfig `diff:"screen"`
		}

		type Validations struct {
			Config *Config `diff:"config"`
		}

		type Governance struct {
			Validations *Validations `diff:"validations"`
		}

		type Source struct {
			Name       string      `diff:"name"`
			Governance *Governance `diff:"governance"`
		}

		trueVal := true

		raw1 := &Source{
			Name:       "test",
			Governance: nil,
		}
		raw2 := &Source{
			Name: "test",
			Governance: &Governance{
				Validations: &Validations{
					Config: &Config{
						Track: &TrackConfig{
							EmbeddedConfig: &EmbeddedConfig{
								DropUnplannedProperties: &trueVal,
							},
						},
						Screen: nil,
					},
				},
			},
		}

		diffs := differ.CompareRawData(raw1, raw2)

		// Should report the nested field from the embedded struct
		assert.Contains(t, diffs, "governance.validations.config.track.drop_unplanned_properties")
		assert.Nil(t, diffs["governance.validations.config.track.drop_unplanned_properties"].SourceValue)
		assert.Equal(t, &trueVal, diffs["governance.validations.config.track.drop_unplanned_properties"].TargetValue)

		// Should NOT report nil fields like screen (which is nil on both sides)
		assert.NotContains(t, diffs, "governance.validations.config.screen")
	})

	t.Run("embedded struct fields - both sources have governance", func(t *testing.T) {
		type EmbeddedConfig struct {
			PropagateViolations     *bool `diff:"propagate_violations"`
			DropUnplannedProperties *bool `diff:"drop_unplanned_properties"`
		}

		type TrackConfig struct {
			*EmbeddedConfig
			DropUnplannedEvents *bool `diff:"drop_unplanned_events"`
		}

		type Config struct {
			Track  *TrackConfig     `diff:"track"`
			Screen *EmbeddedConfig `diff:"screen"`
		}

		type Validations struct {
			Config *Config `diff:"config"`
		}

		type Governance struct {
			Validations *Validations `diff:"validations"`
		}

		type Source struct {
			Name       string      `diff:"name"`
			Governance *Governance `diff:"governance"`
		}

		trueVal := true

		// Both have governance, but different track configs
		raw1 := &Source{
			Name: "test",
			Governance: &Governance{
				Validations: &Validations{
					Config: &Config{
						Track:  nil,
						Screen: nil,
					},
				},
			},
		}
		raw2 := &Source{
			Name: "test",
			Governance: &Governance{
				Validations: &Validations{
					Config: &Config{
						Track: &TrackConfig{
							EmbeddedConfig: &EmbeddedConfig{
								DropUnplannedProperties: &trueVal,
							},
						},
						Screen: nil,
					},
				},
			},
		}

		diffs := differ.CompareRawData(raw1, raw2)

		// Should report the nested field from the embedded struct
		assert.Contains(t, diffs, "governance.validations.config.track.drop_unplanned_properties")
		assert.Nil(t, diffs["governance.validations.config.track.drop_unplanned_properties"].SourceValue)
		assert.Equal(t, &trueVal, diffs["governance.validations.config.track.drop_unplanned_properties"].TargetValue)

		// Should NOT report nil fields like screen (which is nil on both sides)
		assert.NotContains(t, diffs, "governance.validations.config.screen")
	})

	t.Run("embedded struct with ALL nil fields", func(t *testing.T) {
		type EmbeddedConfig struct {
			PropagateViolations     *bool `diff:"propagate_violations"`
			DropUnplannedProperties *bool `diff:"drop_unplanned_properties"`
		}

		type TrackConfig struct {
			*EmbeddedConfig
			DropUnplannedEvents *bool `diff:"drop_unplanned_events"`
		}

		type Config struct {
			Track *TrackConfig `diff:"track"`
		}

		// Both fields in EmbeddedConfig are nil, and DropUnplannedEvents is also nil
		raw1 := &Config{
			Track: nil,
		}
		raw2 := &Config{
			Track: &TrackConfig{
				EmbeddedConfig: &EmbeddedConfig{
					PropagateViolations:     nil,
					DropUnplannedProperties: nil,
				},
				DropUnplannedEvents: nil,
			},
		}

		diffs := differ.CompareRawData(raw1, raw2)

		// When ALL fields are nil, we should not report anything
		// (there's no actual data change, just structure)
		assert.Len(t, diffs, 0, "Should not report diffs when all fields are nil")
	})
}

func TestComputeDiff(t *testing.T) {
	g1 := resources.NewGraph()
	g2 := resources.NewGraph()

	g1.AddResource(resources.NewResource("r0", "some-type", resources.ResourceData{"key1": "value1", "key2": "value2"}, []string{}))
	g1.AddResource(resources.NewResource("r1", "some-type", resources.ResourceData{"key1": "value1", "key2": "value2"}, []string{}))
	g1.AddResource(resources.NewResource("r2", "some-type", resources.ResourceData{"key1": "value1", "key2": "value2"}, []string{}))

	g2.AddResource(resources.NewResource("r0", "some-type", resources.ResourceData{"key1": "value1", "key2": "value2"}, []string{}))
	g2.AddResource(resources.NewResource("r1", "some-type", resources.ResourceData{"key1": "value1", "key2": "value3"}, []string{}))
	g2.AddResource(resources.NewResource("r3", "some-type", resources.ResourceData{"key1": "value1", "key2": "value3"}, []string{}))
	g2.AddResource(resources.NewResource("r4", "some-type", resources.ResourceData{"key1": "value1", "key2": "value4"}, []string{}, resources.WithResourceImportMetadata("remote-id-r4", "workspace-id")))

	diff := differ.ComputeDiff(g1, g2, differ.DiffOptions{WorkspaceID: "workspace-id"})

	assert.Len(t, diff.NewResources, 1)
	assert.Len(t, diff.ImportableResources, 1)
	assert.Len(t, diff.UpdatedResources, 1)
	assert.Len(t, diff.RemovedResources, 1)
	assert.Len(t, diff.UnmodifiedResources, 1)

	assert.Contains(t, diff.NewResources, "some-type:r3")
	assert.Contains(t, diff.ImportableResources, "some-type:r4")
	assert.Equal(t, diff.UpdatedResources["some-type:r1"], differ.ResourceDiff{URN: "some-type:r1", Diffs: map[string]differ.PropertyDiff{"key2": {Property: "key2", SourceValue: "value2", TargetValue: "value3"}}})
	assert.Contains(t, diff.RemovedResources, "some-type:r2")
	assert.Contains(t, diff.UnmodifiedResources, "some-type:r0")
}

func TestComputeDiffWithRawData(t *testing.T) {
	type TestResource struct {
		Name  string                 `diff:"name"`
		Value int                    `diff:"value"`
		Ref   *resources.PropertyRef `diff:"ref"`
	}

	g1 := resources.NewGraph()
	g2 := resources.NewGraph()

	// Add resources with RawData
	g1.AddResource(resources.NewResource("r0", "raw-type", nil, []string{},
		resources.WithRawData(&TestResource{
			Name:  "test",
			Value: 42,
			Ref: &resources.PropertyRef{
				URN:      "test:urn",
				Property: "prop",
			},
		})))

	g1.AddResource(resources.NewResource("r1", "raw-type", nil, []string{},
		resources.WithRawData(&TestResource{
			Name:  "test1",
			Value: 100,
		})))

	// r0 has same data, r1 has different data
	g2.AddResource(resources.NewResource("r0", "raw-type", nil, []string{},
		resources.WithRawData(&TestResource{
			Name:  "test",
			Value: 42,
			Ref: &resources.PropertyRef{
				URN:      "test:urn",
				Property: "prop",
			},
		})))

	g2.AddResource(resources.NewResource("r1", "raw-type", nil, []string{},
		resources.WithRawData(&TestResource{
			Name:  "test1",
			Value: 999, // Different value
		})))

	// New resource with RawData
	g2.AddResource(resources.NewResource("r2", "raw-type", nil, []string{},
		resources.WithRawData(&TestResource{
			Name:  "new",
			Value: 1,
		})))

	diff := differ.ComputeDiff(g1, g2, differ.DiffOptions{})

	assert.Len(t, diff.NewResources, 1)
	assert.Len(t, diff.UpdatedResources, 1)
	assert.Len(t, diff.UnmodifiedResources, 1)

	assert.Contains(t, diff.NewResources, "raw-type:r2")
	assert.Contains(t, diff.UnmodifiedResources, "raw-type:r0")
	assert.Contains(t, diff.UpdatedResources, "raw-type:r1")

	// Check that the diff contains the value field (using diff tag name)
	r1Diff := diff.UpdatedResources["raw-type:r1"]
	assert.Contains(t, r1Diff.Diffs, "value")
	assert.Equal(t, 100, r1Diff.Diffs["value"].SourceValue)
	assert.Equal(t, 999, r1Diff.Diffs["value"].TargetValue)
}
