package providers

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/project"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/typer/plan"
)

// LocalCatalogPlanProvider builds a tracking plan from local YAML specs so that
// `typer generate --local` can produce code without fetching from (or applying
// to) the remote workspace.
//
// It converts the loaded catalog directly into the typer's plan.TrackingPlan
// contract. plan.TrackingPlan — not the remote's JSON-Schema wire format — is the
// internal contract the generators consume, so the local and remote providers are
// two adapters onto the same struct; equivalence is enforced by tests, not by
// routing local specs through the remote serialization.
type LocalCatalogPlanProvider struct {
	dc             *localcatalog.DataCatalog
	trackingPlanID string
}

func NewLocalCatalogPlanProvider(dc *localcatalog.DataCatalog, trackingPlanID string) *LocalCatalogPlanProvider {
	return &LocalCatalogPlanProvider{dc: dc, trackingPlanID: trackingPlanID}
}

// NewLocalCatalogPlanProviderForProject loads and validates the project at
// location offline (no auth or network) and returns a provider for its tracking
// plan. trackingPlanID is optional when the project has exactly one plan.
//
// Only the data catalog provider is registered, since that is all code
// generation reads. A project directory may still hold specs owned by other
// providers (data-graphs, transformations), so unknown kinds are skipped rather
// than failing the load.
func NewLocalCatalogPlanProviderForProject(location, trackingPlanID string) (*LocalCatalogPlanProvider, error) {
	dcProvider := datacatalog.New(nil)
	proj := project.New(dcProvider, project.WithIgnoreUnknownKinds())
	if err := proj.Load(location); err != nil {
		return nil, fmt.Errorf("loading and validating project: %w", err)
	}

	dc := dcProvider.GetLocalCatalog()
	id, err := resolveTrackingPlanID(dc, trackingPlanID)
	if err != nil {
		return nil, err
	}
	return NewLocalCatalogPlanProvider(dc, id), nil
}

// resolveTrackingPlanID defaults to the only plan when the project has exactly one
// and none was requested; otherwise it requires an explicit id.
func resolveTrackingPlanID(dc *localcatalog.DataCatalog, trackingPlanID string) (string, error) {
	if trackingPlanID != "" {
		return trackingPlanID, nil
	}

	switch len(dc.TrackingPlans) {
	case 0:
		return "", fmt.Errorf("no tracking plans found in the project")
	case 1:
		return dc.TrackingPlans[0].LocalID, nil
	default:
		ids := make([]string, 0, len(dc.TrackingPlans))
		for _, tp := range dc.TrackingPlans {
			ids = append(ids, tp.LocalID)
		}
		sort.Strings(ids)
		return "", fmt.Errorf("multiple tracking plans found, specify --tracking-plan-id (available: %s)", strings.Join(ids, ", "))
	}
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

	// ExpandRefs resolves every event/property reference in the rules into typed
	// TPEvent/TPEventProperty structs, leaving custom-typed properties as ref
	// strings that buildProperty resolves against the catalog.
	if err := tp.ExpandRefs(p.dc); err != nil {
		return nil, fmt.Errorf("expanding tracking plan refs: %w", err)
	}

	rules := make([]plan.EventRule, 0, len(tp.EventProps))
	for _, ev := range tp.EventProps {
		rule, err := p.buildEventRule(ev)
		if err != nil {
			return nil, fmt.Errorf("building rule for event %q: %w", ev.Name, err)
		}
		rules = append(rules, *rule)
	}

	return &plan.TrackingPlan{
		Name:     tp.Name,
		Rules:    rules,
		Metadata: plan.PlanMetadata{TrackingPlanID: tp.LocalID},
	}, nil
}

