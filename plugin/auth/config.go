package auth

import (
	"context"
	"fmt"
	"strings"

	"golang.org/x/oauth2"
)

// NewFromConfig creates auth from configuration
func NewFromConfig(ctx context.Context, typ string, other map[string]any) (oauth2.TokenSource, error) {
	factory, err := registry.Get(strings.ToLower(typ))
	if err != nil {
		return nil, err
	}

	v, err := factory(ctx, other)
	if err != nil {
		err = fmt.Errorf("cannot create auth type '%s': %w", typ, err)
	}

	return v, err
}
