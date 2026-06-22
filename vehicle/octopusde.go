package vehicle

import (
	"fmt"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/vehicle/octopusde"
)

// OctopusDe is an api.Vehicle implementation for the Octopus Energy Germany Kraken API
type OctopusDe struct {
	*embed
	*octopusde.API
	account  string
	device   string
	deviceID string
	dataG    func() (octopusde.Device, error)
}

func init() {
	registry.Add("octopus-de", NewOctopusDeFromConfig)
}

// NewOctopusDeFromConfig creates a new vehicle
func NewOctopusDeFromConfig(other map[string]any) (api.Vehicle, error) {
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

	log := util.NewLogger("octopus-de").Redact(cc.Email, cc.Password)

	api, err := octopusde.NewAPI(log, cc.Email, cc.Password)
	if err != nil {
		return nil, err
	}

	v := &OctopusDe{
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
func (v *OctopusDe) status() (octopusde.Device, error) {
	account, err := v.Account(v.account)
	if err != nil {
		return octopusde.Device{}, err
	}
	v.account = account

	devices, err := v.Devices(account)
	if err != nil {
		return octopusde.Device{}, err
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
		return octopusde.Device{}, api.ErrNotAvailable
	}
	return octopusde.Device{}, fmt.Errorf("device not found: %s", v.device)
}

// Soc implements the api.Vehicle interface
func (v *OctopusDe) Soc() (float64, error) {
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

var _ api.SocLimiter = (*OctopusDe)(nil)

// GetLimitSoc implements the api.SocLimiter interface
func (v *OctopusDe) GetLimitSoc() (int64, error) {
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