func (p *LocalCatalogPlanProvider) buildEventRule(ev *localcatalog.TPEvent) (*plan.EventRule, error) {
	evType, err := parseEventType(ev.Type)
	if err != nil {
		return nil, err
	}

	sectionStr := ev.IdentitySection
	if sectionStr == "" {
		sectionStr = "properties"
	}
	section, err := parseIdentitySection(sectionStr)
	if err != nil {
		return nil, err
	}

	// Custom types are scoped per event rule, matching the remote provider which
	// builds them from each event's $defs. The registry shares instances across
	// references within the rule and breaks reference cycles.
	reg := map[string]*plan.CustomType{}

	schema, err := p.buildObjectSchema(ev.Properties, ev.AllowUnplanned, reg)
	if err != nil {
		return nil, err
	}

	variants, err := p.buildVariants(ev.Variants, reg)
	if err != nil {
		return nil, fmt.Errorf("building variants: %w", err)
	}

	return &plan.EventRule{
		Event:    plan.Event{Name: ev.Name, Description: ev.Description, EventType: evType},
		Section:  section,
		Schema:   *schema,
		Variants: variants,
	}, nil
}

func (p *LocalCatalogPlanProvider) buildObjectSchema(props []*localcatalog.TPEventProperty, additionalProps bool, reg map[string]*plan.CustomType) (*plan.ObjectSchema, error) {
	schema := &plan.ObjectSchema{
		Properties:           make(map[string]plan.PropertySchema, len(props)),
		AdditionalProperties: additionalProps,
	}
	for _, prop := range props {
		property, nested, err := p.buildProperty(propInput{
			Name:                 prop.Name,
			Type:                 prop.Type,
			Types:                prop.Types,
			ItemType:             prop.ItemType,
			ItemTypes:            prop.ItemTypes,
			Config:               prop.Config,
			Nested:               prop.Properties,
			AdditionalProperties: prop.AdditionalProperties,
		}, reg)
		if err != nil {
			return nil, err
		}
		schema.Properties[prop.Name] = plan.PropertySchema{Property: property, Required: prop.Required, Schema: nested}
	}
	return schema, nil
}

// propInput is the shape buildProperty needs, decoupled from whether it came from
// a resolved rule property (which supplies Nested object fields) or a property
// definition used inside a custom type or variant case (which does not).
type propInput struct {
	Name      string
	Type      string
	Types     []string
	ItemType  string
	ItemTypes []string
	Config    map[string]any
	// Nested holds an object's fields when the caller has them (rule properties);
	// nil yields an empty object schema, matching the remote provider for bare
	// object properties in $defs.
	Nested               []*localcatalog.TPEventProperty
	AdditionalProperties *bool
}

func (p *LocalCatalogPlanProvider) buildProperty(in propInput, reg map[string]*plan.CustomType) (plan.Property, *plan.ObjectSchema, error) {
	property := plan.Property{Name: in.Name}

	if ct, ok, err := p.customType(in.Type, reg); err != nil {
		return property, nil, err
	} else if ok {
		property.Types = []plan.PropertyType{ct}
		return property, nil, nil
	}

	if len(in.Types) > 0 {
		types, err := toPrimitiveTypes(in.Types)
		if err != nil {
			return property, nil, err
		}
		property.Types = types
		property.Config = toConfig(in.Config)

		// Mirror the remote parser, which still resolves nested-object and
		// array-item detail for a multi-type property, keyed off the first type.
		if types[0] == plan.PrimitiveTypeObject {
			nested, err := p.buildObjectSchema(in.Nested, derefBool(in.AdditionalProperties, true), reg)
			if err != nil {
				return property, nil, err
			}
			return property, nested, nil
		}
		if types[0] == plan.PrimitiveTypeArray {
			itemTypes, err := p.buildItemTypes(in.ItemType, in.ItemTypes, reg)
			if err != nil {
				return property, nil, err
			}
			property.ItemTypes = itemTypes
		}
		return property, nil, nil
	}

	switch in.Type {
	case "object":
		property.Types = []plan.PropertyType{plan.PrimitiveTypeObject}
		nested, err := p.buildObjectSchema(in.Nested, derefBool(in.AdditionalProperties, true), reg)
		if err != nil {
			return property, nil, err
		}
		return property, nested, nil

	case "array":
		property.Types = []plan.PropertyType{plan.PrimitiveTypeArray}
		itemTypes, err := p.buildItemTypes(in.ItemType, in.ItemTypes, reg)
		if err != nil {
			return property, nil, err
		}
		property.ItemTypes = itemTypes
		property.Config = toConfig(in.Config)
		return property, nil, nil

	case "":
		return property, nil, fmt.Errorf("property %q has no type", in.Name)

	default:
		pt, err := parsePrimitiveType(in.Type)
		if err != nil {
			return property, nil, err
		}
		property.Types = []plan.PropertyType{pt}
		property.Config = toConfig(in.Config)
		return property, nil, nil
	}
}

