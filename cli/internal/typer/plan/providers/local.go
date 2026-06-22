package providers

import (
	"context"
	"fmt"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/typer/plan"
)

// LocalCatalogPlanProvider builds a tracking plan from local YAML specs so that
// `typer generate --local` can produce code without fetching from (or applying
// to) the remote workspace. It renders the loaded catalog into the same
// JSON-Schema representation the remote returns and reuses BuildTrackingPlan, so
// local and remote output stay identical.
type LocalCatalogPlanProvider struct {
	dc             *localcatalog.DataCatalog
	trackingPlanID string
}

func NewLocalCatalogPlanProvider(dc *localcatalog.DataCatalog, trackingPlanID string) *LocalCatalogPlanProvider {
	return &LocalCatalogPlanProvider{dc: dc, trackingPlanID: trackingPlanID}
}

func (p *LocalCatalogPlanProvider) GetTrackingPlan(_ context.Context) (*plan.TrackingPlan, error) {
	var tp *localcatalog.TrackingPlanV1
	for _, t := range p.dc.TrackingPlans {
		if t.LocalID == p.trackingPlanID {
			tp = t
			break
		}
	}
	if tp == nil {
		return nil, fmt.Errorf("tracking plan %q not found in local specs", p.trackingPlanID)
	}

	// ExpandRefs resolves every event/property/custom-type reference in the rules
	// into tp.EventProps, leaving custom-typed properties as ref strings that the
	// renderer turns into $defs entries.
	if err := tp.ExpandRefs(p.dc); err != nil {
		return nil, fmt.Errorf("expanding tracking plan refs: %w", err)
	}

	apitp, err := renderTrackingPlanSchemas(tp, p.dc)
	if err != nil {
		return nil, err
	}

	return BuildTrackingPlan(apitp)
}

func renderTrackingPlanSchemas(tp *localcatalog.TrackingPlanV1, dc *localcatalog.DataCatalog) (*catalog.TrackingPlanWithSchemas, error) {
	events := make([]catalog.TrackingPlanEventSchema, 0, len(tp.EventProps))

	for _, ev := range tp.EventProps {
		if len(ev.Variants) > 0 {
			return nil, errVariantUnsupported(fmt.Sprintf("event %q", ev.Name))
		}

		section := ev.IdentitySection
		if section == "" {
			section = "properties"
		}

		defs := map[string]any{}
		objSchema, err := renderObjectSchema(ev.Properties, ev.AllowUnplanned, dc, defs)
		if err != nil {
			return nil, fmt.Errorf("rendering event %q: %w", ev.Name, err)
		}

		sectionProps, err := wrapIdentitySection(section, objSchema)
		if err != nil {
			return nil, fmt.Errorf("rendering event %q: %w", ev.Name, err)
		}

		schema := catalog.TrackingPlanEventSchema{
			Name:            ev.Name,
			Description:     ev.Description,
			EventType:       ev.Type,
			IdentitySection: section,
		}
		schema.Rules.Type = "object"
		schema.Rules.Properties = sectionProps
		if len(defs) > 0 {
			schema.Rules.Defs = defs
		}

		events = append(events, schema)
	}

	return &catalog.TrackingPlanWithSchemas{
		ID:     tp.LocalID,
		Name:   tp.Name,
		Events: events,
	}, nil
}

// wrapIdentitySection nests the event's object schema under the key path the
// parser expects for the given identity section.
func wrapIdentitySection(section string, objSchema map[string]any) (map[string]any, error) {
	switch section {
	case "properties", "traits":
		return map[string]any{section: objSchema}, nil
	case "context.traits":
		return map[string]any{
			"context": map[string]any{
				"properties": map[string]any{
					"traits": objSchema,
				},
			},
		}, nil
	default:
		return nil, fmt.Errorf("unsupported identity section %q", section)
	}
}

