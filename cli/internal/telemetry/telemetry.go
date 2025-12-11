package telemetry

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rudderlabs/analytics-go/v4"
)

type telemetry struct {
	version     string
	anonymousID string
	client      analytics.Client
	cb          *telemetryCallback
	transport   http.RoundTripper
	timeout     time.Duration
}

type option func(*telemetry)

// withClient sets a custom analytics client, in case we need to mock the analytics library in tests
func withClient(client analytics.Client) option {
	return func(t *telemetry) {
		t.client = client
	}
}

// withTransport sets a custom HTTP transport for the analytics client, to test blocking, timeouts, etc.
// if withClient is also provided, this option is ignored
func withTransport(transport http.RoundTripper) option {
	return func(t *telemetry) {
		t.transport = transport
	}
}

func WithTimeout(timeout time.Duration) option {
	return func(t *telemetry) {
		t.timeout = timeout
	}
}

func newTelemetry(version string, anonymousID, writeKey, dataplaneURL string, opts ...option) (*telemetry, error) {
	// Create the callback handler
	cb := newTelemetryCallback()

	tel := &telemetry{
		version:     version,
		anonymousID: anonymousID,
		cb:          cb,
		timeout:     1 * time.Second,
	}

	for _, opt := range opts {
		opt(tel)
	}

	if tel.client == nil {
		ac := analytics.Config{
			BatchSize:    1,
			DataPlaneUrl: dataplaneURL,
			Logger:       NewTelemetryLogger(),
			Callback:     cb,
		}

		if tel.transport != nil {
			ac.Transport = tel.transport
		}

		c, err := analytics.NewWithConfig(writeKey, ac)
		if err != nil {
			// Log error but don't fail startup
			// Telemetry is optional and shouldn't block the CLI
			return nil, fmt.Errorf("failed to create telemetry analytics client: %w", err)
		}

		tel.client = c
	}

	return tel, nil
}

// telemetryCallback implements analytics.Callback to enable blocking track calls
type telemetryCallback struct {
	mu      sync.RWMutex
	pending map[string]chan error
}

func newTelemetryCallback() *telemetryCallback {
	return &telemetryCallback{
		pending: make(map[string]chan error),
	}
}

// Success is called when a message is successfully sent
func (tc *telemetryCallback) Success(msg analytics.Message) {
	tc.complete(msg, nil)
}

// Failure is called when a message fails to send
func (tc *telemetryCallback) Failure(msg analytics.Message, err error) {
	tc.complete(msg, err)
}

func (tc *telemetryCallback) complete(msg analytics.Message, err error) {
	// Extract MessageId from the message
	var messageID string
	switch m := msg.(type) {
	case analytics.Track:
		messageID = m.MessageId
	default:
		return
	}

	tc.mu.RLock()
	ch, exists := tc.pending[messageID]
	tc.mu.RUnlock()

	if exists {
		ch <- err
		close(ch)

		tc.mu.Lock()
		delete(tc.pending, messageID)
		tc.mu.Unlock()
	}
}

func (tc *telemetryCallback) register(messageID string) chan error {
	ch := make(chan error, 1)
	tc.mu.Lock()
	tc.pending[messageID] = ch
	tc.mu.Unlock()
	return ch
}

func (t *telemetry) Track(event string, properties analytics.Properties) error {
	props := analytics.NewProperties()

	for k, v := range props {
		props.Set(k, v)
	}

	// Generate a unique MessageId for this track call
	messageID := uuid.New().String()

	// Register a channel to wait for the callback
	resultCh := t.cb.register(messageID)

	// Enqueue the track event
	err := t.client.Enqueue(analytics.Track{
		MessageId:   messageID,
		Event:       event,
		Properties:  properties,
		AnonymousId: t.anonymousID,
		Context: &analytics.Context{
			App: analytics.AppInfo{
				Name:    "rudder-cli",
				Version: t.version,
			},
		},
	})

	if err != nil {
		// If enqueue fails immediately, clean up the pending channel
		t.cb.mu.Lock()
		delete(t.cb.pending, messageID)
		t.cb.mu.Unlock()
		close(resultCh)
		return fmt.Errorf("failed to enqueue track event: %w", err)
	}

	// Block until the message is sent (or fails), with a timeout
	select {
	case sendErr := <-resultCh:
		if sendErr != nil {
			return fmt.Errorf("failed to send track event: %w", sendErr)
		}
		return nil
	case <-time.After(t.timeout):
		return fmt.Errorf("timeout waiting for track event to be sent")
	}
}
