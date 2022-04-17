package vehicle

import (
	_ "embed"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	persist "github.com/evcc-io/evcc/util/store"
	"github.com/evcc-io/evcc/vehicle/audi"
	"github.com/evcc-io/evcc/vehicle/vag"
	"github.com/evcc-io/evcc/vehicle/vag/idkproxy"
	"github.com/evcc-io/evcc/vehicle/vag/service"
	"github.com/evcc-io/evcc/vehicle/vw"
)

// https://github.com/davidgiga1993/AudiAPI
// https://github.com/TA2k/ioBroker.vw-connect

// Audi is an api.Vehicle implementation for Audi cars
type Audi struct {
	*embed
	*vw.Provider // provides the api implementations
}

func init() {
	registry.AddWithStore("audi", NewAudiFromConfig)
}

//go:embed all:.auditoken
var audiToken string

// NewAudiFromConfig creates a new vehicle
func NewAudiFromConfig(other map[string]interface{}, store persist.Store) (api.Vehicle, error) {
	cc := struct {
		embed               `mapstructure:",squash"`
		User, Password, VIN string
		Cache               time.Duration
		Timeout             time.Duration
	}{
		Cache:   interval,
		Timeout: request.Timeout,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	v := &Audi{
		embed: &cc.embed,
	}

	log := util.NewLogger("audi").Redact(cc.User, cc.Password, cc.VIN)

	hash := persist.Hash(vag.IdkToken, cc.User, cc.Password)
	stp := vag.StoreTokenProvider(store, hash, audiToken)

	idk := idkproxy.New(log, audi.IDKParams)
	rts, err := service.RefreshTokenSource(log, idk, stp, audi.AuthParams, cc.User, cc.Password)
	if err != nil {
		return nil, err
	}

	ts, err := service.MbbTokenSource(log, rts, audi.AuthClientID)
	if err != nil {
		return nil, err
	}

	api := vw.NewAPI(log, ts, audi.Brand, audi.Country)
	api.Client.Timeout = cc.Timeout

	cc.VIN, err = ensureVehicle(cc.VIN, api.Vehicles)

	if err == nil {
		if err = api.HomeRegion(cc.VIN); err == nil {
			v.Provider = vw.NewProvider(api, cc.VIN, cc.Cache)
		}
	}

	return v, err
}
