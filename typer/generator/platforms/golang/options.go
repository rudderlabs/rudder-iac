package golang

import (
	"fmt"
	"go/build"
	"go/token"
	"go/types"
	"io"
	"regexp"
	"strings"
)

// GoOptions contains Go-specific generation options.
type GoOptions struct {
	PackageName    string `mapstructure:"packageName" description:"Package name of the generated Go file (e.g., ruddertyper)"`
	OutputFileName string `mapstructure:"outputFileName" description:"Name of the generated Go file (e.g., ruddertyper.go)"`
}

// DefaultOptions returns the default Go options.
func (g *Generator) DefaultOptions() any {
	return GoOptions{
		PackageName:    "ruddertyper",
		OutputFileName: "ruddertyper.go",
	}
}

var packageNameRegex = regexp.MustCompile(`^[a-z][a-z0-9]*$`)

// Validate reports options that would produce a package Go tooling rejects or
// builds only on some platforms.
func (o GoOptions) Validate() error {
	if err := validatePackageName(o.PackageName); err != nil {
		return fmt.Errorf("invalid packageName %q: %w", o.PackageName, err)
	}
	if err := validateOutputFileName(o.OutputFileName); err != nil {
		return fmt.Errorf("invalid outputFileName %q: %w", o.OutputFileName, err)
	}
	return nil
}

func validatePackageName(name string) error {
	switch {
	case !packageNameRegex.MatchString(name):
		return fmt.Errorf("must match %s", packageNameRegex)
	case name == "main":
		return fmt.Errorf("package main is a command and cannot be imported")
	case token.IsKeyword(name):
		return fmt.Errorf("it is a Go keyword")
	case types.Universe.Lookup(name) != nil:
		return fmt.Errorf("it is a predeclared Go identifier")
	}
	return nil
}

func validateOutputFileName(name string) error {
	switch {
	case !strings.HasSuffix(name, ".go"):
		return fmt.Errorf("must end in .go")
	case strings.HasSuffix(name, "_test.go"):
		return fmt.Errorf("go build ignores _test.go files")
	case strings.ContainsAny(name, `/\`):
		return fmt.Errorf("must be a file name, not a path")
	case strings.HasPrefix(name, "_"), strings.HasPrefix(name, "."):
		return fmt.Errorf("go build ignores files starting with _ or .")
	}

	// Under an OS and architecture that do not exist, go/build accepts only
	// file names without a _GOOS or _GOARCH suffix, i.e. those built everywhere.
	ctx := build.Context{
		GOOS:   "fakeos",
		GOARCH: "fakearch",
		OpenFile: func(string) (io.ReadCloser, error) {
			return io.NopCloser(strings.NewReader("package p\n")), nil
		},
	}
	ok, err := ctx.MatchFile(".", name)
	if err != nil {
		return fmt.Errorf("matching build constraints: %w", err)
	}
	if !ok {
		return fmt.Errorf("a _GOOS or _GOARCH suffix restricts the file to one platform")
	}
	return nil
}
