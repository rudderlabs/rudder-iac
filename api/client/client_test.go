package client_test

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/stretchr/testify/assert"
)

func TestClientEmptyAccessToken(t *testing.T) {
	_, err := client.New("")
	assert.Equal(t, client.ErrEmptyAccessToken, err, "error should be ErrEmptyAccessToken")
}

func TestClientURL(t *testing.T) {
	c, err := client.New("some-access-token")
	assert.NoError(t, err)
	assert.Equal(t, "https://api.rudderstack.com/v2", c.URL(""))
	assert.Equal(t, "https://api.rudderstack.com/v2/path", c.URL("path"))
	assert.Equal(t, "https://api.rudderstack.com/v2/path", c.URL("/path"))
	assert.Equal(t, "https://api.rudderstack.com/v2/path/more", c.URL("/path/more"))
}
