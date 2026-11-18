// Package importreport records remote config an import cannot carry into specs.
//
// A destination adopted by apply gets the whole spec config as a PUT, which
// replaces the remote config. A key the CLI does not model is in the remote
// config but cannot be in the spec, so adopting removes it. `import workspace`
// writes one entry per such destination to FileName, next to the imported
// specs. Apply refuses to adopt a destination while its entry is missing or not
// acknowledged, so the loss is a decision a person made in Git.
//
// The report holds paths and value shapes, never values: an unmodelled key can
// hold a credential the definition does not know is one.
package importreport

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
)

// FileName is the report's name. JSON keeps the project loader, which reads
// only YAML, from mistaking it for a spec.
const FileName = "import-report.json"

// Lost is one part of a remote config that adopting removes.
type Lost struct {
	// Path is the API spelling, such as consentManagement.amp or headers[0].description.
	Path string `json:"path"`
	// Value summarises what is removed, for example "(object, 2 keys)".
	Value string `json:"value"`
}

// Entry is the config one importable resource loses.
type Entry struct {
	URN      string `json:"urn"`
	RemoteID string `json:"remote_id"`
	Name     string `json:"name"`
	Lost     []Lost `json:"lost"`
	// Acknowledged is set by a person after reading Lost. It covers the paths
	// listed here and no others.
	Acknowledged bool `json:"acknowledged"`
}

// Report is the content of one FileName.
type Report struct {
	Entries []Entry `json:"entries"`
}

// Uncovered returns the parts of lost that the report does not acknowledge for
// remoteID: all of them when there is no entry or it is not acknowledged, and
// otherwise the paths that appeared after the entry was written.
func (r *Report) Uncovered(remoteID string, lost []Lost) []Lost {
	if r == nil {
		return lost
	}
	for _, entry := range r.Entries {
		if entry.RemoteID != remoteID {
			continue
		}
		if !entry.Acknowledged {
			return lost
		}
		var uncovered []Lost
		for _, item := range lost {
			if !slices.ContainsFunc(entry.Lost, func(l Lost) bool { return l.Path == item.Path }) {
				uncovered = append(uncovered, item)
			}
		}
		return uncovered
	}
	return lost
}

// Write saves entries to dir/FileName. An entry already there for another
// remote ID stays, with its acknowledgement, so a second import cannot undo a
// review of a destination that is not adopted yet. A new entry replaces the old
// one for the same remote ID.
func Write(dir string, entries []Entry) error {
	if len(entries) == 0 {
		return nil
	}

	path := filepath.Join(dir, FileName)
	merged := slices.Clone(entries)
	existing, err := readFile(path)
	if err != nil {
		return err
	}
	for _, old := range existing.Entries {
		if !slices.ContainsFunc(entries, func(e Entry) bool { return e.RemoteID == old.RemoteID }) {
			merged = append(merged, old)
		}
	}
	slices.SortFunc(merged, func(a, b Entry) int {
		if a.URN < b.URN {
			return -1
		}
		if a.URN > b.URN {
			return 1
		}
		return 0
	})

	data, err := json.MarshalIndent(Report{Entries: merged}, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling import report: %w", err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating import report directory: %w", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("writing import report: %w", err)
	}
	return nil
}

// Load merges every FileName under location, which is a directory or a file in
// one. A project with no report gets an empty one, so nothing is acknowledged.
func Load(location string) (*Report, error) {
	root := location
	if info, err := os.Stat(location); err == nil && !info.IsDir() {
		root = filepath.Dir(location)
	}

	report := &Report{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || d.Name() != FileName {
			return nil
		}
		file, err := readFile(path)
		if err != nil {
			return err
		}
		report.Entries = append(report.Entries, file.Entries...)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("loading import report: %w", err)
	}
	return report, nil
}

func readFile(path string) (*Report, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &Report{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading import report %s: %w", path, err)
	}
	var report Report
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("parsing import report %s: %w", path, err)
	}
	return &report, nil
}
