package telemetry

import (
	"fmt"
	"testing"

	"github.com/rudderlabs/analytics-go/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTrackCommand(t *testing.T) {
	require.Nil(t, tel, "telemetry should be nil initially")
	Initialise("2.3.4")
	require.NotNil(t, tel, "telemetry should be initialized")

	// Use a mock client to capture the tracked events
	mockClient := &mockClient{}
	tel.client = mockClient

	TrackCommand("some-command", fmt.Errorf("some-error"), KV{"K1", "V1"}, KV{"K2", 42})

	require.Len(t, mockClient.enqueuedMessages, 1, "expected one tracked message")

	message := mockClient.enqueuedMessages[0]
	assert.Equal(t, message.(analytics.Track).Event, CommandExecutedEvent, "event name mismatch")
	props := message.(analytics.Track).Properties
	assert.Equal(t, "some-command", props["command"], "command property mismatch")
	assert.Equal(t, true, props["errored"], "errored property mismatch")
	assert.Equal(t, "V1", props["K1"], "extra property K1 mismatch")
	assert.Equal(t, 42, props["K2"], "extra property K2 mismatch")
}

type mockClient struct {
	enqueuedMessages []analytics.Message
	mockError        error
}

func (mc *mockClient) Close() error {
	return nil
}

func (mc *mockClient) Enqueue(m analytics.Message) error {
	mc.enqueuedMessages = append(mc.enqueuedMessages, m)
	return mc.mockError
}
