package vehicle

import (
	"fmt"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/vehicle/octopuskraken"
)

// OctopusIt is an api.Vehicle implementation for Octopus Energy Italy, reusing
// the Germany implementation's Kraken API client.
type OctopusIt struct {
	*embed
	*octopuskraken.API
	account  string
	device   string
	deviceID string
	dataG    func() (octopuskraken.Device, error)
}

func init() {
	registry.Add("octopus-it", NewOctopusItFromConfig)
}

// NewOctopusItFromConfig creates a new vehicle
func NewOctopusItFromConfig(other map[string]any) (api.Vehicle, error) {
	cc := struct {
		embed         `mapstructure:",squash"`
		Email         string
		Password      string
		AccountNumber string
		Device        string
		Cache         time.Duration
	}{
		Cache: interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.Email == "" || cc.Password == "" {
		return nil, api.ErrMissingCredentials
	}

	log := util.NewLogger("octopus-it").Redact(cc.Email, cc.Password)

	api, err := octopuskraken.NewAPI(log, octopuskraken.ItBaseURI, cc.Email, cc.Password)
	if err != nil {
		return nil, err
	}

	v := &OctopusIt{
		embed:   &cc.embed,
		API:     api,
		account: cc.AccountNumber,
		device:  cc.Device,
	}

	v.dataG = util.Cached(v.status, cc.Cache)

	return v, nil
}

// status fetches the live state of the configured device, resolving the account
// and matching device on first use.
func (v *OctopusIt) status() (octopuskraken.Device, error) {
	account, err := v.Account(v.account)
	if err != nil {
		return octopuskraken.Device{}, err
	}
	v.account = account

	devices, err := v.Devices(account)
	if err != nil {
		return octopuskraken.Device{}, err
	}

	for _, d := range devices {
		// match the configured device by id or name, or take the first one
		if v.deviceID != "" {
			if d.ID == v.deviceID {
				return d, nil
			}
			continue
		}
		if v.device == "" || strings.EqualFold(d.ID, v.device) || strings.EqualFold(d.Name, v.device) {
			v.deviceID = d.ID
			v.fromVehicle(d.Name, 0)
			return d, nil
		}
	}

	if v.device == "" {
		return octopuskraken.Device{}, api.ErrNotAvailable
	}
	return octopuskraken.Device{}, fmt.Errorf("device not found: %s", v.device)
}

// Soc implements the api.Vehicle interface
func (v *OctopusIt) Soc() (float64, error) {
	res, err := v.dataG()
	if err != nil {
		return 0, err
	}

	soc, ok := res.Soc()
	if !ok {
		return 0, api.ErrNotAvailable
	}

	return soc, nil
}

var _ api.SocLimiter = (*OctopusIt)(nil)

// GetLimitSoc implements the api.SocLimiter interface
func (v *OctopusIt) GetLimitSoc() (int64, error) {
	res, err := v.dataG()
	if err != nil {
		return 0, err
	}

	soc, ok := res.TargetSoc()
	if !ok {
		return 0, api.ErrNotAvailable
	}

	return int64(soc), nil
}
