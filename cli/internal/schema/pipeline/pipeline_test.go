package pipeline

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rudderlabs/rudder-iac/cli/pkg/logger"
	schemaErrors "github.com/rudderlabs/rudder-iac/cli/pkg/schema/errors"
)

// mockStage implements the Stage interface for testing
type mockStage struct {
	name         string
	processFunc  func(ctx context.Context, input StageInput) (StageOutput, error)
	validateFunc func(input StageInput) error
	rollbackFunc func(ctx context.Context, output StageOutput) error
	canRetry     bool
	maxRetries   int
}

func (m *mockStage) Name() string {
	return m.name
}

func (m *mockStage) Process(ctx context.Context, input StageInput) (StageOutput, error) {
	if m.processFunc != nil {
		return m.processFunc(ctx, input)
	}
	return StageOutput{
		Data:     input.Data,
		Metadata: input.Metadata,
		Context:  input.Context,
		Metrics:  StageMetrics{ItemsIn: 1, ItemsOut: 1},
	}, nil
}

func (m *mockStage) Validate(input StageInput) error {
	if m.validateFunc != nil {
		return m.validateFunc(input)
	}
	return nil
}

func (m *mockStage) Rollback(ctx context.Context, output StageOutput) error {
	if m.rollbackFunc != nil {
		return m.rollbackFunc(ctx, output)
	}
	return nil
}

func (m *mockStage) CanRetry() bool {
	return m.canRetry
}

func (m *mockStage) MaxRetries() int {
	return m.maxRetries
}

// mockMetricsCollector implements MetricsCollector for testing
type mockMetricsCollector struct {
	stageStarts    []string
	stageEnds      []string
	pipelineStarts []string
	pipelineEnds   []string
	errors         []error
}

func (m *mockMetricsCollector) RecordStageStart(stageName string) {
	m.stageStarts = append(m.stageStarts, stageName)
}

func (m *mockMetricsCollector) RecordStageEnd(stageName string, metrics StageMetrics) {
	m.stageEnds = append(m.stageEnds, stageName)
}

func (m *mockMetricsCollector) RecordPipelineStart(pipelineName string) {
	m.pipelineStarts = append(m.pipelineStarts, pipelineName)
}

func (m *mockMetricsCollector) RecordPipelineEnd(pipelineName string, totalDuration time.Duration, success bool) {
	m.pipelineEnds = append(m.pipelineEnds, pipelineName)
}

func (m *mockMetricsCollector) RecordError(stageName string, err error) {
	m.errors = append(m.errors, err)
}

func createTestLogger() *logger.Logger {
	log := logger.New("test")
	return log
}

