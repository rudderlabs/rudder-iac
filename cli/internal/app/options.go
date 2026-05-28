package app

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/project"
	"github.com/rudderlabs/rudder-iac/cli/internal/varsubst"
	"github.com/rudderlabs/rudder-iac/cli/internal/varsubst/resolver"
)

// NewProjectOptions assembles the project options that should be applied to
// every project created via Deps.NewProject. Each capability is gated by its
// own experimental flag in cfg; when a flag is off, the related option is
// omitted entirely. Additional capabilities should be wired in here so all
// command call sites pick them up uniformly.
func NewProjectOptions(cfg config.Config, varFiles []string) ([]project.ProjectOption, error) {
	var opts []project.ProjectOption

	if cfg.ExperimentalFlags.EnableVarSubstitution {
		sub, err := buildSubstitutor(varFiles)
		if err != nil {
			return nil, err
		}
		opts = append(opts, project.WithSubstitutor(sub))
	}

	return opts, nil
}

// buildSubstitutor wires the standard resolver chain: env resolver first
// (highest priority), then a FileResolver per varFile in the order provided.
func buildSubstitutor(varFiles []string) (varsubst.Substitutor, error) {
	envR, err := resolver.NewEnvResolver()
	if err != nil {
		return nil, fmt.Errorf("initialising env resolver: %w", err)
	}

	resolvers := []varsubst.Resolver{envR}
	for _, path := range varFiles {
		r, err := resolver.NewFileResolver(path)
		if err != nil {
			return nil, fmt.Errorf("initialising file resolver: %w", err)
		}
		resolvers = append(resolvers, r)
	}

	return varsubst.NewSubstitutor(resolvers...), nil
}
