package auth

import (
	"context"
	"net/http"

	"github.com/evcc-io/evcc/util"
)

type nop struct{}

func init() {
	registry.AddCtx("nop", NewNopFromConfig)
}

func NewNopFromConfig(ctx context.Context, other map[string]any) (Authorizer, error) {
	var cc struct{}
	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return new(nop), nil
}

func (p *nop) Transport(base http.RoundTripper) http.RoundTripper {
	return base
}
