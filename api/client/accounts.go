package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"
)

type Account struct {
	ID          string `json:"id,omitempty"`
	WorkspaceID string `json:"workspaceId"`
	Name        string `json:"name"`
	Definition  struct {
		Name     string `json:"name"`
		Type     string `json:"type"`
		Category string `json:"category"`
	} `json:"definition"`
	Options    json.RawMessage `json:"options"`
	CreatedAt  *time.Time      `json:"createdAt,omitempty"`
	UpdatedAt  *time.Time      `json:"updatedAt,omitempty"`
	ExternalID string          `json:"externalId,omitempty"`
}

type CreateAccountRequest struct {
	AccountDefinitionName string          `json:"accountDefinitionName"`
	Name                  string          `json:"name"`
	Options               json.RawMessage `json:"options"`
	Secret                json.RawMessage `json:"secret"`
}

// All fields are required — PUT is a full-state replace, so a missing field means
// "set to empty", not "leave unchanged". accountDefinitionName is immutable after
// creation and intentionally absent.
type UpdateAccountRequest struct {
	Name    string          `json:"name"`
	Options json.RawMessage `json:"options"`
	Secret  json.RawMessage `json:"secret"`
}

type accounts struct {
	*service
}

type AccountsPage struct {
	APIPage
	Accounts []Account `json:"data"`
}

func (s *accounts) Next(ctx context.Context, paging Paging) (*AccountsPage, error) {
	page := &AccountsPage{}
	ok, err := s.service.next(ctx, paging, page)
	if !ok {
		page = nil
	}
	return page, err
}

type ListAccountsOptions struct {
	HasExternalID *bool
}

type ListAccountsOption func(*ListAccountsOptions)

// WithHasExternalID filters the account list to managed (true) or
// unmanaged/importable (false) accounts.
func WithHasExternalID(v bool) ListAccountsOption {
	return func(o *ListAccountsOptions) { o.HasExternalID = &v }
}

func (s *accounts) List(ctx context.Context, opts ...ListAccountsOption) (*AccountsPage, error) {
	options := &ListAccountsOptions{}
	for _, opt := range opts {
		opt(options)
	}

	path := s.basePath
	query := url.Values{}
	if options.HasExternalID != nil {
		query.Add("hasExternalId", strconv.FormatBool(*options.HasExternalID))
	}
	if encoded := query.Encode(); encoded != "" {
		path = fmt.Sprintf("%s?%s", path, encoded)
	}

	page := &AccountsPage{}
	if _, err := s.next(ctx, Paging{Next: path}, page); err != nil {
		return nil, err
	}

	return page, nil
}

func (s *accounts) ListAll(ctx context.Context) ([]Account, error) {
	var allAccounts []Account

	page, err := s.List(ctx)
	if err != nil {
		return nil, err
	}
	if page == nil {
		return allAccounts, nil
	}

	allAccounts = append(allAccounts, page.Accounts...)

	for {
		if page.Paging.Next == "" {
			break
		}
		page, err = s.Next(ctx, page.Paging)
		if err != nil {
			return nil, err
		}
		if page == nil {
			break
		}
		allAccounts = append(allAccounts, page.Accounts...)
	}

	return allAccounts, nil
}

func (s *accounts) Get(ctx context.Context, id string) (*Account, error) {
	response := &Account{}
	if err := s.get(ctx, id, response); err != nil {
		return nil, err
	}

	return response, nil
}

func (s *accounts) Create(ctx context.Context, account *CreateAccountRequest) (*Account, error) {
	response := &Account{}
	if err := s.create(ctx, account, response); err != nil {
		return nil, err
	}

	return response, nil
}

func (s *accounts) Update(ctx context.Context, id string, account *UpdateAccountRequest) (*Account, error) {
	response := &Account{}
	if err := s.update(ctx, id, account, response); err != nil {
		return nil, err
	}

	return response, nil
}

func (s *accounts) Delete(ctx context.Context, id string) error {
	return s.service.delete(ctx, id)
}

// SetExternalID assigns a caller-owned stable external id to a remote account.
// PUT /v2/accounts/{id}/external-id  body: {"externalId": externalID}
func (s *accounts) SetExternalID(ctx context.Context, id string, externalID string) error {
	path := fmt.Sprintf("%s/%s/external-id", s.basePath, id)
	data, err := json.Marshal(map[string]string{"externalId": externalID})
	if err != nil {
		return fmt.Errorf("marshalling external id: %w", err)
	}
	if _, err := s.client.Do(ctx, "PUT", path, bytes.NewReader(data)); err != nil {
		return fmt.Errorf("setting external id: %w", err)
	}
	return nil
}