// buildPropertyFromDef converts a property referenced by its definition (a custom
// type field or a variant case property), which carries no rule-supplied nesting.
func (p *LocalCatalogPlanProvider) buildPropertyFromDef(prop *localcatalog.PropertyV1, required bool, reg map[string]*plan.CustomType) (plan.PropertySchema, error) {
	property, nested, err := p.buildProperty(propInput{
		Name:      prop.Name,
		Type:      prop.Type,
		Types:     prop.Types,
		ItemType:  prop.ItemType,
		ItemTypes: prop.ItemTypes,
		Config:    prop.Config,
	}, reg)
	if err != nil {
		return plan.PropertySchema{}, err
	}
	return plan.PropertySchema{Property: property, Required: required, Schema: nested}, nil
}

func (p *LocalCatalogPlanProvider) buildItemTypes(itemType string, itemTypes []string, reg map[string]*plan.CustomType) ([]plan.PropertyType, error) {
	if itemType != "" {
		if ct, ok, err := p.customType(itemType, reg); err != nil {
			return nil, err
		} else if ok {
			return []plan.PropertyType{ct}, nil
		}
		pt, err := parsePrimitiveType(itemType)
		if err != nil {
			return nil, err
		}
		return []plan.PropertyType{pt}, nil
	}

	var types []plan.PropertyType
	for _, it := range itemTypes {
		pt, err := parsePrimitiveType(it)
		if err != nil {
			return nil, err
		}
		types = append(types, pt)
	}
	return types, nil // nil for array-of-any
}

// customType resolves a custom-type reference into a *plan.CustomType, caching
// instances in reg so repeated references share one instance and cycles
// terminate. ok is false when typeStr is not a custom-type reference.
func (p *LocalCatalogPlanProvider) customType(typeStr string, reg map[string]*plan.CustomType) (*plan.CustomType, bool, error) {
	m := localcatalog.CustomTypeRegex.FindStringSubmatch(typeStr)
	if len(m) != 2 {
		return nil, false, nil
	}
	ct := p.dc.CustomType(m[1])
	if ct == nil {
		return nil, false, fmt.Errorf("custom type %q not found in local specs", m[1])
	}

	if existing, ok := reg[ct.Name]; ok {
		return existing, true, nil
	}

	// Description is intentionally not set: the remote provider never populates
	// CustomType.Description (the JSON-Schema parser drops it), so setting it here
	// would make the same custom type render a different doc comment under --local.
	pct := &plan.CustomType{Name: ct.Name}
	reg[ct.Name] = pct // register before children so cyclic references terminate

	primType, err := parsePrimitiveType(ct.Type)
	if err != nil {
		return nil, false, fmt.Errorf("custom type %q: %w", ct.Name, err)
	}
	pct.Type = primType
	pct.Config = toConfig(ct.Config)

	switch ct.Type {
	case "object":
		schema := &plan.ObjectSchema{Properties: map[string]plan.PropertySchema{}, AdditionalProperties: customTypeAdditionalProperties(ct.Config)}
		for _, ctp := range ct.Properties {
			prop, err := p.resolveProperty(ctp.Property)
			if err != nil {
				return nil, false, fmt.Errorf("custom type %q: %w", ct.Name, err)
			}
			ps, err := p.buildPropertyFromDef(prop, ctp.Required, reg)
			if err != nil {
				return nil, false, err
			}
			schema.Properties[prop.Name] = ps
		}
		pct.Schema = schema
	case "array":
		itemTypes, err := p.buildItemTypes(ct.ItemType, ct.ItemTypes, reg)
		if err != nil {
			return nil, false, err
		}
		if len(itemTypes) > 0 {
			pct.ItemType = itemTypes[0]
		}
	}

	variants, err := p.buildVariants(ct.Variants, reg)
	if err != nil {
		return nil, false, fmt.Errorf("custom type %q: %w", ct.Name, err)
	}
	pct.Variants = variants

	return pct, true, nil
}

