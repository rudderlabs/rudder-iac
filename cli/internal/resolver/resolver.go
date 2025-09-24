package resolver

import (
	"context"
	"errors"
	"fmt"
)

var ErrReferenceNotFound = errors.New("reference not found")

type chainedResolvers struct {
	resolvers []ReferenceResolver
}

func (c *chainedResolvers) ResolveToReference(ctx context.Context, entityType string, remoteID string) (string, error) {
	var lastErr error

	for _, r := range c.resolvers {
		ref, err := r.ResolveToReference(ctx, entityType, remoteID)
		if err == nil {
			return ref, nil
		}
		if !errors.Is(err, ErrReferenceNotFound) {
			return "", fmt.Errorf("resolving reference: %w", err)
		}
		lastErr = err
	}

	if lastErr == nil {
		lastErr = ErrReferenceNotFound
	}
	return "", lastErr
}

func ChainResolvers(inOrder ...ReferenceResolver) ReferenceResolver {
	flat := make([]ReferenceResolver, 0, len(inOrder))
	for _, r := range inOrder {
		if r != nil {
			flat = append(flat, r)
		}
	}
	return &chainedResolvers{resolvers: flat}
}

type ReferenceResolver interface {
	ResolveToReference(ctx context.Context, entityType string, remoteID string) (string, error)
}
