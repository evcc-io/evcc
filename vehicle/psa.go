package vehicle

import (
	"strings"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/util"
	"github.com/andig/evcc/vehicle/psa"
)

// https://github.com/flobz/psa_car_controller
// https://github.com/snaptec/openWB/blob/master/modules/soc_psa/psasoc.py

// Peugeot is an api.Vehicle implementation for Peugeot cars
type Peugeot struct {
	*embed
	*psa.Provider // provides the api implementations
}

func init() {
	registry.Add("peugeot", NewPeugeotFromConfig)
}

// NewPeugeotFromConfig creates a new vehicle
func NewPeugeotFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		Title                  string
		Capacity               int64
		ClientID, ClientSecret string
		User, Password, VIN    string
		Cache                  time.Duration
	}{
		Cache: interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	v := &Peugeot{
		embed: &embed{cc.Title, cc.Capacity},
	}

	// if (manufacturer == "Peugeot"):
	//     brand = 'peugeot.com'
	//     realm = 'clientsB2CPeugeot'
	// elif (manufacturer == "Citroen"):
	//     brand = 'citroen.com'
	//     realm = 'clientsB2CCitroen'
	// elif (manufacturer == "DS"):
	//     brand = 'driveds.com'
	//     realm = 'clientsB2CDS'
	// elif (manufacturer == "Opel"):
	//     brand = 'opel.com'
	//     realm = 'clientsB2COpel'
	// elif (manufacturer == "Vauxhall"):
	//     brand = 'vauxhall.co.uk'
	//     realm = 'clientsB2CVauxhall'

	log := util.NewLogger("peugeot")
	api := psa.NewAPI(log, "peugeot.com", "clientsB2CPeugeot", cc.ClientID, cc.ClientSecret)

	err := api.Login(cc.User, cc.Password)
	if err == nil {
		if cc.VIN == "" {
			cc.VIN, err = findVehicle(api.Vehicles())
			if err == nil {
				log.DEBUG.Printf("found vehicle: %v", cc.VIN)
			}
		}
	}

	v.Provider = psa.NewProvider(api, strings.ToUpper(cc.VIN), cc.Cache)

	return v, err
}
