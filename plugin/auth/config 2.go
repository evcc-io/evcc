package auth

import (
	"context"
	"fmt"
	"strings"

	reg "github.com/evcc-io/evcc/util/registry"
)

var registry = reg.New[Authorizer]("auth")

// NewFromConfig creates auth from configuration
func NewFromConfig(ctx context.Context, typ string, other map[string]any) (Authorizer, error) {
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
