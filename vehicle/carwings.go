package vehicle

import (
	"errors"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/provider"
	"github.com/andig/evcc/util"

	"github.com/joeshaw/carwings"
)

const (
	carwingsStatusExpiry   = 5 * time.Minute // if returned status value is older, evcc will init refresh
	carwingsRefreshTimeout = 2 * time.Minute // timeout to get status after refresh
)

// CarWings is an api.Vehicle implementation for CarWings cars
type CarWings struct {
	*embed
	log            *util.Logger
	user, password string
	session        *carwings.Session
	statusG        func() (interface{}, error)
	refreshKey     string
	refreshTime    time.Time
}

func init() {
	registry.Add("carwings", NewCarWingsFromConfig)
}

// NewCarWingsFromConfig creates a new vehicle
func NewCarWingsFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		embed                  `mapstructure:",squash"`
		User, Password, Region string
		Cache                  time.Duration
	}{
		Region: carwings.RegionEurope,
		Cache:  interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.User == "" || cc.Password == "" {
		return nil, errors.New("missing credentials")
	}

	log := util.NewLogger("carwin")

	v := &CarWings{
		embed:    &cc.embed,
		log:      log,
		user:     cc.User,
		password: cc.Password,
		session:  &carwings.Session{Region: cc.Region},
	}

	v.statusG = provider.NewCached(func() (interface{}, error) {
		return nil, v.status()
	}, cc.Cache).InterfaceGetter()

	return v, nil
}

// connectIfRequired will return ErrMustRetry if ErrNotLoggedIn error could be resolved
func (v *CarWings) connectIfRequired(err error) error {
	if err == carwings.ErrNotLoggedIn || err.Error() == "received status code 404" {
		if err = v.session.Connect(v.user, v.password); err == nil {
			err = api.ErrMustRetry
		}
	}
	return err
}

func (v *CarWings) status() error {
	bs, err := v.session.BatteryStatus()
	if err == nil {
		if elapsed := time.Since(bs.Timestamp); elapsed > carwingsStatusExpiry {
			// api result is stale
			if v.refreshKey != "" {
				return v.refreshResult()
			}

			if err = v.refreshRequest(); err != nil {
				return err
			}

			err = api.ErrMustRetry
		} else {
			// reset if elapsed < carwingsStatusExpiry,
			// otherwise next check after soc timeout does not trigger update because refreshResult succeeds on old key
			v.refreshKey = ""
		}
	} else {
		err = v.connectIfRequired(err)
	}

	return err
}

// refreshResult triggers an update if not already in progress, otherwise gets result
func (v *CarWings) refreshResult() error {
	finished, err := v.session.CheckUpdate(v.refreshKey)

	// update successful and completed
	if err == nil && finished {
		v.refreshKey = ""
		return nil
	}

	// update still in progress, keep retrying
	if time.Since(v.refreshTime) < carwingsRefreshTimeout {
		return api.ErrMustRetry
	}

	// give up
	v.refreshKey = ""
	if err == nil {
		err = api.ErrTimeout
	}

	return err
}

// refreshRequest requests status refresh tracked by refreshKey
func (v *CarWings) refreshRequest() (err error) {
	if v.refreshKey, err = v.session.UpdateStatus(); err == nil {
		v.refreshTime = time.Now()
		if v.refreshKey == "" {
			err = errors.New("refresh failed")
		}
	} else {
		err = v.connectIfRequired(err)
	}

	return err
}

// SoC implements the api.Vehicle interface
func (v *CarWings) SoC() (soc float64, err error) {
	soc = 0

	if _, err = v.statusG(); err == nil {
		var bs carwings.BatteryStatus
		if bs, err = v.session.BatteryStatus(); err == nil {
			soc = float64(bs.StateOfCharge)
		}
	}

	return soc, err
}

var _ api.ChargeState = (*CarWings)(nil)

// Status implements the api.ChargeState interface
func (v *CarWings) Status() (status api.ChargeStatus, err error) {
	status = api.StatusA // disconnected

	if _, err = v.statusG(); err == nil {
		var bs carwings.BatteryStatus
		if bs, err = v.session.BatteryStatus(); err == nil {
			if bs.PluginState == carwings.Connected {
				status = api.StatusB // connected, not charging
			}
			if bs.ChargingStatus == carwings.NormalCharging {
				status = api.StatusC // charging
			}
		}
	}

	return status, err
}

var _ api.VehicleRange = (*CarWings)(nil)

// Range implements the api.VehicleRange interface
func (v *CarWings) Range() (rng int64, err error) {
	rng = 0

	if _, err = v.statusG(); err == nil {
		var bs carwings.BatteryStatus
		if bs, err = v.session.BatteryStatus(); err == nil {
			rng = int64(bs.CruisingRangeACOn) / 1000
		}
	}

	return rng, err
}

var _ api.VehicleClimater = (*CarWings)(nil)

// Climater implements the api.VehicleClimater interface
func (v *CarWings) Climater() (active bool, outsideTemp float64, targetTemp float64, err error) {
	if _, err = v.statusG(); err == nil {
		var ccs carwings.ClimateStatus
		if ccs, err = v.session.ClimateControlStatus(); err == nil {
			active = ccs.Running
			targetTemp = float64(ccs.Temperature)
			outsideTemp = targetTemp
		}

		return active, outsideTemp, targetTemp, err
	}

	return false, 0, 0, err
}