func TestNewPipeline(t *testing.T) {
	log := createTestLogger()

	tests := []struct {
		name    string
		options []PipelineOption
		verify  func(t *testing.T, p *Pipeline)
	}{
		{
			name:    "default_pipeline",
			options: nil,
			verify: func(t *testing.T, p *Pipeline) {
				if p.maxRetries != 3 {
					t.Errorf("Expected maxRetries=3, got %d", p.maxRetries)
				}
				if p.rollbackOnError {
					t.Error("Expected rollbackOnError=false")
				}
				if p.continueOnError {
					t.Error("Expected continueOnError=false")
				}
			},
		},
		{
			name: "with_rollback",
			options: []PipelineOption{
				WithRollbackOnError(),
			},
			verify: func(t *testing.T, p *Pipeline) {
				if !p.rollbackOnError {
					t.Error("Expected rollbackOnError=true")
				}
			},
		},
		{
			name: "with_max_retries",
			options: []PipelineOption{
				WithMaxRetries(5),
			},
			verify: func(t *testing.T, p *Pipeline) {
				if p.maxRetries != 5 {
					t.Errorf("Expected maxRetries=5, got %d", p.maxRetries)
				}
			},
		},
		{
			name: "with_continue_on_error",
			options: []PipelineOption{
				WithContinueOnError(),
			},
			verify: func(t *testing.T, p *Pipeline) {
				if !p.continueOnError {
					t.Error("Expected continueOnError=true")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pipeline := NewPipeline("test", log, tt.options...)
			tt.verify(t, pipeline)
		})
	}
}

func TestPipeline_AddStage(t *testing.T) {
	log := createTestLogger()
	pipeline := NewPipeline("test", log)

	stage1 := &mockStage{name: "stage1"}
	stage2 := &mockStage{name: "stage2"}

	pipeline.AddStage(stage1).AddStage(stage2)

	stages := pipeline.GetStages()
	if len(stages) != 2 {
		t.Fatalf("Expected 2 stages, got %d", len(stages))
	}

	if stages[0].Name() != "stage1" {
		t.Errorf("Expected first stage name 'stage1', got '%s'", stages[0].Name())
	}

	if stages[1].Name() != "stage2" {
		t.Errorf("Expected second stage name 'stage2', got '%s'", stages[1].Name())
	}
}

func TestPipeline_Execute_Success(t *testing.T) {
	log := createTestLogger()
	pipeline := NewPipeline("test", log)

	metrics := &mockMetricsCollector{}
	pipeline.SetMetricsCollector(metrics)

	// Add stages that transform data
	stage1 := &mockStage{
		name: "stage1",
		processFunc: func(ctx context.Context, input StageInput) (StageOutput, error) {
			data := input.Data.(string) + "_processed_by_stage1"
			return StageOutput{
				Data:     data,
				Metadata: map[string]string{"stage1": "completed"},
				Context:  input.Context,
				Metrics:  StageMetrics{ItemsIn: 1, ItemsOut: 1},
			}, nil
		},
	}

	stage2 := &mockStage{
		name: "stage2",
		processFunc: func(ctx context.Context, input StageInput) (StageOutput, error) {
			data := input.Data.(string) + "_processed_by_stage2"
			return StageOutput{
				Data:     data,
				Metadata: map[string]string{"stage2": "completed"},
				Context:  input.Context,
				Metrics:  StageMetrics{ItemsIn: 1, ItemsOut: 1},
			}, nil
		},
	}

	pipeline.AddStage(stage1).AddStage(stage2)

	input := PipelineInput{
		Data:     "initial_data",
		Metadata: map[string]string{"initial": "true"},
		Options:  map[string]interface{}{"test": true},
	}

	output, err := pipeline.Execute(context.Background(), input)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if output.Data != "initial_data_processed_by_stage1_processed_by_stage2" {
		t.Errorf("Unexpected output data: %v", output.Data)
	}

	if !output.Metrics.Success {
		t.Error("Expected pipeline success")
	}

	if output.Metrics.StagesExecuted != 2 {
		t.Errorf("Expected 2 stages executed, got %d", output.Metrics.StagesExecuted)
	}

	// Verify metrics collection
	if len(metrics.stageStarts) != 2 {
		t.Errorf("Expected 2 stage starts, got %d", len(metrics.stageStarts))
	}

	if len(metrics.stageEnds) != 2 {
		t.Errorf("Expected 2 stage ends, got %d", len(metrics.stageEnds))
	}
}

func TestPipeline_Execute_StageError(t *testing.T) {
	log := createTestLogger()
	pipeline := NewPipeline("test", log)

	metrics := &mockMetricsCollector{}
	pipeline.SetMetricsCollector(metrics)

	// First stage succeeds, second stage fails
	stage1 := &mockStage{
		name: "stage1",
		processFunc: func(ctx context.Context, input StageInput) (StageOutput, error) {
			return StageOutput{
				Data:     "stage1_success",
				Metadata: input.Metadata,
				Context:  input.Context,
				Metrics:  StageMetrics{ItemsIn: 1, ItemsOut: 1},
			}, nil
		},
	}

	stage2 := &mockStage{
		name: "stage2",
		processFunc: func(ctx context.Context, input StageInput) (StageOutput, error) {
			return StageOutput{}, errors.New("stage2 failed")
		},
	}

	pipeline.AddStage(stage1).AddStage(stage2)

	input := PipelineInput{
		Data: "initial_data",
	}

	output, err := pipeline.Execute(context.Background(), input)

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if output.Metrics.Success {
		t.Error("Expected pipeline failure")
	}

	if output.Metrics.StagesExecuted != 1 {
		t.Errorf("Expected 1 stage executed, got %d", output.Metrics.StagesExecuted)
	}

	if len(output.Errors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(output.Errors))
	}

	// Verify error was recorded
	if len(metrics.errors) != 1 {
		t.Errorf("Expected 1 error recorded, got %d", len(metrics.errors))
	}
}

