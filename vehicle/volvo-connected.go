package vehicle

import (
	"fmt"
	"net/http"
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

	cv := oauth2.GenerateVerifier()
	fmt.Println(oc.AuthCodeURL("state", oauth2.AccessTypeOffline, oauth2.S256ChallengeOption(cv)))
	os.Exit(1)

	var ts oauth2.TokenSource

	// ts, err := identity.Login(cc.User, cc.Password)
	// if err != nil {
	// 	return nil, err
	// }

	// api := connected.NewAPI(log, identity, cc.Sandbox)
	api := connected.NewAPI(log, ts, cc.VccApiKey)

	var err error
	cc.VIN, err = ensureVehicle(cc.VIN, api.Vehicles)

	v := &VolvoConnected{
		embed:    &cc.embed,
		Provider: connected.NewProvider(api, cc.VIN, cc.Cache),
	}

	return v, err
}

var _ api.AuthProvider = (*VolvoConnected)(nil)

func (v *VolvoConnected) SetCallbackParams(baseURL, redirectURL string, authenticated chan<- bool) {
	fmt.Println(baseURL, redirectURL)
}

func (v *VolvoConnected) LoginHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
	}
}

func (v *VolvoConnected) LogoutHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
	}
}
