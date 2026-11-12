package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/typer/plan"
)

// SnapshotFormatVersion is bumped on any change that older CLIs cannot read.
// Customers commit or cache snapshots across CLI upgrades, so the version is
// checked on load instead of failing later with a confusing parse error.
const SnapshotFormatVersion = 1

// Snapshot is a tracking plan as the API returned it, saved so that code
// generation can run later without credentials or network access.
type Snapshot struct {
	FormatVersion int                             `json:"formatVersion"`
	TrackingPlan  catalog.TrackingPlanWithSchemas `json:"trackingPlan"`
}

// WriteSnapshot fetches the tracking plan, or the given version when it is not
// zero, and writes it to path.
func WriteSnapshot(ctx context.Context, store catalog.TrackingPlanStore, trackingPlanID string, version int, path string) (*catalog.TrackingPlanWithSchemas, error) {
	apitp, err := fetchTrackingPlan(ctx, store, trackingPlanID, version)
	if err != nil {
		return nil, fmt.Errorf("fetching tracking plan: %w", err)
	}

	data, err := json.MarshalIndent(Snapshot{FormatVersion: SnapshotFormatVersion, TrackingPlan: *apitp}, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encoding snapshot: %w", err)
	}

	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		return nil, fmt.Errorf("writing snapshot: %w", err)
	}
	return apitp, nil
}

// SnapshotPlanProvider builds a tracking plan from a snapshot file.
type SnapshotPlanProvider struct {
	path string
}

func NewSnapshotPlanProvider(path string) *SnapshotPlanProvider {
	return &SnapshotPlanProvider{path: path}
}

func (p *SnapshotPlanProvider) GetTrackingPlan(_ context.Context) (*plan.TrackingPlan, error) {
	data, err := os.ReadFile(p.path)
	if err != nil {
		return nil, fmt.Errorf("reading snapshot: %w", err)
	}

	var snap Snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return nil, fmt.Errorf("decoding snapshot %s: %w", p.path, err)
	}

	if snap.FormatVersion != SnapshotFormatVersion {
		return nil, fmt.Errorf(
			"snapshot %s has format version %d, this CLI reads version %d: regenerate it with 'typer snapshot'",
			p.path, snap.FormatVersion, SnapshotFormatVersion,
		)
	}

	return planFromAPI(&snap.TrackingPlan)
}
