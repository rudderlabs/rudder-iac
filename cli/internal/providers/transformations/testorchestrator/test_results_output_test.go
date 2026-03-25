package testorchestrator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	transformations "github.com/rudderlabs/rudder-iac/api/client/transformations"
)

func TestToOutput(t *testing.T) {
	t.Run("flattens result wrapper in transformation output", func(t *testing.T) {
		results := &TestResults{
			Status: RunStatusExecuted,
			Transformations: []*TransformationTestWithDefinitions{
				{
					Result: &transformations.TransformationTestResult{
						ID:        "tr-1",
						Name:      "My Transform",
						VersionID: "v-1",
						Imports:   []string{"libA"},
						Pass:      true,
						TestSuiteResult: transformations.TestSuiteRunResult{
							Status: transformations.TestRunStatusPass,
						},
					},
				},
			},
		}

		output, _ := results.ToOutput()

		require.Len(t, output.Transformations, 1)
		assert.Equal(t, TransformationTestOutput{
			ID:        "tr-1",
			Name:      "My Transform",
			VersionID: "v-1",
			Imports:   []string{"libA"},
			Pass:      true,
			TestSuiteResult: TestSuiteResultOutput{
				Status: transformations.TestRunStatusPass,
			},
		}, output.Transformations[0])
	})

	t.Run("maps input and output location from definitions", func(t *testing.T) {
		results := &TestResults{
			Status: RunStatusExecuted,
			Transformations: []*TransformationTestWithDefinitions{
				{
					Result: &transformations.TransformationTestResult{
						Name: "Transform",
						TestSuiteResult: transformations.TestSuiteRunResult{
							Status: transformations.TestRunStatusFail,
							Results: []transformations.TestResult{
								{
									ID:     "suite/input/event.json",
									Name:   "suite (event.json)",
									Status: transformations.TestRunStatusFail,
								},
							},
						},
					},
					Definitions: []*transformations.TestDefinition{
						{
							ID:         "suite/input/event.json",
							Name:       "suite (event.json)",
							Filename:   "event.json",
							InputFile:  "input/event.json",
							OutputFile: "output/event.json",
						},
					},
				},
			},
		}

		output, _ := results.ToOutput()

		result := output.Transformations[0].TestSuiteResult.Results[0]
		assert.Equal(t, "input/event.json", result.InputLocation)
		assert.Equal(t, "output/event.json", result.OutputLocation)
	})

	t.Run("omits locations when no definition matches", func(t *testing.T) {
		results := &TestResults{
			Status: RunStatusExecuted,
			Transformations: []*TransformationTestWithDefinitions{
				{
					Result: &transformations.TransformationTestResult{
						Name: "Remote Transform",
						TestSuiteResult: transformations.TestSuiteRunResult{
							Status: transformations.TestRunStatusPass,
							Results: []transformations.TestResult{
								{
									ID:     "unknown-test",
									Name:   "unknown-test",
									Status: transformations.TestRunStatusPass,
								},
							},
						},
					},
					// No definitions — standalone library transformation result
				},
			},
		}

		output, entries := results.ToOutput()

		result := output.Transformations[0].TestSuiteResult.Results[0]
		assert.Empty(t, result.InputLocation)
		assert.Empty(t, result.OutputLocation)
		assert.Empty(t, result.ActualOutputFile)
		assert.Empty(t, entries)
	})

	t.Run("replaces actualOutput with file path reference", func(t *testing.T) {
		results := &TestResults{
			Status: RunStatusExecuted,
			Transformations: []*TransformationTestWithDefinitions{
				{
					Result: &transformations.TransformationTestResult{
						Name: "My Transform",
						TestSuiteResult: transformations.TestSuiteRunResult{
							Status: transformations.TestRunStatusPass,
							Results: []transformations.TestResult{
								{
									ID:           "suite/input/event.json",
									Name:         "suite (event.json)",
									Status:       transformations.TestRunStatusPass,
									ActualOutput: []any{map[string]any{"key": "value"}},
								},
							},
						},
					},
					Definitions: []*transformations.TestDefinition{
						{
							ID:        "suite/input/event.json",
							Filename:  "event.json",
							InputFile: "input/event.json",
						},
					},
				},
			},
		}

		output, entries := results.ToOutput()

		result := output.Transformations[0].TestSuiteResult.Results[0]
		assert.Equal(t, "My Transform/suite (event.json)/event.json", result.ActualOutputFile)
		assert.Nil(t, result.Errors)

		require.Len(t, entries, 1)
		assert.Equal(t, "My Transform/suite (event.json)/event.json", entries[0].RelPath)
		assert.Equal(t, []any{map[string]any{"key": "value"}}, entries[0].ActualOutput)
	})

	t.Run("uses default_events.json for default-events tests", func(t *testing.T) {
		results := &TestResults{
			Status: RunStatusExecuted,
			Transformations: []*TransformationTestWithDefinitions{
				{
					Result: &transformations.TransformationTestResult{
						Name: "My Transform",
						TestSuiteResult: transformations.TestSuiteRunResult{
							Status: transformations.TestRunStatusPass,
							Results: []transformations.TestResult{
								{
									ID:           "default-events",
									Name:         "default-events",
									Status:       transformations.TestRunStatusPass,
									ActualOutput: []any{map[string]any{"event": "data"}},
								},
							},
						},
					},
					Definitions: []*transformations.TestDefinition{
						{
							ID:       "default-events",
							Name:     "default-events",
							Filename: "default_events.json",
						},
					},
				},
			},
		}

		output, entries := results.ToOutput()

		result := output.Transformations[0].TestSuiteResult.Results[0]
		assert.Equal(t, "My Transform/default-events/default_events.json", result.ActualOutputFile)
		assert.Empty(t, result.InputLocation)
		assert.Empty(t, result.OutputLocation)

		require.Len(t, entries, 1)
		assert.Equal(t, "My Transform/default-events/default_events.json", entries[0].RelPath)
	})

	t.Run("skips actual output entries when actualOutput is empty", func(t *testing.T) {
		results := &TestResults{
			Status: RunStatusExecuted,
			Transformations: []*TransformationTestWithDefinitions{
				{
					Result: &transformations.TransformationTestResult{
						Name: "Transform",
						TestSuiteResult: transformations.TestSuiteRunResult{
							Status: transformations.TestRunStatusError,
							Results: []transformations.TestResult{
								{
									ID:     "suite/input/event.json",
									Name:   "suite (event.json)",
									Status: transformations.TestRunStatusError,
									Errors: []transformations.TestError{{Message: "some error"}},
								},
							},
						},
					},
					Definitions: []*transformations.TestDefinition{
						{
							ID:        "suite/input/event.json",
							Filename:  "event.json",
							InputFile: "input/event.json",
						},
					},
				},
			},
		}

		output, entries := results.ToOutput()

		result := output.Transformations[0].TestSuiteResult.Results[0]
		assert.Empty(t, result.ActualOutputFile)
		assert.Equal(t, "input/event.json", result.InputLocation)
		assert.Empty(t, entries)
	})

	t.Run("sanitizes transformation and test names in paths", func(t *testing.T) {
		results := &TestResults{
			Status: RunStatusExecuted,
			Transformations: []*TransformationTestWithDefinitions{
				{
					Result: &transformations.TransformationTestResult{
						Name: "My/Transform:v2",
						TestSuiteResult: transformations.TestSuiteRunResult{
							Status: transformations.TestRunStatusPass,
							Results: []transformations.TestResult{
								{
									ID:           "test-1",
									Name:         "suite<1>",
									Status:       transformations.TestRunStatusPass,
									ActualOutput: []any{"data"},
								},
							},
						},
					},
					Definitions: []*transformations.TestDefinition{
						{
							ID:       "test-1",
							Filename: "event.json",
						},
					},
				},
			},
		}

		_, entries := results.ToOutput()

		require.Len(t, entries, 1)
		assert.Equal(t, "My_Transform_v2/suite_1_/event.json", entries[0].RelPath)
	})

	t.Run("preserves library results unchanged", func(t *testing.T) {
		libs := []transformations.LibraryTestResult{
			{HandleName: "lib1", VersionID: "v1", Pass: true},
			{HandleName: "lib2", VersionID: "v2", Pass: false, Message: "syntax error"},
		}

		results := &TestResults{
			Status:    RunStatusExecuted,
			Libraries: libs,
		}

		output, _ := results.ToOutput()

		assert.Equal(t, libs, output.Libraries)
	})

	t.Run("handles nil transformations", func(t *testing.T) {
		results := &TestResults{
			Status: RunStatusExecuted,
		}

		output, entries := results.ToOutput()

		assert.Equal(t, RunStatusExecuted, output.Status)
		assert.Nil(t, output.Transformations)
		assert.Nil(t, entries)
	})
}

func TestSanitizePath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "simple"},
		{"path/with/slashes", "path_with_slashes"},
		{"name:v2", "name_v2"},
		{"file<1>", "file_1_"},
		{"a*b?c", "a_b_c"},
		{"back\\slash", "back_slash"},
		{`with"quotes`, "with_quotes"},
		{"pipe|char", "pipe_char"},
		{"  spaces  ", "spaces"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, sanitizePath(tt.input))
		})
	}
}
