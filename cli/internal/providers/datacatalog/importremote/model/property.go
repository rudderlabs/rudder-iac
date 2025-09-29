package model

import (
	"context"
	"fmt"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
)

type ImportableProperty struct {
	localcatalog.Property
}

// ForExport loads the property from the upstream and resolves references to custom types.
// It then returns the property in a format that can be exported to a file.
func (p *ImportableProperty) ForExport(externalID string, upstream *catalog.Property, resolver resolver.ReferenceResolver) (map[string]any, error) {
	p.Property.LocalID = externalID
	p.Property.Name = upstream.Name
	p.Property.Description = upstream.Description
	p.Property.Type = upstream.Type // This could point to custom-type and needs to be referenced properly
	p.Property.Config = upstream.Config

	if localcatalog.CustomTypeRegex.MatchString(p.Property.Type) {
		customTypeRef, err := resolver.ResolveToReference(
			context.Background(),
			state.CustomTypeResourceType,
			p.Property.Type)
		if err != nil {
			return nil, fmt.Errorf("custom type reference resolution for resource: %s: %w", p.Property.LocalID, err)
		}

		if customTypeRef == "" {
			return nil, fmt.Errorf("resolved custom type reference is empty")
		}

		p.Property.Type = customTypeRef
	}

	return nil, nil
}