func (p *LocalCatalogPlanProvider) buildVariants(vs localcatalog.VariantsV1, reg map[string]*plan.CustomType) ([]plan.Variant, error) {
	if len(vs) == 0 {
		return nil, nil
	}

	out := make([]plan.Variant, 0, len(vs))
	for _, v := range vs {
		discriminator, err := p.resolvePropertyName(v.Discriminator)
		if err != nil {
			return nil, fmt.Errorf("resolving discriminator: %w", err)
		}

		cases := make([]plan.VariantCase, 0, len(v.Cases))
		for _, c := range v.Cases {
			schema, err := p.buildVariantCaseSchema(c.Properties, reg)
			if err != nil {
				return nil, err
			}
			cases = append(cases, plan.VariantCase{
				DisplayName: c.DisplayName,
				Match:       c.Match,
				Description: c.Description,
				Schema:      *schema,
			})
		}

		var defaultSchema *plan.ObjectSchema
		if len(v.Default.Properties) > 0 {
			defaultSchema, err = p.buildVariantCaseSchema(v.Default.Properties, reg)
			if err != nil {
				return nil, err
			}
		}

		out = append(out, plan.Variant{
			Type:          "discriminator",
			Discriminator: discriminator,
			Cases:         cases,
			DefaultSchema: defaultSchema,
		})
	}
	return out, nil
}

func (p *LocalCatalogPlanProvider) buildVariantCaseSchema(refs []localcatalog.PropertyReferenceV1, reg map[string]*plan.CustomType) (*plan.ObjectSchema, error) {
	// Variant case schemas do not allow unplanned properties, matching the remote
	// provider (which defaults variant-case additionalProperties to false).
	schema := &plan.ObjectSchema{Properties: make(map[string]plan.PropertySchema, len(refs))}
	for _, r := range refs {
		prop, err := p.resolveProperty(r.Property)
		if err != nil {
			return nil, err
		}
		ps, err := p.buildPropertyFromDef(prop, r.Required, reg)
		if err != nil {
			return nil, err
		}
		schema.Properties[prop.Name] = ps
	}
	return schema, nil
}

func (p *LocalCatalogPlanProvider) resolveProperty(ref string) (*localcatalog.PropertyV1, error) {
	m := localcatalog.PropRegex.FindStringSubmatch(ref)
	if len(m) != 2 {
		return nil, fmt.Errorf("invalid property reference %q", ref)
	}
	prop := p.dc.Property(m[1])
	if prop == nil {
		return nil, fmt.Errorf("property %q not found in local specs", m[1])
	}
	return prop, nil
}

func (p *LocalCatalogPlanProvider) resolvePropertyName(ref string) (string, error) {
	prop, err := p.resolveProperty(ref)
	if err != nil {
		return "", err
	}
	return prop.Name, nil
}

func toPrimitiveTypes(types []string) ([]plan.PropertyType, error) {
	out := make([]plan.PropertyType, 0, len(types))
	for _, t := range types {
		pt, err := parsePrimitiveType(t)
		if err != nil {
			return nil, err
		}
		out = append(out, pt)
	}
	return out, nil
}

func toConfig(config map[string]any) *plan.PropertyConfig {
	if config == nil {
		return nil
	}
	enum, ok := config["enum"]
	if !ok {
		return nil
	}
	enumSlice, ok := enum.([]any)
	if !ok {
		return nil
	}
	return &plan.PropertyConfig{Enum: enumSlice}
}

// customTypeAdditionalProperties derives an object custom type's
// additionalProperties from its config (V1 "additional_properties", V0
// "additionalProperties"), defaulting to true to match the remote parser, which
// defaults absent additionalProperties to true.
func customTypeAdditionalProperties(config map[string]any) bool {
	for _, k := range []string{"additional_properties", "additionalProperties"} {
		if v, ok := config[k]; ok {
			if b, ok := v.(bool); ok {
				return b
			}
		}
	}
	return true
}

func derefBool(b *bool, def bool) bool {
	if b == nil {
		return def
	}
	return *b
}
