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
		region:   cc.Region,
	}

	err := v.initSession()
	if err != nil {
		v.session = nil
	}

	v.statusG = provider.NewCached(func() (interface{}, error) {
		return v.status()
	}, cc.Cache).InterfaceGetter()
	return v, nil
}

// Init new carwings session
func (v *CarWings) initSession() error {
	v.session = &carwings.Session{
		Region: v.region,
	}
	v.refreshKey = ""
	return v.session.Connect(v.user, v.password)
}

// carwingsVehicleStatus holds the relevant data extracted from carwings API
// on vehicle status request
type carwingsVehicleStatus struct {
	VehicleStatus struct {
		SOC            float64
		Range          int64
		ChargingStatus carwings.ChargingStatus
		PlugStatus     carwings.PluginState
		ClimateStatus  struct {
			active      bool
			targetTemp  float64
			outsideTemp float64
		}
		LastRefresh time.Time
	}
}

func (v *CarWings) status() (res carwingsVehicleStatus, err error) {
	// follow up requested refresh
	if v.refreshKey != "" {
		return v.refreshResult()
	}

	// otherwise start normal workflow
	if v.session == nil {
		if err := v.initSession(); err != nil {
			return res, api.ErrNotAvailable
		}
	}

	v.refreshKey, err = v.session.UpdateStatus()

	if err == nil {
		v.refreshTime = time.Now()
	}
	res, err = v.refreshResult()
	if err == nil {
		var lastUpdate time.Time
		lastUpdate = res.VehicleStatus.LastRefresh

		if elapsed := time.Since(lastUpdate); elapsed > carwingsStatusExpiry {
			v.log.DEBUG.Printf("vehicle status is outdated (age %v > %v), requesting refresh", elapsed, carwingsStatusExpiry)

			if err = v.refreshRequest(); err == nil {
				err = api.ErrMustRetry
			}
		}
	}

	return res, err
}

// refreshResult triggers an update if not already in progress, otherwise gets result
func (v *CarWings) refreshResult() (res carwingsVehicleStatus, err error) {

	finished, err := v.session.CheckUpdate(v.refreshKey)
	// update successful and completed
	if err == nil && finished {
		v.refreshKey = ""
		var bs carwings.BatteryStatus
		bs, err = v.session.BatteryStatus()
		if err == nil {
			res.VehicleStatus.SOC = float64(bs.StateOfCharge)
			res.VehicleStatus.Range = int64(bs.CruisingRangeACOn) / 1000
			res.VehicleStatus.ChargingStatus = bs.ChargingStatus
			res.VehicleStatus.PlugStatus = bs.PluginState
			res.VehicleStatus.LastRefresh = bs.Timestamp
		}
		var ccs carwings.ClimateStatus
		ccs, err = v.session.ClimateControlStatus()
		if err == nil {
			res.VehicleStatus.ClimateStatus.active = ccs.Running
			res.VehicleStatus.ClimateStatus.targetTemp = float64(ccs.Temperature)
			res.VehicleStatus.ClimateStatus.outsideTemp = 0
		}
		// Update results here
		return res, nil
	}

	// update still in progress, keep retrying
	if time.Since(v.refreshTime) < carwingsRefreshTimeout {
		return res, api.ErrMustRetry
	}

	// give up
	v.refreshKey = ""
	if err == nil {
		err = api.ErrTimeout
	}

	return res, err
}

// refreshRequest requests status refresh tracked by commandId
func (v *CarWings) refreshRequest() error {
	// otherwise start normal workflow
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
	res, err := v.statusG()
	if res, ok := res.(carwingsVehicleStatus); err == nil && ok {
		return float64(res.VehicleStatus.SOC), nil
	}

	return 0, err
}

var _ api.VehicleClimater = (*CarWings)(nil)

// Climater implements the api.Vehicle.Climater interface
func (v *CarWings) Climater() (active bool, outsideTemp float64, targetTemp float64, err error) {
	res, err := v.statusG()
	if res, ok := res.(carwingsVehicleStatus); err == nil && ok {
		active = res.VehicleStatus.ClimateStatus.active

		targetTemp = res.VehicleStatus.ClimateStatus.targetTemp

		outsideTemp = res.VehicleStatus.ClimateStatus.outsideTemp

		return active, outsideTemp, targetTemp, nil
	}

	return false, 0, 0, api.ErrNotAvailable
}

var _ api.VehicleRange = (*CarWings)(nil)

// Range implements the api.VehicleRange interface
func (v *CarWings) Range() (int64, error) {
	res, err := v.statusG()
	if res, ok := res.(carwingsVehicleStatus); err == nil && ok {
		return res.VehicleStatus.Range, nil
	}

	return 0, err
}

var _ api.ChargeState = (*CarWings)(nil)

// Status implements the api.ChargeState interface
func (v *CarWings) Status() (api.ChargeStatus, error) {
	status := api.StatusA // disconnected

	res, err := v.statusG()
	if res, ok := res.(carwingsVehicleStatus); err == nil && ok {
		if res.VehicleStatus.PlugStatus == carwings.Connected {
			status = api.StatusB // connected, not charging
		}
		if res.VehicleStatus.ChargingStatus == carwings.NormalCharging {
			status = api.StatusC // charging
		}
	}

	return status, err
}
