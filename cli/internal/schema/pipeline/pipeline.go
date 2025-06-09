package pipeline

import (
	"context"
	"fmt"
	"time"

	"github.com/rudderlabs/rudder-iac/cli/pkg/logger"
	schemaErrors "github.com/rudderlabs/rudder-iac/cli/pkg/schema/errors"
)

// StageInput represents input data for a pipeline stage
type StageInput struct {
	Data     interface{}            `json:"data"`
	Metadata map[string]string      `json:"metadata"`
	Context  map[string]interface{} `json:"context"`
}

// StageOutput represents output data from a pipeline stage
type StageOutput struct {
	Data     interface{}            `json:"data"`
	Metadata map[string]string      `json:"metadata"`
	Context  map[string]interface{} `json:"context"`
	Metrics  StageMetrics           `json:"metrics"`
}

// StageMetrics holds performance metrics for a stage
type StageMetrics struct {
	Duration   time.Duration `json:"duration"`
	MemoryUsed int64         `json:"memory_used"`
	ItemsIn    int           `json:"items_in"`
	ItemsOut   int           `json:"items_out"`
	Errors     int           `json:"errors"`
	Warnings   int           `json:"warnings"`
}

// Stage represents a single processing stage in the pipeline
type Stage interface {
	Name() string
	Process(ctx context.Context, input StageInput) (StageOutput, error)
	Validate(input StageInput) error
	Rollback(ctx context.Context, output StageOutput) error
	CanRetry() bool
	MaxRetries() int
}

// MetricsCollector collects pipeline metrics
type MetricsCollector interface {
	RecordStageStart(stageName string)
	RecordStageEnd(stageName string, metrics StageMetrics)
	RecordPipelineStart(pipelineName string)
	RecordPipelineEnd(pipelineName string, totalDuration time.Duration, success bool)
	RecordError(stageName string, err error)
}

// Pipeline orchestrates multiple stages
type Pipeline struct {
	name            string
	stages          []Stage
	logger          *logger.Logger
	metrics         MetricsCollector
	rollbackOnError bool
	maxRetries      int
	retryDelay      time.Duration
	continueOnError bool
}

// PipelineInput represents input to the entire pipeline
type PipelineInput struct {
	Data     interface{}            `json:"data"`
	Metadata map[string]string      `json:"metadata"`
	Options  map[string]interface{} `json:"options"`
}

// PipelineOutput represents output from the entire pipeline
type PipelineOutput struct {
	Data         interface{}       `json:"data"`
	Metadata     map[string]string `json:"metadata"`
	StageResults []StageOutput     `json:"stage_results"`
	Metrics      PipelineMetrics   `json:"metrics"`
	Errors       []error           `json:"errors,omitempty"`
}

// PipelineMetrics holds metrics for the entire pipeline
type PipelineMetrics struct {
	TotalDuration  time.Duration           `json:"total_duration"`
	StageMetrics   map[string]StageMetrics `json:"stage_metrics"`
	TotalErrors    int                     `json:"total_errors"`
	TotalWarnings  int                     `json:"total_warnings"`
	Success        bool                    `json:"success"`
	StagesExecuted int                     `json:"stages_executed"`
}

// PipelineOption configures pipeline behavior
type PipelineOption func(*Pipeline)

// WithRollbackOnError enables rollback when an error occurs
func WithRollbackOnError() PipelineOption {
	return func(p *Pipeline) {
		p.rollbackOnError = true
	}
}

// WithMaxRetries sets the maximum number of retries for failed stages
func WithMaxRetries(retries int) PipelineOption {
	return func(p *Pipeline) {
		p.maxRetries = retries
	}
}

// WithRetryDelay sets the delay between retries
func WithRetryDelay(delay time.Duration) PipelineOption {
	return func(p *Pipeline) {
		p.retryDelay = delay
	}
}

// WithContinueOnError allows the pipeline to continue even if a stage fails
func WithContinueOnError() PipelineOption {
	return func(p *Pipeline) {
		p.continueOnError = true
	}
}

// defaultMetricsCollector provides basic metrics collection
type defaultMetricsCollector struct {
	logger *logger.Logger
}

func (c *defaultMetricsCollector) RecordStageStart(stageName string) {
	c.logger.Info(fmt.Sprintf("Stage '%s' started", stageName))
}

func (c *defaultMetricsCollector) RecordStageEnd(stageName string, metrics StageMetrics) {
	c.logger.Info(fmt.Sprintf("Stage '%s' completed in %v", stageName, metrics.Duration))
}

