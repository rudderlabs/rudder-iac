package model

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
)

// ImportableVariants is an embeddable type for handling variant conversion from upstream.
type ImportableVariants struct {
	Variants []localcatalog.Variant
}

// fromUpstream processes remote variants and resolves all property references within them.
// It handles discriminators, cases with their properties, and default properties.
func (iv *ImportableVariants) fromUpstream(
	remoteVariants catalog.Variants,
	resolver resolver.ReferenceResolver,
) error {
	if len(remoteVariants) == 0 {
		iv.Variants = nil
		return nil
	}

	iv.Variants = make([]localcatalog.Variant, 0, len(remoteVariants))

	for i := range remoteVariants {
		remoteVariant := &remoteVariants[i]
		localVariant := localcatalog.Variant{
			Type:          remoteVariant.Type,
			Discriminator: remoteVariant.Discriminator,
			Cases:         make([]localcatalog.VariantCase, 0, len(remoteVariant.Cases)),
			Default:       make([]localcatalog.PropertyReference, 0, len(remoteVariant.Default)),
		}

		discriminatorRef, err := resolver.ResolveToReference(
			state.PropertyResourceType,
			remoteVariant.Discriminator,
		)
		if err != nil {
			return fmt.Errorf("resolving reference for discriminator: %s: %w",
				remoteVariant.Discriminator, err)
		}
		if discriminatorRef == "" {
			return fmt.Errorf("resolved reference is empty for discriminator: %s", remoteVariant.Discriminator)
		}
		localVariant.Discriminator = discriminatorRef

		for _, remoteCase := range remoteVariant.Cases {
			localCase := localcatalog.VariantCase{
				DisplayName: remoteCase.DisplayName,
				Match:       remoteCase.Match,
				Description: remoteCase.Description,
				Properties:  make([]localcatalog.PropertyReference, 0, len(remoteCase.Properties)),
			}

			for _, remoteProp := range remoteCase.Properties {
				propRef, err := resolver.ResolveToReference(
					state.PropertyResourceType,
					remoteProp.ID,
				)
				if err != nil {
					return fmt.Errorf("resolving reference for property %s in variant case %s: %w", remoteProp.ID, remoteCase.DisplayName, err)
				}
				if propRef == "" {
					return fmt.Errorf("resolved reference is empty for property %s in variant case %s", remoteProp.ID, remoteCase.DisplayName)
				}

				localCase.Properties = append(localCase.Properties, localcatalog.PropertyReference{
					Ref:      propRef,
					Required: remoteProp.Required,
				})
			}

			localVariant.Cases = append(localVariant.Cases, localCase)
		}

		for _, remoteProp := range remoteVariant.Default {
			propRef, err := resolver.ResolveToReference(
				state.PropertyResourceType,
				remoteProp.ID,
			)
			if err != nil {
				return fmt.Errorf("resolving reference for property %s in variant default: %w",
					remoteProp.ID, err)
			}
			if propRef == "" {
				return fmt.Errorf("resolved reference is empty for property %s in variant default", remoteProp.ID)
			}

			localVariant.Default = append(localVariant.Default, localcatalog.PropertyReference{
				Ref:      propRef,
				Required: remoteProp.Required,
			})
		}

		iv.Variants = append(iv.Variants, localVariant)
	}

	return nil
}
