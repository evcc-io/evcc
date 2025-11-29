package vehicle

import (
	"errors"
	"strconv"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/homeassistant"
)

type HomeAssistant struct {
	*embed
	conn *homeassistant.Connection
	soc  string
}

// Register on startup
func init() {
	registry.Add("homeassistant", NewHomeAssistantVehicleFromConfig)
}

// Constructor from YAML config
func NewHomeAssistantVehicleFromConfig(other map[string]any) (api.Vehicle, error) {
	var cc struct {
		embed   `mapstructure:",squash"`
		URI     string
		Token_  string `mapstructure:"token"` // TODO deprecated
		Home    string // TODO deprecated
		Sensors struct {
			Soc        string // required
			Range      string // optional
			Status     string // optional
			LimitSoc   string // optional
			Odometer   string // optional
			Climater   string // optional
			FinishTime string // optional
		}
		Services struct {
			Start         string `mapstructure:"start_charging"` // script.* optional
			Stop          string `mapstructure:"stop_charging"`  // script.* optional
			Wakeup        string // script.* optional
			SetMaxCurrent string // number.* or input_number.* optional
		}
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.Sensors.Soc == "" {
		return nil, errors.New("missing soc sensor")
	}

	log := util.NewLogger("ha-vehicle")

	conn, err := homeassistant.NewConnection(log, cc.URI, cc.Home)
	if err != nil {
		return nil, err
	}

	res := &HomeAssistant{
		embed: &cc.embed,
		conn:  conn,
		soc:   cc.Sensors.Soc,
	}

	// prepare optional feature functions with concise names
	var (
		limitSoc   func() (int64, error)
		status     func() (api.ChargeStatus, error)
		rng        func() (int64, error)
		odo        func() (float64, error)
		climater   func() (bool, error)
		finish     func() (time.Time, error)
		enable     func(bool) error
		wakeup     func() error
		maxcurrent func(int64) error
	)

	if cc.Sensors.LimitSoc != "" {
		limitSoc = func() (int64, error) {
			f, err := conn.GetFloatState(cc.Sensors.LimitSoc)
			return int64(f), err
		}
	}
	if cc.Sensors.Status != "" {
		status = func() (api.ChargeStatus, error) { return conn.GetChargeStatus(cc.Sensors.Status) }
	}
	if cc.Sensors.Range != "" {
		rng = func() (int64, error) {
			f, err := conn.GetFloatState(cc.Sensors.Range)
			return int64(f), err
		}
	}
	if cc.Sensors.Odometer != "" {
		odo = func() (float64, error) { return conn.GetFloatState(cc.Sensors.Odometer) }
	}
	if cc.Sensors.Climater != "" {
		climater = func() (bool, error) { return conn.GetBoolState(cc.Sensors.Climater) }
	}
	if cc.Sensors.FinishTime != "" {
		finish = func() (time.Time, error) { return res.finishTime(cc.Sensors.FinishTime) }
	}
	if cc.Services.Start != "" && cc.Services.Stop != "" {
		enable = func(enable bool) error { return res.enable(cc.Services.Start, cc.Services.Stop, enable) }
	}
	if cc.Services.Wakeup != "" {
		wakeup = func() error { return conn.CallSwitchService(cc.Services.Wakeup, true) }
	}
	if cc.Services.SetMaxCurrent != "" {
		maxcurrent = func(current int64) error { return conn.CallNumberService(cc.Services.SetMaxCurrent, float64(current)) }
	}

	// decorate all features
	return decorateVehicle(
		res,
		limitSoc,
		status,
		rng,
		odo,
		climater,
		maxcurrent,
		nil,
		finish,
		wakeup,
		enable,
	), nil
}

func (v *HomeAssistant) Soc() (float64, error) {
	return v.conn.GetFloatState(v.soc)
}

func (v *HomeAssistant) finishTime(entity string) (time.Time, error) {
	s, err := v.conn.GetState(entity)
	if err != nil {
		return time.Time{}, err
	}

	if ts, err := strconv.ParseInt(s, 10, 64); err == nil {
		return time.Unix(ts, 0), nil
	}

	return time.Parse(time.RFC3339, s)
}

func (v *HomeAssistant) enable(on, off string, enable bool) error {
	if enable {
		return v.conn.CallSwitchService(on, true)
	}

	return v.conn.CallSwitchService(off, true)
}