func (c *defaultMetricsCollector) RecordPipelineStart(pipelineName string) {
	c.logger.Info(fmt.Sprintf("Pipeline '%s' started", pipelineName))
}

func (c *defaultMetricsCollector) RecordPipelineEnd(pipelineName string, totalDuration time.Duration, success bool) {
	status := "successful"
	if !success {
		status = "failed"
	}
	c.logger.Info(fmt.Sprintf("Pipeline '%s' %s in %v", pipelineName, status, totalDuration))
}

func (c *defaultMetricsCollector) RecordError(stageName string, err error) {
	c.logger.Error(fmt.Sprintf("Error in stage '%s': %v", stageName, err))
}

// NewPipeline creates a new pipeline
func NewPipeline(name string, log *logger.Logger, options ...PipelineOption) *Pipeline {
	p := &Pipeline{
		name:            name,
		stages:          make([]Stage, 0),
		logger:          log,
		metrics:         &defaultMetricsCollector{logger: log},
		rollbackOnError: false,
		maxRetries:      3,
		retryDelay:      time.Second * 2,
		continueOnError: false,
	}

	for _, opt := range options {
		opt(p)
	}

	return p
}

// AddStage adds a stage to the pipeline
func (p *Pipeline) AddStage(stage Stage) *Pipeline {
	p.stages = append(p.stages, stage)
	return p
}

// SetMetricsCollector sets a custom metrics collector
func (p *Pipeline) SetMetricsCollector(collector MetricsCollector) *Pipeline {
	p.metrics = collector
	return p
}

// Execute runs the pipeline with the given input
func (p *Pipeline) Execute(ctx context.Context, input PipelineInput) (*PipelineOutput, error) {
	startTime := time.Now()

	p.metrics.RecordPipelineStart(p.name)

	output := &PipelineOutput{
		Data:         input.Data,
		Metadata:     input.Metadata,
		StageResults: make([]StageOutput, 0, len(p.stages)),
		Metrics: PipelineMetrics{
			StageMetrics:   make(map[string]StageMetrics),
			Success:        true,
			StagesExecuted: 0,
		},
		Errors: make([]error, 0),
	}

	// Convert pipeline input to stage input
	currentInput := StageInput{
		Data:     input.Data,
		Metadata: input.Metadata,
		Context:  input.Options,
	}

	var executedStages []Stage

	for _, stage := range p.stages {
		stageStartTime := time.Now()
		p.metrics.RecordStageStart(stage.Name())

		// Validate input for this stage
		if err := stage.Validate(currentInput); err != nil {
			stageErr := schemaErrors.NewProcessError(
				schemaErrors.ErrorTypeProcessValidation,
				fmt.Sprintf("Stage '%s' validation failed", stage.Name()),
				schemaErrors.WithCause(err),
				schemaErrors.WithOperation("pipeline_execute"),
				schemaErrors.WithComponent(stage.Name()),
			)

			output.Errors = append(output.Errors, stageErr)
			output.Metrics.TotalErrors++

			if !p.continueOnError {
				output.Metrics.Success = false
				p.handleError(ctx, executedStages, stageErr)
				return output, stageErr
			}
			continue
		}

		// Execute stage with retries
		stageOutput, err := p.executeStageWithRetries(ctx, stage, currentInput)
		stageDuration := time.Since(stageStartTime)

		// Update stage metrics
		if err == nil {
			stageOutput.Metrics.Duration = stageDuration
		} else {
			stageOutput = StageOutput{
				Metrics: StageMetrics{
					Duration: stageDuration,
					Errors:   1,
				},
			}
		}

		p.metrics.RecordStageEnd(stage.Name(), stageOutput.Metrics)
		output.Metrics.StageMetrics[stage.Name()] = stageOutput.Metrics
		output.StageResults = append(output.StageResults, stageOutput)

		if err != nil {
			output.Errors = append(output.Errors, err)
			output.Metrics.TotalErrors++
			p.metrics.RecordError(stage.Name(), err)

			if !p.continueOnError {
				output.Metrics.Success = false
				p.handleError(ctx, executedStages, err)
				return output, err
			}
		} else {
			// Stage executed successfully
			executedStages = append(executedStages, stage)
			output.Metrics.StagesExecuted++

			// Prepare input for next stage
			currentInput = StageInput{
				Data:     stageOutput.Data,
				Metadata: stageOutput.Metadata,
				Context:  stageOutput.Context,
			}

			// Update pipeline output
			output.Data = stageOutput.Data
			if stageOutput.Metadata != nil {
				if output.Metadata == nil {
					output.Metadata = make(map[string]string)
				}
				for k, v := range stageOutput.Metadata {
					output.Metadata[k] = v
				}
			}
		}

		// Check for context cancellation
		select {
		case <-ctx.Done():
			cancelErr := schemaErrors.NewProcessError(
				schemaErrors.ErrorTypeInternal,
				"Pipeline execution cancelled",
				schemaErrors.WithCause(ctx.Err()),
			)
			output.Errors = append(output.Errors, cancelErr)
			output.Metrics.Success = false
			return output, cancelErr
		default:
			// Continue
		}
	}

	// Calculate final metrics
	output.Metrics.TotalDuration = time.Since(startTime)
	output.Metrics.Success = len(output.Errors) == 0 || p.continueOnError

	p.metrics.RecordPipelineEnd(p.name, output.Metrics.TotalDuration, output.Metrics.Success)

	if len(output.Errors) > 0 && !p.continueOnError {
		return output, output.Errors[0]
	}

	return output, nil
}

