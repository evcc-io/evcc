package plugin

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/evcc-io/evcc/plugin/auth"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/transport"
	"github.com/jpfielding/go-http-digest/pkg/digest"
)

// Auth is the authorization config
type Auth struct {
	Type, User, Password, Token string

	Source string
	Other  map[string]any `mapstructure:",remain"`
}

func (p *Auth) Transport(ctx context.Context, log *util.Logger, base http.RoundTripper) (http.RoundTripper, error) {
	switch strings.ToLower(p.Type) {
	case "digest":
		return digest.NewTransport(p.User, p.Password, base), nil

	case "basic":
		return transport.BasicAuth(p.User, p.Password, base), nil

	case "bearer":
		if p.Token == "" && p.Password != "" {
			p.Token = p.Password
			log.WARN.Println("using password for bearer auth is deprecated, use token instead")
		}
		return transport.BearerAuth(p.Token, base), nil

	default:
		if p.Source == "" {
			return nil, fmt.Errorf("unknown auth type '%s'", p.Type)
		}

		if p.User != "" {
			p.Other["user"] = p.User
		}
		if p.Password != "" {
			p.Other["password"] = p.Password
		}

		authorizer, err := auth.NewFromConfig(ctx, p.Source, p.Other)
		if err != nil {
			return nil, err
		}

		return authorizer.Transport(base)
	}
}
