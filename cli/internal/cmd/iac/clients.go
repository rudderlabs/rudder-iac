package iac

import (
	"context"
	"fmt"
)

type Store struct {
	Pulumi *PulumiHelper
}

func NewStore(ctx context.Context, conf *PulumiConfig) (*Store, error) {
	h, err := NewPulumiHelper(ctx, conf)
	if err != nil {
		return nil, fmt.Errorf("creating pulumi helper: %w", err)
	}

	return &Store{
		Pulumi: h,
	}, nil
}
