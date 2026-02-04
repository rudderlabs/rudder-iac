package model

import (
	"encoding/json"
	"fmt"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/types"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/utils"
)

type ImportableCustomType struct {
	localcatalog.CustomTypeV1
}

// ForExport loads the custom type from the upstream and resolves references to properties and other custom types.
// It then returns the custom type in a format that can be exported to a file.
func (ct *ImportableCustomType) ForExport(
	externalID string,
	upstream *catalog.CustomType,
	resolver resolver.ReferenceResolver,
) (map[string]any, error) {
	if err := ct.fromUpstream(externalID, upstream, resolver); err != nil {
		return nil, fmt.Errorf("loading custom type from upstream: %w", err)
	}

	toReturn := make(map[string]any)
	byt, err := json.Marshal(ct.CustomTypeV1)
	if err != nil {
		return nil, fmt.Errorf("marshalling custom type: %w", err)
	}

	if err := json.Unmarshal(byt, &toReturn); err != nil {
		return nil, fmt.Errorf("unmarshalling custom type: %w", err)
	}

	return toReturn, nil
}

func (ct *ImportableCustomType) fromUpstream(
	externalID string,
	upstream *catalog.CustomType,
	resolver resolver.ReferenceResolver,
) error {
	ct.CustomTypeV1.LocalID = externalID
	ct.CustomTypeV1.Name = upstream.Name
	ct.CustomTypeV1.Description = upstream.Description
	ct.CustomTypeV1.Type = upstream.Type

	ct.CustomTypeV1.Config = make(map[string]any)
	// Convert camelCase keys to snake_case when copying from upstream
	for key, value := range upstream.Config {
		snakeKey := utils.ToSnakeCase(key)
		ct.CustomTypeV1.Config[snakeKey] = value
	}

	ct.CustomTypeV1.Properties = make(
		[]localcatalog.CustomTypePropertyV1,
		0,
		len(upstream.Properties),
	)

	for _, prop := range upstream.Properties {
		propertyRef, err := resolver.ResolveToReference(
			types.PropertyResourceType,
			prop.ID,
		)
		if err != nil {
			return fmt.Errorf("resolving reference for property %s: %w", prop.ID, err)
		}

		if propertyRef == "" {
			return fmt.Errorf("resolved reference is empty for property %s", prop.ID)
		}

		ct.CustomTypeV1.Properties = append(ct.CustomTypeV1.Properties, localcatalog.CustomTypePropertyV1{
			Property: propertyRef,
			Required: prop.Required,
		})
	}

	for _, item := range upstream.ItemDefinitions {
		itemMap, ok := item.(map[string]any)
		if !ok {
			continue
		}

		itemID, ok := itemMap["id"].(string)
		if !ok || itemID == "" {
			continue
		}

		customTypeRef, err := resolver.ResolveToReference(
			types.CustomTypeResourceType,
			itemID,
		)
		if err != nil {
			return fmt.Errorf("resolving reference for itemTypes: %s: %w", itemID, err)
		}

		if customTypeRef == "" {
			return fmt.Errorf("resolved reference is empty for item_types: %s", itemID)
		}

		ct.CustomTypeV1.Config["item_types"] = []any{customTypeRef}
		break
	}

	// Process variants and resolve property references within them
	var importableVariants ImportableVariantsV1
	if err := importableVariants.fromUpstream(upstream.Variants, resolver); err != nil {
		return fmt.Errorf("processing variants: %w", err)
	}
	ct.CustomTypeV1.Variants = importableVariants.VariantsV1

	return nil
}
