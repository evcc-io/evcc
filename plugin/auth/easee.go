package auth

import (
	"context"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/easee"
	"github.com/evcc-io/evcc/util"
	"golang.org/x/oauth2"
)

func init() {
	registry.AddCtx("easee", newEaseeFromConfig)
}

func newEaseeFromConfig(_ context.Context, other map[string]any) (oauth2.TokenSource, error) {
	var cc struct {
		User     string
		Password string
	}
	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.User == "" {
		return nil, api.ErrMissingCredentials
	}

	if cc.Password == "" && !easee.HasPersistedAuth(cc.User) {
		return nil, api.ErrCredentialsRequired
	}

	log := util.NewLogger("easee").Redact(cc.User, cc.Password)
	return easee.PersistentTokenSource(log, cc.User, cc.Password)
}
