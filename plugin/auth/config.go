package auth

import (
	"context"
	"fmt"
	"strings"

	reg "github.com/evcc-io/evcc/util/registry"
	"golang.org/x/oauth2"
)

var registry = reg.New[oauth2.TokenSource]("auth")

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
