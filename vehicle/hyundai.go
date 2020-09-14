package vehicle

import (
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/util"
	"github.com/andig/evcc/vehicle/bluelink"
)

// Hyundai is an api.Vehicle implementation
type Hyundai struct {
	*embed
	*bluelink.API
}

func init() {
	registry.Add("hyundai", NewHyundaiFromConfig)
}

// NewHyundaiFromConfig creates a new Vehicle
func NewHyundaiFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		Title          string
		Capacity       int64
		User, Password string `validate:"required"`
		Cache          time.Duration
	}{}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	settings := bluelink.Config{
		URI:               "https://prd.eu-ccapi.hyundai.com:8080",
		TokenAuth:         "NmQ0NzdjMzgtM2NhNC00Y2YzLTk1NTctMmExOTI5YTk0NjU0OktVeTQ5WHhQekxwTHVvSzB4aEJDNzdXNlZYaG10UVI5aVFobUlGampvWTRJcHhzVg==",
		CCSPServiceID:     "6d477c38-3ca4-4cf3-9557-2a1929a94654",
		CCSPApplicationID: "99cfff84-f4e2-4be8-a5ed-e5b755eb6581",
	}

	log := util.NewLogger("hyundai")
	api, err := bluelink.New(log, cc.User, cc.Password, cc.Cache, settings)
	if err != nil {
		return nil, err
	}

	v := &Hyundai{
		embed: &embed{cc.Title, cc.Capacity},
		API:   api,
	}

	return v, nil
}