func TestPipeline_Execute_ContinueOnError(t *testing.T) {
	log := createTestLogger()
	pipeline := NewPipeline("test", log, WithContinueOnError())

	// First stage fails, second stage succeeds
	stage1 := &mockStage{
		name: "stage1",
		processFunc: func(ctx context.Context, input StageInput) (StageOutput, error) {
			return StageOutput{}, errors.New("stage1 failed")
		},
	}

	stage2 := &mockStage{
		name: "stage2",
		processFunc: func(ctx context.Context, input StageInput) (StageOutput, error) {
			return StageOutput{
				Data:     "stage2_success",
				Metadata: input.Metadata,
				Context:  input.Context,
				Metrics:  StageMetrics{ItemsIn: 1, ItemsOut: 1},
			}, nil
		},
	}

	pipeline.AddStage(stage1).AddStage(stage2)

	input := PipelineInput{
		Data: "initial_data",
	}

	output, err := pipeline.Execute(context.Background(), input)

	// Should not return error due to continue on error
	if err != nil {
		t.Fatalf("Expected no error with continue on error, got %v", err)
	}

	if !output.Metrics.Success {
		t.Error("Expected pipeline success with continue on error")
	}

	if output.Metrics.StagesExecuted != 1 {
		t.Errorf("Expected 1 stage executed, got %d", output.Metrics.StagesExecuted)
	}

	if len(output.Errors) != 1 {
		t.Errorf("Expected 1 error recorded, got %d", len(output.Errors))
	}

	if output.Data != "stage2_success" {
		t.Errorf("Expected data from successful stage, got %v", output.Data)
	}
}

func TestPipeline_Execute_ValidationError(t *testing.T) {
	log := createTestLogger()
	pipeline := NewPipeline("test", log)

	stage := &mockStage{
		name: "stage1",
		validateFunc: func(input StageInput) error {
			return errors.New("validation failed")
		},
	}

	pipeline.AddStage(stage)

	input := PipelineInput{
		Data: "initial_data",
	}

	output, err := pipeline.Execute(context.Background(), input)

	if err == nil {
		t.Fatal("Expected validation error, got nil")
	}

	if output.Metrics.Success {
		t.Error("Expected pipeline failure")
	}

	if output.Metrics.StagesExecuted != 0 {
		t.Errorf("Expected 0 stages executed, got %d", output.Metrics.StagesExecuted)
	}
}

func TestPipeline_Execute_WithRetries(t *testing.T) {
	log := createTestLogger()
	pipeline := NewPipeline("test", log, WithMaxRetries(2), WithRetryDelay(time.Millisecond))

	attemptCount := 0
	stage := &mockStage{
		name:       "stage1",
		canRetry:   true,
		maxRetries: 2,
		processFunc: func(ctx context.Context, input StageInput) (StageOutput, error) {
			attemptCount++
			if attemptCount < 3 {
				return StageOutput{}, schemaErrors.NewProcessError(
					schemaErrors.ErrorTypeInternal,
					"temporary failure",
					schemaErrors.AsRetryable(),
				)
			}
			return StageOutput{
				Data:     "success_after_retries",
				Metadata: input.Metadata,
				Context:  input.Context,
				Metrics:  StageMetrics{ItemsIn: 1, ItemsOut: 1},
			}, nil
		},
	}

	pipeline.AddStage(stage)

	input := PipelineInput{
		Data: "initial_data",
	}

	output, err := pipeline.Execute(context.Background(), input)

	if err != nil {
		t.Fatalf("Expected success after retries, got %v", err)
	}

	if output.Data != "success_after_retries" {
		t.Errorf("Unexpected output data: %v", output.Data)
	}

	if attemptCount != 3 {
		t.Errorf("Expected 3 attempts (1 initial + 2 retries), got %d", attemptCount)
	}
}

