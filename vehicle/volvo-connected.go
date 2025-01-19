package vehicle

import (
	"fmt"
	"os"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/vehicle/volvo/connected"
	"golang.org/x/oauth2"
)

// VolvoConnected is an api.Vehicle implementation for Volvo Connected Car vehicles
type VolvoConnected struct {
	*embed
	*connected.Identity
	*connected.Provider
}

func init() {
	registry.Add("volvo-connected", NewVolvoConnectedFromConfig)
}

// NewVolvoConnectedFromConfig creates a new VolvoConnected vehicle
func NewVolvoConnectedFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		embed       `mapstructure:",squash"`
		User        string
		VIN         string
		Credentials ClientCredentials
		RedirectUri string
		VccApiKey   string
		Cache       time.Duration
	}{
		Cache: interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if err := cc.Credentials.Error(); err != nil {
		return nil, err
	}

	log := util.NewLogger("volvo-cc").Redact(cc.Credentials.ID, cc.Credentials.Secret, cc.VIN, cc.VccApiKey)

	oc := connected.Oauth2Config(log, cc.Credentials.ID, cc.Credentials.Secret, cc.RedirectUri)

	ts := connected.NewIdentity(log, oc)

	cv := oauth2.GenerateVerifier()
	fmt.Println(oc.AuthCodeURL("state", oauth2.AccessTypeOffline, oauth2.S256ChallengeOption(cv)))
	os.Exit(1)

	api := connected.NewAPI(log, ts, cc.VccApiKey)

	var err error
	cc.VIN, err = ensureVehicle(cc.VIN, api.Vehicles)

	v := &VolvoConnected{
		embed:    &cc.embed,
		Identity: ts,
		Provider: connected.NewProvider(api, cc.VIN, cc.Cache),
	}

	return v, err
}
