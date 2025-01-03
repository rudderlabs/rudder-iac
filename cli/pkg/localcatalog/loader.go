package localcatalog

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/rudderlabs/rudder-iac/cli/pkg/logger"
	"gopkg.in/yaml.v3"
)

var (
	log = logger.New("localcatalog")
)

// entity group is logical grouping of entities defined
// as metadata->name in respective yaml file
type EntityGroup string

// Create a reverse lookup based on the groupName and identifier per entity
type DataCatalog struct {
	Properties    map[EntityGroup][]Property    `json:"properties"`
	Events        map[EntityGroup][]Event       `json:"events"`
	TrackingPlans map[EntityGroup]*TrackingPlan `json:"trackingPlans"` // Only one tracking plan per entity group
}

func (dc *DataCatalog) ExpandRefs() error {
	for _, tp := range dc.TrackingPlans {
		if err := tp.ExpandRefs(dc); err != nil {
			return fmt.Errorf("inflating refs for tracking plan: %w", err)
		}
	}
	return nil
}

func (dc *DataCatalog) Property(groupName string, id string) *Property {
	if props, ok := dc.Properties[EntityGroup(groupName)]; ok {
		for _, prop := range props {
			if prop.LocalID == id {
				return &prop
			}
		}
	}
	return nil
}

func (dc *DataCatalog) Event(groupName string, id string) *Event {
	if events, ok := dc.Events[EntityGroup(groupName)]; ok {
		for _, event := range events {
			if event.LocalID == id {
				return &event
			}
		}
	}
	return nil
}

func (dc *DataCatalog) TPEventRule(tpGroup, ruleID string) *TPRule {
	tp, ok := dc.TrackingPlans[EntityGroup(tpGroup)]
	if !ok {
		return nil
	}

	for _, rule := range tp.Rules {
		if rule.LocalID == ruleID && rule.Type == "event_rule" {
			return rule
		}
	}

	return nil
}

func (dc *DataCatalog) TPEventRules(tpGroup string) ([]*TPRule, bool) {
	tp, ok := dc.TrackingPlans[EntityGroup(tpGroup)]
	if !ok {
		return nil, false
	}

	var toReturn []*TPRule
	for _, rule := range tp.Rules {
		if rule.Type != "event_rule" {
			continue
		}
		toReturn = append(toReturn, rule)
	}

	return toReturn, true
}

// Read simply reads the directory for files which contain
// data catalog entities defined. It parses all the folders for files which can contain resource
// definitions
func Read(loc string) (*DataCatalog, error) {
	log.Info("reading data catalog entities", "location", loc)

	var abspath = loc

	abs := filepath.IsAbs(loc)
	if !abs {
		abspath = filepath.Join(".", loc)
	}

	s, err := os.Stat(abspath)
	if err != nil {
		return nil, fmt.Errorf("checking if path exists: %w", err)
	}

	var (
		files []string
	)

	if !s.IsDir() {
		files = []string{abspath}
	} else {
		files, err = getFiles(abspath)
		if err != nil {
			return nil, fmt.Errorf("reading all files in directory: %w", err)
		}
	}

	dc := &DataCatalog{
		Properties:    map[EntityGroup][]Property{},
		Events:        map[EntityGroup][]Event{},
		TrackingPlans: map[EntityGroup]*TrackingPlan{},
	}

	for _, file := range files {
		log.Debug("loading entities from file into catalog", "file", file)

		log.Info("extension", "file", file, "ext", filepath.Ext(file))

		if filepath.Ext(file) != ".yaml" && filepath.Ext(file) != ".yml" {
			log.Debug("skipping file, not a yaml file", "file", file)
			continue
		}

		byt, err := getFileBytes(file)
		if err != nil {
			return nil, fmt.Errorf("fetching file: %s data: %w", file, err)
		}

		if err := extractEntities(byt, dc); err != nil {
			return nil, fmt.Errorf("extracting data catalog entity from file: %s : %w", file, err)
		}
	}

	// Once the entities are extracted, we need to inflate the references
	return dc, nil
}

// extractEntities parses the entity from file bytes
// and updates the datacatalog struct with it.
func extractEntities(byt []byte, dc *DataCatalog) error {
	def := ResourceDefinition{}

	if err := yaml.Unmarshal(byt, &def); err != nil {
		return fmt.Errorf("unmarshalling resource: %w", err)
	}

	switch def.Kind {
	case "properties":
		properties, err := ExtractProperties(&def)
		if err != nil {
			return fmt.Errorf("extracting properties: %w", err)
		}
		dc.Properties[EntityGroup(def.Metadata.Name)] = properties

	case "events":
		events, err := ExtractEvents(&def)
		if err != nil {
			return fmt.Errorf("extracting property entity: %w", err)
		}
		dc.Events[EntityGroup(def.Metadata.Name)] = events

	case "tp":
		tp, err := ExtractTrackingPlan(&def)
		if err != nil {
			return fmt.Errorf("extracting tracking plan: %w", err)
		}

		dc.TrackingPlans[EntityGroup(def.Metadata.Name)] = &tp

	default:
		return fmt.Errorf("unknown kind: %s", def.Kind)
	}

	return nil
}

func getFileBytes(fname string) ([]byte, error) {
	f, err := os.Open(fname)
	if err != nil {
		return nil, fmt.Errorf("opening file: %w", err)
	}
	defer f.Close()

	byt, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	return byt, nil
}

func getFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading directory: %s for files: %w", dir, err)
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			f, err := getFiles(fmt.Sprintf("%s/%s", dir, entry.Name()))
			if err != nil {
				return nil, fmt.Errorf("getting files: %w", err)
			}
			files = append(files, f...)
			continue
		}
		files = append(files, fmt.Sprintf("%s/%s", dir, entry.Name()))
	}

	return files, nil
}