func TestPipeline_Validate(t *testing.T) {
	log := createTestLogger()

	tests := []struct {
		name           string
		setupPipeline  func() *Pipeline
		expectError    bool
		errorSubstring string
	}{
		{
			name: "empty_pipeline",
			setupPipeline: func() *Pipeline {
				return NewPipeline("test", log)
			},
			expectError:    true,
			errorSubstring: "at least one stage",
		},
		{
			name: "duplicate_stage_names",
			setupPipeline: func() *Pipeline {
				pipeline := NewPipeline("test", log)
				stage1 := &mockStage{name: "duplicate"}
				stage2 := &mockStage{name: "duplicate"}
				return pipeline.AddStage(stage1).AddStage(stage2)
			},
			expectError:    true,
			errorSubstring: "Duplicate stage name",
		},
		{
			name: "empty_stage_name",
			setupPipeline: func() *Pipeline {
				pipeline := NewPipeline("test", log)
				stage := &mockStage{name: ""}
				return pipeline.AddStage(stage)
			},
			expectError:    true,
			errorSubstring: "non-empty names",
		},
		{
			name: "valid_pipeline",
			setupPipeline: func() *Pipeline {
				pipeline := NewPipeline("test", log)
				stage1 := &mockStage{name: "stage1"}
				stage2 := &mockStage{name: "stage2"}
				return pipeline.AddStage(stage1).AddStage(stage2)
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pipeline := tt.setupPipeline()
			err := pipeline.Validate()

			if tt.expectError {
				if err == nil {
					t.Fatal("Expected error, got nil")
				}
				if tt.errorSubstring != "" && !containsSubstring(err.Error(), tt.errorSubstring) {
					t.Errorf("Expected error to contain '%s', got '%s'", tt.errorSubstring, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("Expected no error, got %v", err)
				}
			}
		})
	}
}

func TestPipeline_Execute_ContextCancellation(t *testing.T) {
	log := createTestLogger()
	pipeline := NewPipeline("test", log)

	stage := &mockStage{
		name: "slow_stage",
		processFunc: func(ctx context.Context, input StageInput) (StageOutput, error) {
			// Simulate slow processing
			select {
			case <-ctx.Done():
				return StageOutput{}, ctx.Err()
			case <-time.After(100 * time.Millisecond):
				return StageOutput{
					Data:     "completed",
					Metadata: input.Metadata,
					Context:  input.Context,
					Metrics:  StageMetrics{ItemsIn: 1, ItemsOut: 1},
				}, nil
			}
		},
	}

	pipeline.AddStage(stage)

	input := PipelineInput{
		Data: "initial_data",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	output, err := pipeline.Execute(ctx, input)

	if err == nil {
		t.Fatal("Expected context cancellation error, got nil")
	}

	if output.Metrics.Success {
		t.Error("Expected pipeline failure due to cancellation")
	}
}

// Helper function to check if a string contains a substring
func containsSubstring(s, substring string) bool {
	return len(substring) == 0 || (len(s) >= len(substring) && findSubstring(s, substring))
}

func findSubstring(s, substring string) bool {
	for i := 0; i <= len(s)-len(substring); i++ {
		if s[i:i+len(substring)] == substring {
			return true
		}
	}
	return false
}
