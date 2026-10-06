package golang_test

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/typer/generator/core"
	"github.com/rudderlabs/rudder-iac/typer/generator/platforms/golang"
	"github.com/rudderlabs/rudder-iac/typer/plan"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOptionsValidate(t *testing.T) {
	tests := []struct {
		name    string
		options golang.GoOptions
		wantErr string
	}{
		{"defaults", golang.GoOptions{PackageName: "ruddertyper", OutputFileName: "ruddertyper.go"}, ""},
		{"digits after the first letter", golang.GoOptions{PackageName: "events2", OutputFileName: "events.go"}, ""},
		{"file name with an OS-like stem", golang.GoOptions{PackageName: "ruddertyper", OutputFileName: "linux.go"}, ""},
		{"file name with an underscore", golang.GoOptions{PackageName: "ruddertyper", OutputFileName: "rudder_typer.go"}, ""},
		{
			"upper-case package name",
			golang.GoOptions{PackageName: "RudderTyper", OutputFileName: "ruddertyper.go"},
			`invalid packageName "RudderTyper": must match ^[a-z][a-z0-9]*$`,
		},
		{
			"underscore in package name",
			golang.GoOptions{PackageName: "rudder_typer", OutputFileName: "ruddertyper.go"},
			`invalid packageName "rudder_typer": must match ^[a-z][a-z0-9]*$`,
		},
		{
			"package name starting with a digit",
			golang.GoOptions{PackageName: "1typer", OutputFileName: "ruddertyper.go"},
			`invalid packageName "1typer": must match ^[a-z][a-z0-9]*$`,
		},
		{
			"empty package name",
			golang.GoOptions{PackageName: "", OutputFileName: "ruddertyper.go"},
			`invalid packageName "": must match ^[a-z][a-z0-9]*$`,
		},
		{
			"main",
			golang.GoOptions{PackageName: "main", OutputFileName: "ruddertyper.go"},
			`invalid packageName "main": package main is a command and cannot be imported`,
		},
		{
			"keyword",
			golang.GoOptions{PackageName: "func", OutputFileName: "ruddertyper.go"},
			`invalid packageName "func": it is a Go keyword`,
		},
		{
			"predeclared type",
			golang.GoOptions{PackageName: "string", OutputFileName: "ruddertyper.go"},
			`invalid packageName "string": it is a predeclared Go identifier`,
		},
		{
			"predeclared function",
			golang.GoOptions{PackageName: "len", OutputFileName: "ruddertyper.go"},
			`invalid packageName "len": it is a predeclared Go identifier`,
		},
		{
			"not a Go file",
			golang.GoOptions{PackageName: "ruddertyper", OutputFileName: "ruddertyper.txt"},
			`invalid outputFileName "ruddertyper.txt": must end in .go`,
		},
		{
			"test file",
			golang.GoOptions{PackageName: "ruddertyper", OutputFileName: "ruddertyper_test.go"},
			`invalid outputFileName "ruddertyper_test.go": go build ignores _test.go files`,
		},
		{
			"path",
			golang.GoOptions{PackageName: "ruddertyper", OutputFileName: "gen/ruddertyper.go"},
			`invalid outputFileName "gen/ruddertyper.go": must be a file name, not a path`,
		},
		{
			"windows path",
			golang.GoOptions{PackageName: "ruddertyper", OutputFileName: `gen\ruddertyper.go`},
			`invalid outputFileName "gen\\ruddertyper.go": must be a file name, not a path`,
		},
		{
			"leading underscore",
			golang.GoOptions{PackageName: "ruddertyper", OutputFileName: "_ruddertyper.go"},
			`invalid outputFileName "_ruddertyper.go": go build ignores files starting with _ or .`,
		},
		{
			"leading dot",
			golang.GoOptions{PackageName: "ruddertyper", OutputFileName: ".ruddertyper.go"},
			`invalid outputFileName ".ruddertyper.go": go build ignores files starting with _ or .`,
		},
		{
			"GOOS suffix",
			golang.GoOptions{PackageName: "ruddertyper", OutputFileName: "ruddertyper_linux.go"},
			`invalid outputFileName "ruddertyper_linux.go": a _GOOS or _GOARCH suffix restricts the file to one platform`,
		},
		{
			"GOARCH suffix",
			golang.GoOptions{PackageName: "ruddertyper", OutputFileName: "ruddertyper_arm64.go"},
			`invalid outputFileName "ruddertyper_arm64.go": a _GOOS or _GOARCH suffix restricts the file to one platform`,
		},
		{
			"GOOS and GOARCH suffix",
			golang.GoOptions{PackageName: "ruddertyper", OutputFileName: "ruddertyper_windows_amd64.go"},
			`invalid outputFileName "ruddertyper_windows_amd64.go": a _GOOS or _GOARCH suffix restricts the file to one platform`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.options.Validate()
			if tt.wantErr == "" {
				assert.NoError(t, err)
				return
			}
			assert.EqualError(t, err, tt.wantErr)
		})
	}
}

func TestGenerateOptions(t *testing.T) {
	gen := &golang.Generator{}

	t.Run("defaults", func(t *testing.T) {
		assert.Equal(t, golang.GoOptions{PackageName: "ruddertyper", OutputFileName: "ruddertyper.go"}, gen.DefaultOptions())
	})

	t.Run("unset options take their defaults", func(t *testing.T) {
		files, err := gen.Generate(&plan.TrackingPlan{}, core.GenerateOptions{}, golang.GoOptions{PackageName: "events"})
		require.NoError(t, err)
		require.Len(t, files, 1)
		assert.Equal(t, "ruddertyper.go", files[0].Path)
		assert.Contains(t, files[0].Content, "\npackage events\n")
	})

	t.Run("invalid options fail generation", func(t *testing.T) {
		_, err := gen.Generate(&plan.TrackingPlan{}, core.GenerateOptions{}, golang.GoOptions{OutputFileName: "events_test.go"})
		assert.EqualError(t, err, `invalid outputFileName "events_test.go": go build ignores _test.go files`)
	})
}
