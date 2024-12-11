package provider

import (
	"context"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
)

type PropertyProvider struct {
	client client.DataCatalog
}

func NewPropertyProvider(dc client.DataCatalog) syncer.Provider {
	return &PropertyProvider{
		client: dc,
	}
}

func (p *PropertyProvider) Create(ctx context.Context, ID string, resourceType string, data resources.ResourceData) *resources.ResourceData {

	payload := client.PropertyCreate{
		Name:        data["display_name"].(string),
		Description: data["description"].(string),
		Type:        data["type"].(string),
		Config:      data["propConfig"].(map[string]interface{}),
	}

	property, _ := p.client.CreateProperty(ctx, payload)
	data["id"] = property.ID
	data["workspaceID"] = property.WorkspaceId

	return &data
}

func (p *PropertyProvider) Update(_ context.Context, ID string, resourceType string, data resources.ResourceData) *resources.ResourceData {
	return nil
}

func (p *PropertyProvider) Delete(_ context.Context, ID string, resourceType string, data resources.ResourceData) *resources.ResourceData {
	return nil
}
