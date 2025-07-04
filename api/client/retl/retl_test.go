package retl_test

import (
	"fmt"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/api/client/retl"
	"github.com/stretchr/testify/assert"
)

func TestIsRETLNotFoundError(t *testing.T) {
	apiErr := &client.APIError{
		HTTPStatusCode: 404,
		Message:        "not found",
	}

	assert.True(t, retl.IsRETLNotFoundError(apiErr))

	apiErr.HTTPStatusCode = 400
	apiErr.Message = "resource not found"
	assert.True(t, retl.IsRETLNotFoundError(apiErr))

	apiErr.Message = "some other error"
	assert.False(t, retl.IsRETLNotFoundError(apiErr))

	apiErr.HTTPStatusCode = 500
	assert.False(t, retl.IsRETLNotFoundError(apiErr))

	assert.False(t, retl.IsRETLNotFoundError(fmt.Errorf("not an api error")))
}

func TestIsRETLAlreadyExistsError(t *testing.T) {
	apiErr := &client.APIError{
		HTTPStatusCode: 400,
		Message:        "already exists",
	}

	assert.True(t, retl.IsRETLAlreadyExistsError(apiErr))

	apiErr.Message = "some other error"
	assert.False(t, retl.IsRETLAlreadyExistsError(apiErr))

	apiErr.HTTPStatusCode = 500
	assert.False(t, retl.IsRETLAlreadyExistsError(apiErr))

	assert.False(t, retl.IsRETLAlreadyExistsError(fmt.Errorf("not an api error")))
}
