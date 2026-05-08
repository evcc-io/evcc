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
		Account  string
		User     string
		Password string
	}
	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}
	if cc.Account == "" {
		cc.Account = cc.User
	}

	hasCredentials := cc.User != "" && cc.Password != ""
	if !hasCredentials && !easee.HasPersistedAuth(cc.Account) {
		return nil, api.ErrMissingCredentials
	}

	log := util.NewLogger("easee").Redact(cc.User, cc.Password)
	return easee.PersistentTokenSource(log, cc.Account, cc.User, cc.Password)
}