// executeStageWithRetries executes a stage with retry logic
func (p *Pipeline) executeStageWithRetries(ctx context.Context, stage Stage, input StageInput) (StageOutput, error) {
	var lastErr error

	maxRetries := p.maxRetries
	if stage.CanRetry() && stage.MaxRetries() > 0 {
		maxRetries = stage.MaxRetries()
	}

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			p.logger.Info(fmt.Sprintf("Retrying stage '%s' (attempt %d/%d)", stage.Name(), attempt, maxRetries))

			// Wait before retrying
			select {
			case <-ctx.Done():
				return StageOutput{}, ctx.Err()
			case <-time.After(p.retryDelay):
				// Continue with retry
			}
		}

		output, err := stage.Process(ctx, input)
		if err == nil {
			return output, nil
		}

		lastErr = err

		// Check if this error is retryable
		if !stage.CanRetry() || !schemaErrors.ShouldRetry(err) {
			break
		}
	}

	return StageOutput{}, schemaErrors.NewProcessError(
		schemaErrors.ErrorTypeInternal,
		fmt.Sprintf("Stage '%s' failed after %d retries", stage.Name(), maxRetries),
		schemaErrors.WithCause(lastErr),
		schemaErrors.WithOperation("stage_execute"),
		schemaErrors.WithComponent(stage.Name()),
	)
}

// handleError handles pipeline errors and performs rollback if configured
func (p *Pipeline) handleError(ctx context.Context, executedStages []Stage, err error) {
	if !p.rollbackOnError {
		return
	}

	p.logger.Warn(fmt.Sprintf("Pipeline error occurred, rolling back %d stages", len(executedStages)))

	// Rollback stages in reverse order
	for i := len(executedStages) - 1; i >= 0; i-- {
		stage := executedStages[i]

		rollbackCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		if err := stage.Rollback(rollbackCtx, StageOutput{}); err != nil {
			p.logger.Error(fmt.Sprintf("Failed to rollback stage '%s': %v", stage.Name(), err))
		} else {
			p.logger.Info(fmt.Sprintf("Successfully rolled back stage '%s'", stage.Name()))
		}
		cancel()
	}
}

// GetStages returns the list of stages in the pipeline
func (p *Pipeline) GetStages() []Stage {
	return p.stages
}

// GetName returns the pipeline name
func (p *Pipeline) GetName() string {
	return p.name
}

// Validate validates the entire pipeline configuration
func (p *Pipeline) Validate() error {
	if len(p.stages) == 0 {
		return schemaErrors.NewProcessError(
			schemaErrors.ErrorTypeConfiguration,
			"Pipeline must have at least one stage",
			schemaErrors.AsUserError(),
		)
	}

	// Check for duplicate stage names
	stageNames := make(map[string]bool)
	for _, stage := range p.stages {
		name := stage.Name()
		if name == "" {
			return schemaErrors.NewProcessError(
				schemaErrors.ErrorTypeConfiguration,
				"All stages must have non-empty names",
				schemaErrors.AsUserError(),
			)
		}

		if stageNames[name] {
			return schemaErrors.NewProcessError(
				schemaErrors.ErrorTypeConfiguration,
				fmt.Sprintf("Duplicate stage name: %s", name),
				schemaErrors.AsUserError(),
			)
		}

		stageNames[name] = true
	}

	return nil
}
