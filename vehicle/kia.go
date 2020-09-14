package vehicle

import (
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/util"
	"github.com/andig/evcc/vehicle/bluelink"
)

// Kia is an api.Vehicle implementation
type Kia struct {
	*embed
	*bluelink.API
}

func init() {
	registry.Add("kia", NewKiaFromConfig)
}

// NewKiaFromConfig creates a new Vehicle
func NewKiaFromConfig(other map[string]interface{}) (api.Vehicle, error) {
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
		URI:               "https://prd.eu-ccapi.kia.com:8080",
		TokenAuth:         "ZmRjODVjMDAtMGEyZi00YzY0LWJjYjQtMmNmYjE1MDA3MzBhOnNlY3JldA==",
		CCSPServiceID:     "fdc85c00-0a2f-4c64-bcb4-2cfb1500730a",
		CCSPApplicationID: "693a33fa-c117-43f2-ae3b-61a02d24f417",
	}

	log := util.NewLogger("kia")
	api, err := bluelink.New(log, cc.User, cc.Password, cc.Cache, settings)
	if err != nil {
		return nil, err
	}

	v := &Kia{
		embed: &embed{cc.Title, cc.Capacity},
		API:   api,
	}

	return v, nil
}
