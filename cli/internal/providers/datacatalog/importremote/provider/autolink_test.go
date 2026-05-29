package provider

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/types"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/stretchr/testify/assert"
)

func TestFindLocalCategoryMatch(t *testing.T) {
	graph := resources.NewGraph()
	graph.AddResource(resources.NewResource(
		"checkout", types.CategoryResourceType,
		resources.ResourceData{"name": "Checkout"},
		nil,
	))
	graph.AddResource(resources.NewResource(
		"user-actions", types.CategoryResourceType,
		resources.ResourceData{"name": "User Actions"},
		nil,
	))

	tests := []struct {
		name       string
		remoteName string
		wantID     string
		wantFound  bool
	}{
		{
			name:       "exact match",
			remoteName: "Checkout",
			wantID:     "checkout",
			wantFound:  true,
		},
		{
			name:       "no match",
			remoteName: "Payments",
			wantID:     "",
			wantFound:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, found := findLocalCategoryMatch(graph, tt.remoteName)
			assert.Equal(t, tt.wantFound, found)
			assert.Equal(t, tt.wantID, id)
		})
	}
}

func TestFindLocalEventMatch(t *testing.T) {
	graph := resources.NewGraph()
	graph.AddResource(resources.NewResource(
		"page-viewed", types.EventResourceType,
		resources.ResourceData{"name": "Page Viewed", "eventType": "track"},
		nil,
	))
	graph.AddResource(resources.NewResource(
		"identify", types.EventResourceType,
		resources.ResourceData{"name": "", "eventType": "identify"},
		nil,
	))

	tests := []struct {
		name            string
		remoteName      string
		remoteEventType string
		wantID          string
		wantFound       bool
	}{
		{
			name:            "track event match",
			remoteName:      "Page Viewed",
			remoteEventType: "track",
			wantID:          "page-viewed",
			wantFound:       true,
		},
		{
			name:            "non-track event match on eventType",
			remoteName:      "",
			remoteEventType: "identify",
			wantID:          "identify",
			wantFound:       true,
		},
		{
			name:            "no match",
			remoteName:      "Checkout",
			remoteEventType: "track",
			wantID:          "",
			wantFound:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, found := findLocalEventMatch(graph, tt.remoteName, tt.remoteEventType)
			assert.Equal(t, tt.wantFound, found)
			assert.Equal(t, tt.wantID, id)
		})
	}
}

func TestFindLocalPropertyMatch(t *testing.T) {
	graph := resources.NewGraph()
	graph.AddResource(resources.NewResource(
		"email", types.PropertyResourceType,
		resources.ResourceData{"name": "Email", "type": "string", "config": map[string]interface{}{}},
		nil,
	))
	graph.AddResource(resources.NewResource(
		"tags", types.PropertyResourceType,
		resources.ResourceData{
			"name":   "Tags",
			"type":   "array",
			"config": map[string]interface{}{"item_types": []interface{}{"string"}},
		},
		nil,
	))

	tests := []struct {
		name       string
		remoteName string
		remoteType string
		remoteConf map[string]interface{}
		wantID     string
		wantFound  bool
	}{
		{
			name:       "simple type match",
			remoteName: "Email",
			remoteType: "string",
			remoteConf: map[string]interface{}{},
			wantID:     "email",
			wantFound:  true,
		},
		{
			name:       "array with itemTypes match",
			remoteName: "Tags",
			remoteType: "array",
			remoteConf: map[string]interface{}{"item_types": []interface{}{"string"}},
			wantID:     "tags",
			wantFound:  true,
		},
		{
			name:       "same name different type no match",
			remoteName: "Email",
			remoteType: "number",
			remoteConf: map[string]interface{}{},
			wantID:     "",
			wantFound:  false,
		},
		{
			name:       "no match",
			remoteName: "Phone",
			remoteType: "string",
			remoteConf: map[string]interface{}{},
			wantID:     "",
			wantFound:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, found := findLocalPropertyMatch(graph, tt.remoteName, tt.remoteType, tt.remoteConf)
			assert.Equal(t, tt.wantFound, found)
			assert.Equal(t, tt.wantID, id)
		})
	}
}

func TestFindLocalTrackingPlanMatch(t *testing.T) {
	graph := resources.NewGraph()
	graph.AddResource(resources.NewResource(
		"api-tracking", types.TrackingPlanResourceType,
		resources.ResourceData{"name": "API Tracking"},
		nil,
	))

	tests := []struct {
		name       string
		remoteName string
		wantID     string
		wantFound  bool
	}{
		{
			name:       "exact match",
			remoteName: "API Tracking",
			wantID:     "api-tracking",
			wantFound:  true,
		},
		{
			name:       "no match",
			remoteName: "Mobile Tracking",
			wantID:     "",
			wantFound:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, found := findLocalTrackingPlanMatch(graph, tt.remoteName)
			assert.Equal(t, tt.wantFound, found)
			assert.Equal(t, tt.wantID, id)
		})
	}
}
