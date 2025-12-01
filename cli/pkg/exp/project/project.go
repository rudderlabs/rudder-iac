package project

import (
	"context"

	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/project"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

// ReadOnlyGraph provides read-only access to a resource dependency graph.
// It allows querying resources and their relationships without modifying the graph.
type ReadOnlyGraph interface {
	// Resources returns a map of all resources in the graph, keyed by their URN.
	Resources() map[string]*resources.Resource
	// GetResource retrieves a specific resource by its URN.
	// Returns the resource and true if found, nil and false otherwise.
	GetResource(urn string) (*resources.Resource, bool)
	// GetDependencies returns the URNs of all resources that the specified resource depends on.
	GetDependencies(urn string) []string
	// GetDependents returns the URNs of all resources that depend on the specified resource.
	GetDependents(urn string) []string
}

// Load loads a Rudder CLI project from the specified location and returns a read-only resource graph.
// The location parameter specifies the directory path containing the project YAML specs.
// Returns a ReadOnlyGraph interface for querying project resources and their dependencies,
// or an error if the project cannot be loaded or the resource graph cannot be constructed.
func Load(_ context.Context, location string) (ReadOnlyGraph, error) {
	config.InitConfig("")
	deps, err := app.NewDeps()
	if err != nil {
		return nil, err
	}

	p := project.New(location, deps.CompositeProvider())
	if err := p.Load(); err != nil {
		return nil, err
	}

	return p.ResourceGraph()
}
