package vehicle

import (
	"net/http"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/provider"
)

const (
	vwURL = "https://www.portal.volkswagen-we.com"
)

// VW is an api.Vehicle implementation for VW cars
type VW struct {
	*embed
	accountID, pin string
	chargeStateG   provider.FloatGetter
}

// NewVWFromConfig creates a new vehicle
func NewVWFromConfig(log *api.Logger, other map[string]interface{}) api.Vehicle {
	cc := struct {
		Title          string
		Capacity       int64
		AccountID, PIN string
		Cache          time.Duration
	}{}
	api.DecodeOther(log, other, &cc)

	if err := session.Connect(cc.User, cc.Password); err != nil {
		log.FATAL.Fatalf("cannot create VW: %v", err)
	}

	v := &VW{
		accountID: cc.AccountID,
		pin:       cc.PIN,
	}

	v.chargeStateG = provider.NewCached(v.chargeState, cc.Cache).FloatGetter()

	return v
}

func (v *VW) headers(header *http.Header) {
	for k, v := range map[string]string{
		"Accept-Encoding": "gzip, deflate, br",
		"Accept-Language": "en-US,nl;q=0.7,en;q=0.3",
		"Accept":          "application/json, text/plain, */*",
		"Content-Type":    "application/json;charset=UTF-8",
		"User-Agent":      "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:68.0) Gecko/20100101 Firefox/68.0",
		"Connection":      "keep-alive",
		"Pragma":          "no-cache",
		"Cache-Control":   "no-cache",
	} {
		header.Set(k, v)
	}
}

// chargeState implements the Vehicle.ChargeState interface
func (v *VW) chargeState() (float64, error) {
	bs, err := v.session.BatteryStatus()
	return float64(bs.StateOfCharge), err
}

// ChargeState implements the Vehicle.ChargeState interface
func (v *VW) ChargeState() (float64, error) {
	return v.chargeStateG()
}
