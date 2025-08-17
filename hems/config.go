package hems

import (
	"context"
	"fmt"
	"strings"

	"github.com/evcc-io/evcc/core/site"
	"github.com/evcc-io/evcc/server"
)

// HEMS describes the HEMS system interface
type HEMS interface {
	Run()
}

// Factory creates a HEMS from config
type Factory func(ctx context.Context, other map[string]interface{}, site site.API, httpd *server.HTTPd) (HEMS, error)

var registry = make(map[string]Factory)

// Add registers a HEMS factory
func Add(name string, factory Factory) {
	if _, exists := registry[name]; exists {
		panic(fmt.Sprintf("cannot register duplicate hems type: %s", name))
	}
	registry[name] = factory
}

// Types returns the list of types
func Types() []string {
	types := make([]string, 0, len(registry))
	for typ := range registry {
		types = append(types, typ)
	}
	return types
}

// NewFromConfig creates new HEMS from config
func NewFromConfig(ctx context.Context, typ string, other map[string]interface{}, site site.API, httpd *server.HTTPd) (HEMS, error) {
	factory, exists := registry[strings.ToLower(typ)]
	if !exists {
		return nil, fmt.Errorf("invalid hems type: %s", typ)
	}

	v, err := factory(ctx, other, site, httpd)
	if err != nil {
		return nil, fmt.Errorf("cannot create hems type '%s': %w", typ, err)
	}

	return v, nil
}
