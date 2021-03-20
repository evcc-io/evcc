package vehicle

import (
	"errors"
	"strings"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/util"
	"github.com/andig/evcc/vehicle/volvo"
	"golang.org/x/oauth2"
)

// Volvo is an api.Vehicle implementation for Volvo cars
type Volvo struct {
	*embed
	*volvo.Provider
}

func init() {
	registry.Add("Volvo", NewVolvoFromConfig)
}

// NewVolvoFromConfig creates a new Volvo vehicle
func NewVolvoFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		Title                  string
		Capacity               int64
		ClientID, ClientSecret string
		Tokens                 Tokens
		VIN                    string
		Cache                  time.Duration
	}{
		Cache: interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.ClientID == "" && cc.Tokens.Access == "" {
		return nil, errors.New("missing credentials")
	}

	var options []volvo.ClientOption
	if cc.Tokens.Access != "" {
		options = append(options, volvo.WithToken(&oauth2.Token{
			AccessToken:  cc.Tokens.Access,
			RefreshToken: cc.Tokens.Refresh,
			Expiry:       time.Now(),
		}))
	}

	log := util.NewLogger("Volvo")

	identity, err := volvo.NewIdentity(log, cc.ClientID, cc.ClientSecret, options...)
	if err != nil {
		return nil, err
	}

	api := volvo.NewAPI(log, identity)

	v := &Volvo{
		embed:    &embed{cc.Title, cc.Capacity},
		Provider: volvo.NewProvider(api, strings.ToUpper(cc.VIN), cc.Cache),
	}

	return v, nil
}
