package auth

import (
	reg "github.com/evcc-io/evcc/util/registry"
	"golang.org/x/oauth2"
)

var registry = reg.New[oauth2.TokenSource]("auth")

func Register(typ string, fun func(map[string]any) (oauth2.TokenSource, error)) {
	registry.Add(typ, fun)
}
