package model

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/validate"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/samber/lo"
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

	// FIXME: should we not cherry-pick the fields we need ?
	byt, err := json.Marshal(p.Property)
	if err != nil {
		return nil, fmt.Errorf("marshalling property: %w", err)
	}

	toReturn := make(map[string]any)
	if err := json.Unmarshal(byt, &toReturn); err != nil {
		return nil, fmt.Errorf("unmarshalling property: %w", err)
	}

	return toReturn, nil
}

func (p *ImportableProperty) fromUpstream(externalID string, upstream *catalog.Property, resolver resolver.ReferenceResolver) error {
	p.Property.LocalID = externalID
	p.Property.Name = upstream.Name
	p.Property.Description = upstream.Description
	p.Property.Type = upstream.Type
	p.Property.Config = upstream.Config

	// If the type not matches the valid types, it means it's a customType ID
	// reference which needs to be resolved to a custom type reference
	if isCustomType(p.Property.Type) {
		customTypeRef, err := resolver.ResolveToReference(
			state.CustomTypeResourceType,
			p.Property.Type)
		if err != nil {
			return fmt.Errorf("custom type reference resolution for resource: %s: %w", p.Property.LocalID, err)
		}

		if customTypeRef == "" {
			return fmt.Errorf("resolved custom type reference is empty")
		}

		p.Property.Type = customTypeRef
	}

	return nil
}

// isCustomType checks if the type is a custom type id reference
// by making sure to return true if the type doesn't contain any pre-defined static types
func isCustomType(typ string) bool {
	rawtypes := strings.Split(typ, ",") // typ = "number,integer,string"

	types := make([]string, 0, len(rawtypes))
	for _, t := range rawtypes {
		types = append(types, strings.TrimSpace(t))
	}

	return lo.Some(validate.ValidTypes, types)
}
