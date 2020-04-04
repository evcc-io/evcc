package vehicle

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/provider"

	"github.com/joeshaw/carwings"
)

const (
	gigyaURL = "https://renault-wrd-prod-1-euw1-myrapp-one.s3-eu-west-1.amazonaws.com/configuration/android/config_%s.json"
)

type gigyaResponse struct {
	Servers gigyaServers
}

type gigyaServers struct {
	GigyaProd gigyaServerKeys `json:"gigyaProd"`
	WiredProd gigyaServerKeys `json:"wiredProd"`
}

type gigyaServerKeys struct {
	Target string `json:"target"`
	APIKey string `json:"apikey"`
}

// Renault is an api.Vehicle implementation for Renault cars
type Renault struct {
	*embed
	session      *carwings.Session
	chargeStateG provider.FloatGetter
}

// NewRenaultFromConfig creates a new vehicle
func NewRenaultFromConfig(log *api.Logger, other map[string]interface{}) api.Vehicle {
	cc := struct {
		Title                  string
		Capacity               int64
		User, Password, Region string
		Cache                  time.Duration
	}{}
	api.DecodeOther(log, other, &cc)

	v := &Renault{
		embed: &embed{cc.Title, cc.Capacity},
	}

	if err := v.login(cc.Region); err != nil {
		log.FATAL.Fatalf("cannot create renault: %v", err)
	}

	v.chargeStateG = provider.NewCached(v.chargeState, cc.Cache).FloatGetter()

	return v
}

func (v *Renault) login(region string) error {
	resp, err := http.Get(fmt.Sprintf(gigyaURL, region))
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var gr gigyaResponse
	if err := json.Unmarshal(b, &gr); err != nil {
		return err
	}
}

// chargeState implements the Vehicle.ChargeState interface
func (v *Renault) chargeState() (float64, error) {
	bs, err := v.session.BatteryStatus()
	return float64(bs.StateOfCharge), err
}

// ChargeState implements the Vehicle.ChargeState interface
func (v *Renault) ChargeState() (float64, error) {
	return v.chargeStateG()
}
