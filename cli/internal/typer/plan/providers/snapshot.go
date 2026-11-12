package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/typer/plan"
)

// SnapshotFormatVersion is bumped on any change that older CLIs cannot read.
// Customers commit or cache snapshots across CLI upgrades, so the version is
// checked on load instead of failing later with a confusing parse error.
const SnapshotFormatVersion = 1

// Snapshot is a tracking plan saved so that code generation can run later
// without credentials or network access. It holds the CLI's own copy of the API
// response, so fields the CLI does not read are not kept. It contains the
// workspace and user IDs the API returns, and no credentials.
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

	if err := writeFileAtomically(path, append(data, '\n')); err != nil {
		return nil, fmt.Errorf("writing snapshot: %w", err)
	}
	return apitp, nil
}

// writeFileAtomically writes through a temporary file in the same directory and
// renames it over path. A write that fails halfway, such as on a full disk,
// then leaves the previous snapshot in place instead of a truncated file.
func writeFileAtomically(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".snapshot-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
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

	// Regenerating a newer snapshot with the same newer CLI writes the same
	// version again, so the fix there is a CLI upgrade, not a new snapshot.
	if snap.FormatVersion > SnapshotFormatVersion {
		return nil, fmt.Errorf(
			"snapshot %s has format version %d, this CLI reads version %d: upgrade rudder-cli to read it",
			p.path, snap.FormatVersion, SnapshotFormatVersion,
		)
	}
	if snap.FormatVersion != SnapshotFormatVersion {
		return nil, fmt.Errorf("%s is not a snapshot this CLI can read: regenerate it with 'typer snapshot'", p.path)
	}

	// Both writers store a non-nil events list, even for a plan with no events,
	// so a nil list means the file has no tracking plan in it.
	if snap.TrackingPlan.Events == nil {
		return nil, fmt.Errorf("snapshot %s has no tracking plan: regenerate it with 'typer snapshot'", p.path)
	}

	return planFromAPI(&snap.TrackingPlan)
}
