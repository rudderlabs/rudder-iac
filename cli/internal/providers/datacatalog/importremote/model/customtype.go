package model

import (
	"encoding/json"
	"fmt"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/types"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
)

type ImportableCustomType struct {
	localcatalog.CustomType
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
	byt, err := json.Marshal(ct.CustomType)
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
	ct.CustomType.LocalID = externalID
	ct.CustomType.Name = upstream.Name
	ct.CustomType.Description = upstream.Description
	ct.CustomType.Type = upstream.Type

	if upstream.Config != nil {
		ct.CustomType.Config = make(map[string]any)
		for key, value := range upstream.Config {
			ct.CustomType.Config[key] = value
		}
	}

	ct.CustomType.Properties = make(
		[]localcatalog.CustomTypeProperty,
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

		ct.CustomType.Properties = append(ct.CustomType.Properties, localcatalog.CustomTypeProperty{
			Ref:      propertyRef,
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

		if ct.CustomType.Config == nil {
			ct.CustomType.Config = make(map[string]any)
		}
		ct.CustomType.Config["itemTypes"] = []any{customTypeRef}
		break
	}

	// Process variants and resolve property references within them
	var importableVariants ImportableVariants
	if err := importableVariants.fromUpstream(upstream.Variants, resolver); err != nil {
		return fmt.Errorf("processing variants: %w", err)
	}
	ct.CustomType.Variants = importableVariants.Variants

	return nil
}

type ImportableCustomTypeV1 struct {
	localcatalog.CustomTypeV1
}

// ForExport loads the custom type from the upstream and resolves references to properties and other custom types.
// It then returns the custom type in a format that can be exported to a file.
func (ct *ImportableCustomTypeV1) ForExport(
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

func (ct *ImportableCustomTypeV1) fromUpstream(
	externalID string,
	upstream *catalog.CustomType,
	resolver resolver.ReferenceResolver,
) error {
	v0CustomType := ImportableCustomType{}
	if err := v0CustomType.fromUpstream(externalID, upstream, resolver); err != nil {
		return fmt.Errorf("loading custom type from upstream: %w", err)
	}

	err := ct.CustomTypeV1.FromV0(v0CustomType.CustomType)
	if err != nil {
		return fmt.Errorf("converting custom type to v1: %w", err)
	}

	return nil
}
