package vehicle

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/provider"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"

	"github.com/joeshaw/carwings"
)

const (
	carwingsRequestTimeout = 90 * time.Second
	carwingsStatusExpiry   = 5 * time.Minute // if returned status value is older, evcc will init refresh
	carwingsRefreshTimeout = 2 * time.Minute // timeout to get status after refresh
)

// CarWings is an api.Vehicle implementation for CarWings cars
type CarWings struct {
	*embed
	user, password string
	session        *carwings.Session
	statusG        func() (carwings.BatteryStatus, error)
	climateG       func() (carwings.ClimateStatus, error)
	refreshKey     string
	refreshTime    time.Time
}

func init() {
	registry.Add("carwings", NewCarWingsFromConfig)
}

// NewCarWingsFromConfig creates a new vehicle
func NewCarWingsFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		embed                       `mapstructure:",squash"`
		User, Password, Region, VIN string
		Cache                       time.Duration
	}{
		Region: carwings.RegionEurope,
		Cache:  interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.User == "" || cc.Password == "" {
		return nil, api.ErrMissingCredentials
	}

	log := util.NewLogger("carwings").Redact(cc.User, cc.Password, cc.VIN)

	// http client with high dial/handshake timeout
	transport := request.NewTripper(log, &http.Transport{
		Proxy: http.ProxyFromEnvironment, // default
		DialContext: (&net.Dialer{
			Timeout:   carwingsRequestTimeout,
			KeepAlive: 30 * time.Second, // default
		}).DialContext,
		TLSHandshakeTimeout:   carwingsRequestTimeout,
		ForceAttemptHTTP2:     true,             // default
		MaxIdleConns:          100,              // default
		IdleConnTimeout:       90 * time.Second, // default
		ExpectContinueTimeout: 1 * time.Second,  // default
	})

	carwings.Client = &http.Client{
		Timeout:   carwingsRequestTimeout,
		Transport: transport,
	}

	v := &CarWings{
		embed:    &cc.embed,
		user:     cc.User,
		password: cc.Password,
		session: &carwings.Session{
			Region: cc.Region,
			VIN:    cc.VIN,
		},
	}

	// initial connect
	if err := v.session.Connect(v.user, v.password); err != nil {
		return nil, fmt.Errorf("login failed: %w", err)
	}

	v.statusG = provider.Cached(v.status, cc.Cache)
	v.climateG = provider.Cached(v.session.ClimateControlStatus, cc.Cache)

	return v, nil
}

// connectIfRequired will return ErrMustRetry if ErrNotLoggedIn error could be resolved
func (v *CarWings) connectIfRequired(err error) error {
	if errors.Is(err, carwings.ErrNotLoggedIn) || err.Error() == "received status code 404" {
		if err = v.session.Connect(v.user, v.password); err == nil {
			err = api.ErrMustRetry
		}
	}
	return err
}

func (v *CarWings) status() (carwings.BatteryStatus, error) {
	// api result is stale
	if v.refreshKey != "" {
		if err := v.refreshResult(); err != nil {
			return *new(carwings.BatteryStatus), err
		}
	}

	bs, err := v.session.BatteryStatus()

	if err == nil {
		if elapsed := time.Since(bs.Timestamp); elapsed > carwingsStatusExpiry {
			if err = v.refreshRequest(); err != nil {
				return *new(carwings.BatteryStatus), err
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

	return bs, err
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

// Soc implements the api.Vehicle interface
func (v *CarWings) Soc() (float64, error) {
	res, err := v.statusG()
	if err == nil {
		return float64(res.StateOfCharge), nil
	}

	return 0, err
}

var _ api.ChargeState = (*CarWings)(nil)

// Status implements the api.ChargeState interface
func (v *CarWings) Status() (api.ChargeStatus, error) {
	status := api.StatusA // disconnected

	res, err := v.statusG()
	if err == nil {
		if res.PluginState == carwings.Connected {
			status = api.StatusB // connected, not charging
		}
		if res.ChargingStatus == carwings.NormalCharging {
			status = api.StatusC // charging
		}
	}

	return status, err
}

var _ api.VehicleRange = (*CarWings)(nil)

// Range implements the api.VehicleRange interface
func (v *CarWings) Range() (int64, error) {
	res, err := v.statusG()
	if err == nil {
		return int64(res.CruisingRangeACOn) / 1000, nil
	}

	return 0, err
}

var _ api.VehicleClimater = (*CarWings)(nil)

// Climater implements the api.VehicleClimater interface
func (v *CarWings) Climater() (bool, error) {
	res, err := v.climateG()

	// silence ErrClimateStatusUnavailable errors
	if errors.Is(err, carwings.ErrClimateStatusUnavailable) {
		return false, nil
	}

	if err == nil {
		return res.Running, nil
	}

	return false, err
}
