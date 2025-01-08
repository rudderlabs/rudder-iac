package provider

import (
	"errors"
	"strings"

	"github.com/rudderlabs/rudder-iac/api/client"
)

func IsNotFound(err error) bool {
	var apiErr *client.APIError

	if ok := errors.As(err, &apiErr); !ok {
		return false
	}
	return apiErr.HTTPStatusCode == 400 && strings.Contains(apiErr.Message, "not found")
}
