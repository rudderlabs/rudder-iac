//go:build ignore

// Throwaway harness: captures the remote GetTrackingPlanWithSchemas response for
// the applied testdata/project tracking plan into a golden JSON file, used by the
// local-vs-remote equivalence test. Run with `go run capture_golden.go`. Not part
// of the build.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
)

const externalID = "typer-test-tracking-plan"
const out = "cli/internal/typer/plan/providers/testdata/remote_tracking_plan.golden.json"

func main() {
	config.InitConfig("")

	deps, err := app.NewDeps()
	check(err)

	dc, err := catalog.NewRudderDataCatalog(deps.Client())
	check(err)

	ctx := context.Background()
	tps, err := dc.GetTrackingPlans(ctx, catalog.ListOptions{})
	check(err)

	var id string
	for _, tp := range tps {
		if tp.ExternalID == externalID {
			id = tp.ID
			break
		}
	}
	if id == "" {
		fmt.Fprintf(os.Stderr, "tracking plan with externalId %q not found (have %d plans)\n", externalID, len(tps))
		os.Exit(1)
	}

	full, err := dc.GetTrackingPlanWithSchemas(ctx, id)
	check(err)

	data, err := json.MarshalIndent(full, "", "  ")
	check(err)
	check(os.WriteFile(out, append(data, '\n'), 0o644))
	fmt.Printf("wrote %s (tp %s, %d events)\n", out, id, len(full.Events))
}

func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
