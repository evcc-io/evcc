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
	region         string
	session        *carwings.Session
	statusG        func() (bool, error)
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
		region:   cc.Region,
	}

	v.statusG = provider.NewCached(func() (bool, error) {
		return v.status()
	}, cc.Cache).BoolGetter()
	return v, nil
}

// Init new carwings session
func (v *CarWings) initSession() error {
	v.session = &carwings.Session{
		Region: v.region,
	}
	return v.session.Connect(v.user, v.password)
}

func (v *CarWings) status() (success bool, err error) {
	// follow up requested refresh
	if v.refreshKey != "" {
		return v.refreshResult()
	}

	if err = v.refreshRequest(); err != nil {
		return false, err
	}

	success, err = v.refreshResult()
	if err == nil {
		bs, err := v.session.BatteryStatus()
		if err == nil {
			if elapsed := time.Since(bs.Timestamp); elapsed > carwingsStatusExpiry {
				v.log.DEBUG.Printf("vehicle status is outdated (age %v > %v), requesting refresh", elapsed, carwingsStatusExpiry)
				if err = v.refreshRequest(); err == nil {
					err = api.ErrMustRetry
				}
			}
		}
	}

	return success, err
}

// refreshResult triggers an update if not already in progress, otherwise gets result
func (v *CarWings) refreshResult() (success bool, err error) {
	finished, err := v.session.CheckUpdate(v.refreshKey)
	// update successful and completed
	if err == nil && finished {
		v.refreshKey = ""
		return finished, err
	}

	// update still in progress, keep retrying
	if time.Since(v.refreshTime) < carwingsRefreshTimeout {
		return false, api.ErrMustRetry
	}

	// give up
	v.refreshKey = ""
	if err == nil {
		err = api.ErrTimeout
	}

	return false, err
}

// refreshRequest requests status refresh tracked by refreshKey
func (v *CarWings) refreshRequest() error {
	if v.session == nil {
		if err := v.initSession(); err != nil {
			return api.ErrNotAvailable
		}
	}

	var err error
	v.refreshKey, err = v.session.UpdateStatus()
	if err == nil {
		v.refreshTime = time.Now()
		if v.refreshKey == "" {
			err = errors.New("refresh failed")
		}
	}

	return err
}

// SoC implements the api.Vehicle interface
func (v *CarWings) SoC() (float64, error) {
	ok, err := v.statusG()
	if err == nil && ok {
		var bs carwings.BatteryStatus
		bs, err = v.session.BatteryStatus()
		if err == nil {
			return float64(bs.StateOfCharge), nil
		}
	}

	return 0, err
}

var _ api.VehicleClimater = (*CarWings)(nil)

// Climater implements the api.Vehicle.Climater interface
func (v *CarWings) Climater() (active bool, outsideTemp float64, targetTemp float64, err error) {
	ok, err := v.statusG()
	if err == nil && ok {
		var ccs carwings.ClimateStatus
		ccs, err = v.session.ClimateControlStatus()
		if err == nil {
			active = ccs.Running
			targetTemp = float64(ccs.Temperature)
			outsideTemp = targetTemp
		}

		return active, outsideTemp, targetTemp, nil
	}

	return false, 0, 0, api.ErrNotAvailable
}

var _ api.VehicleRange = (*CarWings)(nil)

// Range implements the api.VehicleRange interface
func (v *CarWings) Range() (Range int64, err error) {
	ok, err := v.statusG()
	if err == nil && ok {
		var bs carwings.BatteryStatus
		bs, err = v.session.BatteryStatus()
		if err == nil {
			Range = int64(bs.CruisingRangeACOn) / 1000
		}
		return Range, nil
	}

	return 0, err
}

var _ api.ChargeState = (*CarWings)(nil)

// Status implements the api.ChargeState interface
func (v *CarWings) Status() (api.ChargeStatus, error) {
	status := api.StatusA // disconnected

	ok, err := v.statusG()
	if err == nil && ok {
		var bs carwings.BatteryStatus
		bs, err = v.session.BatteryStatus()
		if err == nil {
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
