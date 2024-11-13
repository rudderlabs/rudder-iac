//go:build integrationtest
// +build integrationtest

package client_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/stretchr/testify/require"
)

func newClient() (*client.Client, error) {
	baseUrl := os.Getenv("RUDDERSTACK_API_HOST")
	if baseUrl == "" {
		baseUrl = "http://localhost:5555/v2"
	}
	return client.New(os.Getenv("RUDDERSTACK_API_ACCESS_TOKEN"), client.WithBaseURL(baseUrl))
}

func TestClientIntegration(t *testing.T) {
	c, err := newClient()
	require.NoError(t, err)

	// get list of sources, destinations and connections before creating any resources
	preSources, err := c.Sources.List(context.Background())
	require.NoError(t, err)

	preDestinations, err := c.Destinations.List(context.Background())
	require.NoError(t, err)

	preConnections, err := c.Connections.List(context.Background())
	require.NoError(t, err)

	// create a new source, destination and connection
	source, err := c.Sources.Create(context.Background(), &client.Source{
		Type: "HTTP",
		Name: "source-name",
	})
	require.NoError(t, err)
	require.NotEmpty(t, source.ID)

	destination, err := c.Destinations.Create(context.Background(), &client.Destination{
		Type:   "WEBHOOK",
		Name:   "destinaton-name",
		Config: json.RawMessage("{}"),
	})
	require.NoError(t, err)
	require.NotEmpty(t, destination.ID)

	connection, err := c.Connections.Create(context.Background(), &client.Connection{
		SourceID:      source.ID,
		DestinationID: destination.ID,
	})
	require.NoError(t, err)
	require.NotEmpty(t, connection.ID)

	// check current list of sources, destinations and connections
	sources, err := c.Sources.List(context.Background())
	require.NoError(t, err)
	require.Equal(t, len(sources.Sources), len(preSources.Sources)+1)

	destinations, err := c.Destinations.List(context.Background())
	require.NoError(t, err)
	require.Equal(t, len(destinations.Destinations), len(preDestinations.Destinations)+1)

	connections, err := c.Connections.List(context.Background())
	require.NoError(t, err)
	require.Equal(t, len(connections.Connections), len(preConnections.Connections)+1)

	// check retrieval of individual resources
	postSource, err := c.Sources.Get(context.Background(), source.ID)
	require.NoError(t, err)
	require.Equal(t, source, postSource)

	postDestination, err := c.Destinations.Get(context.Background(), destination.ID)
	require.NoError(t, err)
	require.Equal(t, destination, postDestination)

	postConnection, err := c.Connections.Get(context.Background(), connection.ID)
	require.NoError(t, err)
	require.Equal(t, connection, postConnection)
	fmt.Println(postConnection)
}
