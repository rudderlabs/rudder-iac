package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rudderlabs/rudder-iac/typer/generator/core"
	"github.com/rudderlabs/rudder-iac/typer/generator/platforms/kotlin"
	"github.com/rudderlabs/rudder-iac/typer/plan/testutils"
)

// Writes the reference Kotlin output (and composeImmutable and common variants) into the
// validator project so `make typer-kotlin-validate` compiles all of them. Defaults are
// suitable for the in-repo Makefile target; override the destination root via
// the first arg if needed.
func main() {
	root := "typer/generator/platforms/kotlin/testdata/validator/src/main/kotlin"
	if len(os.Args) > 1 {
		root = os.Args[1]
	}

	plan := testutils.GetReferenceTrackingPlan()
	gen := &kotlin.Generator{}

	variants := []struct {
		path string
		opts kotlin.KotlinOptions
	}{
		{
			path: filepath.Join(root, "com/rudderstack/ruddertyper/Main.kt"),
			opts: kotlin.KotlinOptions{},
		},
		{
			path: filepath.Join(root, "com/rudderstack/ruddertyper/composeimmutable/Main.kt"),
			opts: kotlin.KotlinOptions{
				PackageName:      "com.rudderstack.ruddertyper.composeimmutable",
				ComposeImmutable: true,
			},
		},
		{
			path: filepath.Join(root, "com/rudderstack/ruddertyper/common/Main.kt"),
			opts: kotlin.KotlinOptions{
				PackageName: "com.rudderstack.ruddertyper.common",
				Emit:        kotlin.EmitCommon,
			},
		},
	}

	for _, v := range variants {
		files, err := gen.Generate(plan, core.GenerateOptions{RudderCLIVersion: "1.0.0"}, v.opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "generating %s: %v\n", v.path, err)
			os.Exit(1)
		}
		if err := os.MkdirAll(filepath.Dir(v.path), 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "mkdir %s: %v\n", v.path, err)
			os.Exit(1)
		}
		if err := os.WriteFile(v.path, []byte(files[0].Content), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "writing %s: %v\n", v.path, err)
			os.Exit(1)
		}
	}
}
