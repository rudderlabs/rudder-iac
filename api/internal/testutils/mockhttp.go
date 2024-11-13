package testutils

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type Call struct {
	// Validate is an optional function that, if set, will validate an incoming request
	Validate       func(req *http.Request) bool
	ResponseStatus int
	ResponseBody   string
	ResponseError  error
}

type mockHTTPClient struct {
	t         *testing.T
	callIndex int
	calls     []Call
}

func NewMockHTTPClient(t *testing.T, calls ...Call) *mockHTTPClient {
	c := &mockHTTPClient{
		t:     t,
		calls: calls,
	}

	return c
}

func (c *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	if c.callIndex >= len(c.calls) {
		err := fmt.Errorf("received unexpected request: %v", req)
		c.t.Error(err)
		return nil, err
	}

	call := c.calls[c.callIndex]
	c.callIndex += 1

	if call.Validate != nil {
		assert.True(c.t, call.Validate(req))
	}

	return &http.Response{
		StatusCode: call.ResponseStatus,
		Body:       io.NopCloser(strings.NewReader(call.ResponseBody)),
	}, call.ResponseError
}

func (c *mockHTTPClient) AssertNumberOfCalls() {
	if c.callIndex < len(c.calls) {
		c.t.Errorf("missing calls: expected %d, received %d", len(c.calls), c.callIndex)
	}
}

// ValidateRequest is a utility
func ValidateRequest(t *testing.T, req *http.Request, method string, url string, body string) bool {
	if !assert.Equal(t, method, req.Method) {
		return false
	}

	if body != "" {
		bodyBytes, err := ioutil.ReadAll(req.Body)
		require.NoError(t, err)
		if !assert.JSONEq(t, body, string(bodyBytes)) {
			return false
		}
	}

	return true
}
