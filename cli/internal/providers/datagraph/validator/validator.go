package validator

import (
	"context"
	"errors"
	"fmt"
	"io"

	dgClient "github.com/rudderlabs/rudder-iac/api/client/datagraph"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
)

var ErrValidationFailed = errors.New("one or more validations failed")

// Mode is a sealed interface representing the validation mode.
// Implemented by ModeAll, ModeModified, and ModeSingle.
type Mode interface {
	validationMode()
}

type ModeAll struct{}
type ModeModified struct{}
type ModeSingle struct {
	ResourceType string
	TargetID     string
}

func (ModeAll) validationMode()      {}
func (ModeModified) validationMode() {}
func (ModeSingle) validationMode()   {}

// ValidatorProvider abstracts the provider methods needed for validation
type ValidatorProvider interface {
	provider.ManagedRemoteResourceLoader
	provider.StateLoader
	Client() dgClient.DataGraphClient
}

// Project abstracts the project methods needed for validation
type Project interface {
	ResourceGraph() (*resources.Graph, error)
}

// DisplayFunc formats and renders a validation report.
type DisplayFunc func(report *ValidationReport)

// Config holds configuration for a validation run
type Config struct {
	Mode        Mode
	WorkspaceID string
	JSONOutput  bool
	Writer      io.Writer
	DisplayFunc DisplayFunc
	Concurrency int
}

// Validate orchestrates a complete validation run: builds a resource graph,
// runs validations, and displays results.
func Validate(ctx context.Context, project Project, p ValidatorProvider, cfg Config) error {
	graph, err := project.ResourceGraph()
	if err != nil {
		return fmt.Errorf("getting resource graph: %w", err)
	}

	var reporter ValidationReporter
	if cfg.JSONOutput {
		reporter = noopReporter{}
	} else {
		reporter = newProgressReporterIfTerminal()
	}

	runner := NewRunner(p.Client(), p, graph, reporter, cfg.Concurrency)
	report, err := runner.Run(ctx, cfg.Mode, cfg.WorkspaceID)
	if err != nil {
		return fmt.Errorf("running validations: %w", err)
	}

	if report.Status == RunStatusNoResources {
		if cfg.JSONOutput {
			cfg.DisplayFunc(report)
		} else {
			fmt.Fprintln(cfg.Writer, "No resources to validate")
		}
		return nil
	}

	cfg.DisplayFunc(report)

	if report.HasFailures() {
		return ErrValidationFailed
	}

	return nil
}

// progressReporter wraps ui.TaskReporter to implement ValidationReporter
// with lifecycle methods for starting and stopping the UI.
type progressReporter struct {
	reporter *ui.TaskReporter
}

func (p *progressReporter) TaskStarted(id, description string) {
	p.reporter.Start(id, description)
}

func (p *progressReporter) TaskCompleted(id, description string, err error) {
	p.reporter.Complete(id, description, err)
}

func (p *progressReporter) start(total int) {
	p.reporter = ui.NewTaskReporter(total)
	go p.reporter.Run() //nolint:errcheck
}

func (p *progressReporter) done() {
	if p.reporter != nil {
		p.reporter.Done()
	}
}

func newProgressReporterIfTerminal() ValidationReporter {
	if !ui.IsTerminal() {
		return noopReporter{}
	}
	return &progressReporter{}
}
