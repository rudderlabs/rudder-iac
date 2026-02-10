package model

import (
	"fmt"

	"github.com/go-viper/mapstructure/v2"
	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/types"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
)

type ImportableProperty struct {
	localcatalog.Property
}

// ForExport loads the property from the upstream and resolves references to custom types.
// It then returns the property in a format that can be exported to a file.
func (p *ImportableProperty) ForExport(
	externalID string,
	upstream *catalog.Property,
	resolver resolver.ReferenceResolver,
) (map[string]any, error) {
	if err := p.fromUpstream(externalID, upstream, resolver); err != nil {
		return nil, fmt.Errorf("loading property from upstream: %w", err)
	}

	toReturn := make(map[string]any)
	if err := mapstructure.Decode(p.Property, &toReturn); err != nil {
		return nil, fmt.Errorf("decoding property: %w", err)
	}

	return toReturn, nil
}

func (p *ImportableProperty) fromUpstream(externalID string, upstream *catalog.Property, resolver resolver.ReferenceResolver) error {
	p.Property.LocalID = externalID
	p.Property.Name = upstream.Name
	p.Property.Description = upstream.Description
	p.Property.Type = upstream.Type
	if upstream.Config != nil {
		p.Property.Config = make(map[string]interface{})
		for key, value := range upstream.Config {
			p.Property.Config[key] = value
		}
	}

	switch {
	// If the type not matches the valid types, it means it's a customType ID
	// reference which needs to be resolved to a custom type reference
	case isCustomType(upstream):
		customTypeRef, err := resolver.ResolveToReference(
			types.CustomTypeResourceType,
			upstream.DefinitionId)
		if err != nil {
			return fmt.Errorf("custom type reference resolution for resource: %s: %w", p.Property.LocalID, err)
		}

		if customTypeRef == "" {
			return fmt.Errorf("resolved custom type reference is empty")
		}
		p.Property.Type = customTypeRef
		// Hardcode the config to nil when property references a custom type
		// other we receive $ref data in it
		p.Property.Config = nil
	case upstream.Type == "array" && upstream.ItemDefinitionId != "":
		customTypeRef, err := resolver.ResolveToReference(
			types.CustomTypeResourceType,
			upstream.ItemDefinitionId)
		if err != nil {
			return fmt.Errorf("custom type reference resolution for resource: %s: %w", p.Property.LocalID, err)
		}

		if customTypeRef == "" {
			return fmt.Errorf("resolved custom type reference is empty")
		}
		p.Property.Config = map[string]interface{}{
			"itemTypes": []interface{}{customTypeRef},
		}
	}

	return nil
}

type ImportablePropertyV1 struct {
	localcatalog.PropertyV1
}

// ForExport loads the property from the upstream and resolves references to custom types.
// It then returns the property in a format that can be exported to a file.
func (p *ImportablePropertyV1) ForExport(
	externalID string,
	upstream *catalog.Property,
	resolver resolver.ReferenceResolver,
) (map[string]any, error) {
	if err := p.fromUpstream(externalID, upstream, resolver); err != nil {
		return nil, fmt.Errorf("loading property from upstream: %w", err)
	}

	toReturn := make(map[string]any)
	if err := mapstructure.Decode(p.PropertyV1, &toReturn); err != nil {
		return nil, fmt.Errorf("decoding property: %w", err)
	}

	return toReturn, nil
}

func (p *ImportablePropertyV1) fromUpstream(externalID string, upstream *catalog.Property, resolver resolver.ReferenceResolver) error {
	v0Prop := ImportableProperty{}
	if err := v0Prop.fromUpstream(externalID, upstream, resolver); err != nil {
		return fmt.Errorf("loading property from upstream: %w", err)
	}

	err := p.FromV0(v0Prop.Property)
	if err != nil {
		return fmt.Errorf("converting property to v1: %w", err)
	}
	return nil
}

func isCustomType(property *catalog.Property) bool {
	return property.DefinitionId != ""
}
