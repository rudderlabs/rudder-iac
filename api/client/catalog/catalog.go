package catalog

import (
	"errors"
	"strings"

	"github.com/rudderlabs/rudder-iac/api/client"
)

type DataCatalog interface {
	EventStore
	PropertyStore
	TrackingPlanStore
	StateStore
	CustomTypeStore
	CategoryStore
}

type RudderDataCatalog struct {
	client *client.Client
}

func NewRudderDataCatalog(client *client.Client) DataCatalog {
	return &RudderDataCatalog{
		client: client,
	}
}

func IsCatalogNotFoundError(err error) bool {
	var apiErr *client.APIError

	if ok := errors.As(err, &apiErr); !ok {
		return false
	}
	return apiErr.HTTPStatusCode == 400 && strings.Contains(apiErr.Message, "not found")
}

func IsCatalogAlreadyExistsError(err error) bool {
	var apiErr *client.APIError

	if ok := errors.As(err, &apiErr); !ok {
		return false
	}
	return apiErr.HTTPStatusCode == 400 && strings.Contains(apiErr.Message, "already exists")
}
