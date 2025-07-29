package kotlin

import (
	_ "embed"

	"github.com/rudderlabs/rudder-iac/cli/internal/typer/generator/core"
	"github.com/rudderlabs/rudder-iac/cli/internal/typer/plan"
)

func Generate(plan *plan.TrackingPlan) ([]*core.File, error) {
	ctx := &RootContext{
		Name: plan.Name,
	}

	mainFile, err := GenerateFile("Main.kt", kotlinTemplate, ctx)
	if err != nil {
		return nil, err
	}

	return []*core.File{
		mainFile,
	}, nil
}
