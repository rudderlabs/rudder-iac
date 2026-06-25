package client

import (
	"context"
	"encoding/json"
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
	Options   json.RawMessage `json:"options"`
	CreatedAt *time.Time      `json:"createdAt,omitempty"`
	UpdatedAt *time.Time      `json:"updatedAt,omitempty"`
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

func (s *accounts) List(ctx context.Context) (*AccountsPage, error) {
	page := &AccountsPage{}
	if err := s.list(ctx, page); err != nil {
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
