package model

import (
	"fmt"

	"github.com/go-viper/mapstructure/v2"
	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
)

type ImportableCategory struct {
	localcatalog.CategoryV1
}

// ForExport loads the category from the upstream and returns it in a format
// that can be exported to a file.
func (c *ImportableCategory) ForExport(
	externalID string,
	upstream *catalog.Category,
	resolver resolver.ReferenceResolver,
) (map[string]any, error) {
	if err := c.fromUpstream(externalID, upstream); err != nil {
		return nil, fmt.Errorf("loading category from upstream: %w", err)
	}

	toReturn := make(map[string]any)
	if err := mapstructure.Decode(c.CategoryV1, &toReturn); err != nil {
		return nil, fmt.Errorf("decoding category: %w", err)
	}

	return toReturn, nil
}

func (c *ImportableCategory) fromUpstream(externalID string, upstream *catalog.Category) error {
	c.CategoryV1.LocalID = externalID
	c.CategoryV1.Name = upstream.Name

	return nil
}
