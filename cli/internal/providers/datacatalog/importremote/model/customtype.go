package model

import (
	"encoding/json"
	"fmt"
	"maps"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/state"
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

	// Deep copy the config map
	ct.CustomType.Config = make(map[string]any)
	maps.Copy(
		ct.CustomType.Config,
		upstream.Config,
	)

	// Resolve property references in the Properties field
	ct.CustomType.Properties = make(
		[]localcatalog.CustomTypeProperty,
		0,
		len(upstream.Properties),
	)

	for _, prop := range upstream.Properties {
		propertyRef, err := resolver.ResolveToReference(
			state.PropertyResourceType,
			prop.ID,
		)
		if err != nil {
			return fmt.Errorf("property reference resolution for custom type: %s, property: %s: %w",
				ct.CustomType.LocalID, prop.ID, err)
		}

		if propertyRef == "" {
			return fmt.Errorf("resolved property reference is empty for property: %s", prop.ID)
		}

		ct.CustomType.Properties = append(ct.CustomType.Properties, localcatalog.CustomTypeProperty{
			Ref:      propertyRef,
			Required: prop.Required,
		})
	}

	// Resolve custom type references in Config["itemTypes"]
	// ItemDefinitions contain the actual custom type data with IDs
	if len(upstream.ItemDefinitions) > 0 {
		// We only support one item definition currently
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
				state.CustomTypeResourceType,
				itemID,
			)
			if err != nil {
				return fmt.Errorf("custom type reference resolution for itemTypes in custom type: %s, item: %s: %w",
					ct.CustomType.LocalID, itemID, err)
			}

			if customTypeRef == "" {
				return fmt.Errorf("resolved custom type reference is empty for itemTypes: %s", itemID)
			}

			ct.CustomType.Config["itemTypes"] = []any{customTypeRef}
			break
		}
	}

	// Process variants and resolve property references within them
	ct.CustomType.Variants = make([]localcatalog.Variant, 0, len(upstream.Variants))
	for _, remoteVariant := range upstream.Variants {
		// Create a local catalog variant
		localVariant := localcatalog.Variant{
			Type:          remoteVariant.Type,
			Discriminator: remoteVariant.Discriminator,
			Cases:         make([]localcatalog.VariantCase, 0, len(remoteVariant.Cases)),
			Default:       make([]localcatalog.PropertyReference, 0, len(remoteVariant.Default)),
		}

		// Resolve discriminator property reference
		discriminatorRef, err := resolver.ResolveToReference(
			state.PropertyResourceType,
			remoteVariant.Discriminator,
		)
		if err != nil {
			return fmt.Errorf("discriminator property reference resolution for variant in custom type: %s, property: %s: %w",
				ct.CustomType.LocalID, remoteVariant.Discriminator, err)
		}
		if discriminatorRef == "" {
			return fmt.Errorf("resolved discriminator property reference is empty for property: %s", remoteVariant.Discriminator)
		}
		localVariant.Discriminator = discriminatorRef

		// Process each case in the variant
		for _, remoteCase := range remoteVariant.Cases {
			localCase := localcatalog.VariantCase{
				DisplayName: remoteCase.DisplayName,
				Match:       remoteCase.Match,
				Description: remoteCase.Description,
				Properties:  make([]localcatalog.PropertyReference, 0, len(remoteCase.Properties)),
			}

			// Resolve property references in the case
			for _, remoteProp := range remoteCase.Properties {
				propRef, err := resolver.ResolveToReference(
					state.PropertyResourceType,
					remoteProp.ID,
				)
				if err != nil {
					return fmt.Errorf("property reference resolution for variant case in custom type: %s, property: %s: %w",
						ct.CustomType.LocalID, remoteProp.ID, err)
				}
				if propRef == "" {
					return fmt.Errorf("resolved property reference is empty in variant case for property: %s", remoteProp.ID)
				}

				localCase.Properties = append(localCase.Properties, localcatalog.PropertyReference{
					Ref:      propRef,
					Required: remoteProp.Required,
				})
			}

			localVariant.Cases = append(localVariant.Cases, localCase)
		}

		// Process default properties in the variant
		for _, remoteProp := range remoteVariant.Default {
			propRef, err := resolver.ResolveToReference(
				state.PropertyResourceType,
				remoteProp.ID,
			)
			if err != nil {
				return fmt.Errorf("property reference resolution for variant default in custom type: %s, property: %s: %w",
					ct.CustomType.LocalID, remoteProp.ID, err)
			}
			if propRef == "" {
				return fmt.Errorf("resolved property reference is empty in variant default for property: %s", remoteProp.ID)
			}

			localVariant.Default = append(localVariant.Default, localcatalog.PropertyReference{
				Ref:      propRef,
				Required: remoteProp.Required,
			})
		}

		ct.CustomType.Variants = append(ct.CustomType.Variants, localVariant)
	}

	return nil
}
