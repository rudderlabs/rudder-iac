package connection

import (
	"context"

	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"
)

// MockConnectionClient is a recording RETLStore double for the connection
// handler tests. Calls keeps the cross-method order, which is what proves a
// partial create was compensated with exactly one delete and no second create;
// the per-method slices keep the arguments, and each method delegates to an
// optional func for canned responses.
//
// The embedded interface covers the source and preview surfaces the connection
// handler never touches: calling one panics loudly instead of quietly returning
// a zero value.
type MockConnectionClient struct {
	retlClient.RETLStore

	Calls []string

	CreateCalls        []retlClient.CreateRETLConnectionRequest
	UpdateCalls        []UpdateCall
	DeleteCalls        []string
	GetCalls           []string
	SetExternalIDCalls []retlClient.SetRETLConnectionExternalIDRequest

	CreateFunc func(req *retlClient.CreateRETLConnectionRequest) (*retlClient.RETLConnection, error)
	UpdateFunc func(id string, req *retlClient.UpdateRETLConnectionRequest) (*retlClient.RETLConnection, error)
	DeleteFunc func(id string) error
	GetFunc    func(id string) (*retlClient.RETLConnection, error)
	// SetExternalIDFunc is the one override handed the context: the claim is
	// where a test kills it to prove the recovery does not run on a dead one.
	SetExternalIDFunc func(ctx context.Context, req *retlClient.SetRETLConnectionExternalIDRequest) error
}

// UpdateCall records one UpdateConnection invocation: a PUT carries the
// connection id in the path rather than the body.
type UpdateCall struct {
	ID      string
	Request retlClient.UpdateRETLConnectionRequest
}

// A mismatched override is the only way a fake embedding its own interface can
// break the contract; this catches one instead of letting the fake rot.
var _ retlClient.RETLStore = (*MockConnectionClient)(nil)

func (m *MockConnectionClient) CreateConnection(_ context.Context, req *retlClient.CreateRETLConnectionRequest) (*retlClient.RETLConnection, error) {
	m.Calls = append(m.Calls, "CreateConnection")
	m.CreateCalls = append(m.CreateCalls, *req)
	if m.CreateFunc != nil {
		return m.CreateFunc(req)
	}
	return &retlClient.RETLConnection{
		ID:            "conn-remote-1",
		SourceID:      req.SourceID,
		DestinationID: req.DestinationID,
	}, nil
}

func (m *MockConnectionClient) UpdateConnection(_ context.Context, id string, req *retlClient.UpdateRETLConnectionRequest) (*retlClient.RETLConnection, error) {
	m.Calls = append(m.Calls, "UpdateConnection")
	m.UpdateCalls = append(m.UpdateCalls, UpdateCall{ID: id, Request: *req})
	if m.UpdateFunc != nil {
		return m.UpdateFunc(id, req)
	}
	// The PUT body carries no endpoints, so an echo cannot invent them; tests
	// that assert the output supply their own response.
	return &retlClient.RETLConnection{ID: id}, nil
}

// DeleteConnection and GetConnection are the two calls the partial-create
// recovery makes, so they refuse a dead context the way a real request would:
// the call is still recorded, but a cancelled or expired context never reaches
// the canned response.
func (m *MockConnectionClient) DeleteConnection(ctx context.Context, id string) error {
	m.Calls = append(m.Calls, "DeleteConnection")
	m.DeleteCalls = append(m.DeleteCalls, id)
	if err := ctx.Err(); err != nil {
		return err
	}
	if m.DeleteFunc != nil {
		return m.DeleteFunc(id)
	}
	return nil
}

func (m *MockConnectionClient) GetConnection(ctx context.Context, id string) (*retlClient.RETLConnection, error) {
	m.Calls = append(m.Calls, "GetConnection")
	m.GetCalls = append(m.GetCalls, id)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if m.GetFunc != nil {
		return m.GetFunc(id)
	}
	return &retlClient.RETLConnection{ID: id}, nil
}

func (m *MockConnectionClient) SetConnectionExternalId(ctx context.Context, req *retlClient.SetRETLConnectionExternalIDRequest) error {
	m.Calls = append(m.Calls, "SetConnectionExternalId")
	m.SetExternalIDCalls = append(m.SetExternalIDCalls, *req)
	if m.SetExternalIDFunc != nil {
		return m.SetExternalIDFunc(ctx, req)
	}
	return nil
}
