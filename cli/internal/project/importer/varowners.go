package importer

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/loader"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/writer"
	"github.com/rudderlabs/rudder-iac/cli/internal/varsubst"
)

// variableUse is one "{{ .NAME }}" reference: the spec key it fills, written
// as kind, spec id and key path, and the file it is in.
type variableUse struct {
	owner string
	file  string
}

// checkExistingVariableOwners refuses an import whose new specs reference a
// variable that a spec already in the project uses for another secret. A
// slice-by-slice adoption imports into a project that already manages earlier
// slices, and every provider names a secret variable from the resource id and
// key path (secret.VariableName), so a new destination "pg" can be handed the
// PG_PASSWORD an existing account "pg" fills.
//
// The same variable for the same spec key — kind, spec id and key path all
// equal — is the same secret, and is allowed. The project's spec files are
// read as written, before substitution; other files are skipped.
func checkExistingVariableOwners(location string, entities []writer.FormattableEntity) error {
	existing, err := projectVariableUses(location)
	if err != nil {
		return err
	}
	if len(existing) == 0 {
		return nil
	}

	var clashes []string
	for _, entity := range entities {
		spec, ok := entity.Content.(*specs.Spec)
		if !ok || spec == nil {
			continue
		}
		uses := map[string][]variableUse{}
		collectSpecVariableUses(spec.Kind, specIdentity(spec.Spec, spec.Metadata), spec.Spec, "spec", entity.RelativePath, uses)
		for name, newUses := range uses {
			for _, use := range newUses {
				for _, old := range existing[name] {
					if old.owner != use.owner {
						clashes = append(clashes, fmt.Sprintf("%s (in %s and new %s)", name, old.file, use.file))
					}
				}
			}
		}
	}
	if len(clashes) == 0 {
		return nil
	}
	sort.Strings(clashes)
	return fmt.Errorf("imported specs would share secret variables with specs already in the project: %s; the names come from each resource's id, so rename the new or the existing resource and import again",
		strings.Join(dedupe(clashes), ", "))
}

// projectVariableUses reads every YAML spec under location and records the
// variable references it holds.
func projectVariableUses(location string) (map[string][]variableUse, error) {
	uses := map[string][]variableUse{}
	if location == "" {
		return uses, nil
	}
	err := filepath.WalkDir(location, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != location && strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		ext := filepath.Ext(path)
		if ext != loader.ExtensionYAML && ext != loader.ExtensionYML {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading %s: %w", path, err)
		}
		rel, err := filepath.Rel(location, path)
		if err != nil {
			rel = path
		}
		decoder := yaml.NewDecoder(bytes.NewReader(raw))
		for {
			var doc struct {
				Kind     string         `yaml:"kind"`
				Metadata map[string]any `yaml:"metadata"`
				Spec     map[string]any `yaml:"spec"`
			}
			if err := decoder.Decode(&doc); err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				// Not a spec this check can read; loading the project reports it.
				return nil
			}
			if doc.Kind == "" || doc.Spec == nil {
				continue
			}
			collectSpecVariableUses(doc.Kind, specIdentity(doc.Spec, doc.Metadata), doc.Spec, "spec", rel, uses)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("reading project specs for secret variables: %w", err)
	}
	return uses, nil
}

// specIdentity names the resource a spec file declares: its spec id, else its
// metadata name.
func specIdentity(spec, metadata map[string]any) string {
	if id, ok := spec["id"].(string); ok && id != "" {
		return id
	}
	name, _ := metadata["name"].(string)
	return name
}

func collectSpecVariableUses(kind, id string, value any, path, file string, uses map[string][]variableUse) {
	switch v := value.(type) {
	case map[string]any:
		for key, item := range v {
			collectSpecVariableUses(kind, id, item, path+"."+key, file, uses)
		}
	case []any:
		for i, item := range v {
			collectSpecVariableUses(kind, id, item, fmt.Sprintf("%s[%d]", path, i), file, uses)
		}
	case string:
		for _, name := range varsubst.ExtractVariableNames([]byte(v)) {
			uses[name] = append(uses[name], variableUse{owner: kind + "/" + id + "/" + path, file: file})
		}
	}
}

func dedupe(sorted []string) []string {
	out := sorted[:0]
	for i, s := range sorted {
		if i == 0 || s != sorted[i-1] {
			out = append(out, s)
		}
	}
	return out
}
