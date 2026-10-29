package vehicle

import (
	"context"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/vehicle/polestar/dataportal"
)

// PolestarDataPortal is an api.Vehicle implementation for Polestar cars using
// the official Polestar Data Portal M2M API.
type PolestarDataPortal struct {
	*embed
	*dataportal.Provider
}

func init() {
	registry.AddCtx("polestar-data-portal", NewPolestarDataPortalFromConfig)
}

// NewPolestarDataPortalFromConfig creates a new vehicle
func NewPolestarDataPortalFromConfig(ctx context.Context, other map[string]any) (api.Vehicle, error) {
	cc := struct {
		embed                             `mapstructure:",squash"`
		ClientID, ClientSecret, AccountID string
		VIN                               string
		Cache                             time.Duration
	}{
		Cache: 15 * time.Minute,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.ClientID == "" || cc.ClientSecret == "" || cc.AccountID == "" {
		return nil, api.ErrMissingCredentials
	}

	log := util.LoggerFromContext(ctx, "polestar").Redact(cc.ClientID, cc.ClientSecret, cc.AccountID, cc.VIN)

	identity := dataportal.NewIdentity(log, cc.ClientID, cc.ClientSecret)
	api := dataportal.NewAPI(log, identity, cc.AccountID)

	vin, err := ensureVehicle(cc.VIN, api.Vehicles)
	if err != nil {
		return nil, err
	}

	v := &PolestarDataPortal{
		embed:    &cc.embed,
		Provider: dataportal.NewProvider(api, strings.ToUpper(vin), cc.Cache),
	}

	return v, nil
}
