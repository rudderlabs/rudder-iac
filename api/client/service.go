package client

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
)

type service struct {
	basePath string
	client   *Client
}

func (s *service) next(ctx context.Context, paging Paging, result interface{}) (bool, error) {
	if paging.Next == "" {
		return false, nil
	}

	res, err := s.client.Do(ctx, "GET", paging.Next, nil)
	if err != nil {
		return false, err
	}

	if err = json.Unmarshal(res, result); err != nil {
		return false, err
	}

	return true, nil
}

func (s *service) list(ctx context.Context, result interface{}) error {
	_, err := s.next(ctx, Paging{Next: s.basePath}, result)
	return err
}

func (s *service) get(ctx context.Context, id string, result interface{}) error {
	res, err := s.client.Do(ctx, "GET", strings.Join([]string{s.basePath, id}, "/"), nil)
	if err != nil {
		return err
	}

	if err = json.Unmarshal(res, result); err != nil {
		return err
	}

	return nil
}

func (s *service) create(ctx context.Context, input interface{}, result interface{}) error {
	body, err := json.Marshal(input)
	if err != nil {
		return err
	}

	res, err := s.client.Do(ctx, "POST", s.basePath, bytes.NewReader(body))
	if err != nil {
		return err
	}

	if err = json.Unmarshal(res, result); err != nil {
		return err
	}

	return nil
}

func (s *service) update(ctx context.Context, id string, input interface{}, result interface{}) error {
	body, err := json.Marshal(input)
	if err != nil {
		return err
	}

	res, err := s.client.Do(ctx, "PUT", strings.Join([]string{s.basePath, id}, "/"), bytes.NewReader(body))
	if err != nil {
		return err
	}

	if err = json.Unmarshal(res, result); err != nil {
		return err
	}

	return nil
}

func (s *service) delete(ctx context.Context, id string) error {
	_, err := s.client.Do(ctx, "DELETE", strings.Join([]string{s.basePath, id}, "/"), nil)
	return err
}
