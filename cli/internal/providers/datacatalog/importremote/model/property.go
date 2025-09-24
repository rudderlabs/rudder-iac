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

func (p *ImportableProperty) FromUpstream(externalID string, upstream *catalog.Property) error {
	p.Property.LocalID = externalID
	p.Property.Name = upstream.Name
	p.Property.Description = upstream.Description
	p.Property.Type = upstream.Type // This could point to custom-type and needs to be referenced properly
	p.Property.Config = upstream.Config

	return nil
}

func (p *ImportableProperty) Flatten(inputResolver resolver.ReferenceResolver) (map[string]any, error) {
	flattened := map[string]any{
		"id":          p.Property.LocalID,
		"name":        p.Property.Name,
		"description": p.Property.Description,
		"propConfig":  p.Property.Config,
	}

	// property -> type -> def_123abc
	if !localcatalog.CustomTypeRegex.MatchString(p.Property.Type) {
		flattened["type"] = p.Property.Type
	} else {
		customTypeRef, err := inputResolver.ResolveToReference(
			context.Background(),
			state.CustomTypeResourceType,
			p.Property.Type)
		if err != nil {
			return nil, fmt.Errorf("resolving custom type reference: %w", err)
		}
		flattened["type"] = customTypeRef // #/custom-types/metadata-name/
	}

	return flattened, nil
}
