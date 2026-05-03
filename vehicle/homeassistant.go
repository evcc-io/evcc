package vehicle

import (
	"errors"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/implement"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/homeassistant"
)

type HomeAssistant struct {
	*embed
	implement.Capabilities
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
		Home_   string `mapstructure:"home"`  // TODO deprecated
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
			Start         string `mapstructure:"start_charging"` // script.* or switch.* optional
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

	conn, err := homeassistant.NewConnection(log, cc.URI, cc.Home_)
	if err != nil {
		return nil, err
	}

	res := &HomeAssistant{
		embed:        &cc.embed,
		Capabilities: implement.Caps(),
		conn:         conn,
		soc:          cc.Sensors.Soc,
	}

	if cc.Sensors.LimitSoc != "" {
		implement.Implements(res, implement.SocLimiter(func() (int64, error) {
			f, err := conn.GetFloatState(cc.Sensors.LimitSoc)
			return int64(f), err
		}))
	}
	if cc.Sensors.Status != "" {
		implement.Implements(res, implement.ChargeState(func() (api.ChargeStatus, error) { return conn.GetChargeStatus(cc.Sensors.Status) }))
	}
	if cc.Sensors.Range != "" {
		implement.Implements(res, implement.VehicleRange(func() (int64, error) {
			f, err := conn.GetFloatState(cc.Sensors.Range)
			return int64(f), err
		}))
	}
	if cc.Sensors.Odometer != "" {
		implement.Implements(res, implement.VehicleOdometer(func() (float64, error) { return conn.GetFloatState(cc.Sensors.Odometer) }))
	}
	if cc.Sensors.Climater != "" {
		implement.Implements(res, implement.VehicleClimater(func() (bool, error) { return conn.GetBoolState(cc.Sensors.Climater) }))
	}
	if cc.Sensors.FinishTime != "" {
		implement.Implements(res, implement.VehicleFinishTimer(func() (time.Time, error) { return conn.GetTimeState(cc.Sensors.FinishTime) }))
	}

	var enable func(bool) error
	if cc.Services.Start != "" && cc.Services.Stop != "" {
		enable = func(enable bool) error {
			if enable {
				return conn.CallSwitchService(cc.Services.Start, true)
			}
			return conn.CallSwitchService(cc.Services.Stop, true)
		}
	} else if strings.HasPrefix(cc.Services.Start, "switch") {
		enable = func(enable bool) error { return conn.CallSwitchService(cc.Services.Start, enable) }
	}
	if enable != nil {
		implement.Implements(res, implement.ChargeController(enable))
	}

	if cc.Services.Wakeup != "" {
		implement.Implements(res, implement.Resurrector(func() error { return conn.CallSwitchService(cc.Services.Wakeup, true) }))
	}
	if cc.Services.SetMaxCurrent != "" {
		implement.Implements(res, implement.CurrentController(func(current int64) error { return conn.CallNumberService(cc.Services.SetMaxCurrent, float64(current)) }))
	}

	return res, nil
}

func (v *HomeAssistant) Soc() (float64, error) {
	return v.conn.GetFloatState(v.soc)
}