func renderObjectSchema(props []*localcatalog.TPEventProperty, additionalProps bool, dc *localcatalog.DataCatalog, defs map[string]any) (map[string]any, error) {
	properties := map[string]any{}
	var required []any

	for _, prop := range props {
		node, err := renderProperty(prop, dc, defs)
		if err != nil {
			return nil, err
		}
		properties[prop.Name] = node
		if prop.Required {
			required = append(required, prop.Name)
		}
	}

	schema := map[string]any{
		"type":                 "object",
		"additionalProperties": additionalProps,
		"properties":           properties,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema, nil
}

// renderProperty renders a tracking-plan rule property. Object properties take
// their nested fields from the rule (prop.Properties), unlike custom types.
func renderProperty(prop *localcatalog.TPEventProperty, dc *localcatalog.DataCatalog, defs map[string]any) (map[string]any, error) {
	if ref, ok, err := renderCustomRef(prop.Type, dc, defs); err != nil {
		return nil, err
	} else if ok {
		return ref, nil
	}

	if len(prop.Types) > 0 {
		return withEnum(map[string]any{"type": toAnySlice(prop.Types)}, prop.Config), nil
	}

	switch prop.Type {
	case "object":
		return renderObjectSchema(prop.Properties, derefBool(prop.AdditionalProperties, true), dc, defs)
	case "array":
		node := map[string]any{"type": "array"}
		items, err := renderArrayItems(prop.ItemType, prop.ItemTypes, dc, defs)
		if err != nil {
			return nil, err
		}
		if items != nil {
			node["items"] = items
		}
		return withEnum(node, prop.Config), nil
	case "":
		return nil, fmt.Errorf("property %q has no type", prop.Name)
	default:
		return withEnum(map[string]any{"type": prop.Type}, prop.Config), nil
	}
}

// renderCustomRef detects a custom-type reference, registers the referenced type
// into defs, and returns a $ref node. ok is false when typeStr is not a ref.
func renderCustomRef(typeStr string, dc *localcatalog.DataCatalog, defs map[string]any) (map[string]any, bool, error) {
	m := localcatalog.CustomTypeRegex.FindStringSubmatch(typeStr)
	if len(m) != 2 {
		return nil, false, nil
	}

	ct := dc.CustomType(m[1])
	if ct == nil {
		return nil, false, fmt.Errorf("custom type %q not found in local specs", m[1])
	}
	if err := registerCustomType(ct, dc, defs); err != nil {
		return nil, false, err
	}
	return map[string]any{"$ref": "#/$defs/" + ct.Name}, true, nil
}

func registerCustomType(ct *localcatalog.CustomTypeV1, dc *localcatalog.DataCatalog, defs map[string]any) error {
	if _, exists := defs[ct.Name]; exists {
		return nil
	}
	if len(ct.Variants) > 0 {
		return errVariantUnsupported(fmt.Sprintf("custom type %q", ct.Name))
	}

	// Reserve the name before rendering children so a self/mutual reference does
	// not recurse forever.
	defs[ct.Name] = map[string]any{}

	node, err := renderCustomTypeNode(ct, dc, defs)
	if err != nil {
		return err
	}
	defs[ct.Name] = node
	return nil
}

// renderCustomTypeNode renders a custom type definition. Object custom types take
// their fields from ct.Properties (each a reference to a property).
func renderCustomTypeNode(ct *localcatalog.CustomTypeV1, dc *localcatalog.DataCatalog, defs map[string]any) (map[string]any, error) {
	switch ct.Type {
	case "object":
		properties := map[string]any{}
		var required []any
		for _, ctp := range ct.Properties {
			m := localcatalog.PropRegex.FindStringSubmatch(ctp.Property)
			if len(m) != 2 {
				return nil, fmt.Errorf("custom type %q: invalid property ref %q", ct.Name, ctp.Property)
			}
			prop := dc.Property(m[1])
			if prop == nil {
				return nil, fmt.Errorf("custom type %q: property %q not found", ct.Name, m[1])
			}
			node, err := renderPropertyDefinition(prop, dc, defs)
			if err != nil {
				return nil, err
			}
			properties[prop.Name] = node
			if ctp.Required {
				required = append(required, prop.Name)
			}
		}
		schema := map[string]any{"type": "object", "properties": properties}
		if len(required) > 0 {
			schema["required"] = required
		}
		return schema, nil
	case "array":
		node := map[string]any{"type": "array"}
		items, err := renderArrayItems(ct.ItemType, ct.ItemTypes, dc, defs)
		if err != nil {
			return nil, err
		}
		if items != nil {
			node["items"] = items
		}
		return withEnum(node, ct.Config), nil
	default:
		return withEnum(map[string]any{"type": ct.Type}, ct.Config), nil
	}
}

// renderPropertyDefinition renders a property from its definition (used for the
// fields of an object custom type). Unlike rule properties, a bare object
// property carries no nested fields.
func renderPropertyDefinition(prop *localcatalog.PropertyV1, dc *localcatalog.DataCatalog, defs map[string]any) (map[string]any, error) {
	if ref, ok, err := renderCustomRef(prop.Type, dc, defs); err != nil {
		return nil, err
	} else if ok {
		return ref, nil
	}

	if len(prop.Types) > 0 {
		return withEnum(map[string]any{"type": toAnySlice(prop.Types)}, prop.Config), nil
	}

	switch prop.Type {
	case "array":
		node := map[string]any{"type": "array"}
		items, err := renderArrayItems(prop.ItemType, prop.ItemTypes, dc, defs)
		if err != nil {
			return nil, err
		}
		if items != nil {
			node["items"] = items
		}
		return withEnum(node, prop.Config), nil
	case "object":
		return map[string]any{"type": "object"}, nil
	case "":
		return nil, fmt.Errorf("property %q has no type", prop.Name)
	default:
		return withEnum(map[string]any{"type": prop.Type}, prop.Config), nil
	}
}

func renderArrayItems(itemType string, itemTypes []string, dc *localcatalog.DataCatalog, defs map[string]any) (map[string]any, error) {
	if itemType != "" {
		if ref, ok, err := renderCustomRef(itemType, dc, defs); err != nil {
			return nil, err
		} else if ok {
			return ref, nil
		}
		return map[string]any{"type": itemType}, nil
	}
	if len(itemTypes) > 0 {
		return map[string]any{"type": toAnySlice(itemTypes)}, nil
	}
	return nil, nil // array of any: no items constraint
}

func withEnum(node map[string]any, config map[string]any) map[string]any {
	if config == nil {
		return node
	}
	if enum, ok := config["enum"]; ok {
		node["enum"] = enum
	}
	return node
}

func toAnySlice(ss []string) []any {
	out := make([]any, len(ss))
	for i, s := range ss {
		out[i] = s
	}
	return out
}

func derefBool(b *bool, def bool) bool {
	if b == nil {
		return def
	}
	return *b
}

func errVariantUnsupported(where string) error {
	return fmt.Errorf(
		"%s uses variants, which are not yet supported by `typer generate --local`; "+
			"apply the plan and generate from remote, or remove the variant",
		where,
	)
}
