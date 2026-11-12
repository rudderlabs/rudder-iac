package providers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/typer/plan"
)

// SnapshotFormatVersion is bumped on any change that older CLIs cannot read.
// Customers commit or cache snapshots across CLI upgrades, so the version is
// checked on load instead of failing later with a confusing parse error.
const SnapshotFormatVersion = 1

// Snapshot is a tracking plan saved so that code generation can run later
// without an access token or API calls. It holds the CLI's own copy of the API
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

	sortEvents(apitp.Events)

	data, err := json.MarshalIndent(Snapshot{FormatVersion: SnapshotFormatVersion, TrackingPlan: *apitp}, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encoding snapshot: %w", err)
	}

	if err := writeFileAtomically(path, append(data, '\n')); err != nil {
		return nil, fmt.Errorf("writing snapshot: %w", err)
	}
	return apitp, nil
}

// sortEvents puts the events in a fixed order. The API returns them in the
// order its pages arrive, which varies between runs, and a build that treats
// the snapshot as an input would otherwise rerun on a file that only reordered.
// The ID can be empty for a pinned version, so type and name come first.
func sortEvents(events []catalog.TrackingPlanEventSchema) {
	key := func(e catalog.TrackingPlanEventSchema) string {
		return strings.Join([]string{e.EventType, e.Name, e.ID}, "\x00")
	}
	slices.SortStableFunc(events, func(a, b catalog.TrackingPlanEventSchema) int {
		return strings.Compare(key(a), key(b))
	})
}

// writeFileAtomically writes through a temporary file in the same directory and
// renames it over the target. A write that fails halfway, such as on a full disk,
// then leaves the previous snapshot in place instead of a truncated file.
//
// It keeps what an in-place write would: a symlink at path stays a symlink and
// the file it points to is replaced, an existing file keeps its mode, and a new
// file gets the mode the umask allows.
func writeFileAtomically(path string, data []byte) error {
	resolved, err := filepath.EvalSymlinks(path)
	switch {
	case err == nil:
		path = resolved
	case !errors.Is(err, os.ErrNotExist):
		return err
	}

	var (
		mode    os.FileMode
		existed bool
	)
	if info, err := os.Stat(path); err == nil {
		mode, existed = info.Mode().Perm(), true
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), ".snapshot-*")
	if err != nil {
		// A writable file in a directory the user cannot write to cannot be
		// replaced, but it can still be updated, as before
		if existed {
			return os.WriteFile(path, data, mode)
		}
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

	if !existed {
		if mode, err = newFileMode(path); err != nil {
			return err
		}
	}
	if err := os.Chmod(tmp.Name(), mode); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// newFileMode returns the mode a new file at path would get, which depends on
// the process umask that the standard library does not expose. It creates the
// file with the default permissions to find out, and the rename that follows
// replaces it.
func newFileMode(path string) (os.FileMode, error) {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o666)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		os.Remove(path)
		return 0, err
	}
	return info.Mode().Perm(), nil
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
