package vehicle

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/implement"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/homeassistant"
)

type HomeAssistant struct {
	*embed
	implement.Caps
	conn      *homeassistant.Connection
	soc       string
	statusMap map[string]string
}

// Register on startup
func init() {
	registry.Add("homeassistant", NewHomeAssistantVehicleFromConfig)
}

// Constructor from YAML config
func validateStatusMap(statusMap map[string]string) (map[string]string, error) {
	if len(statusMap) == 0 {
		return nil, nil
	}

	for key, value := range statusMap {
		trimmedKey := strings.TrimSpace(key)
		trimmedValue := strings.TrimSpace(value)
		if trimmedKey == "" {
			return nil, fmt.Errorf("contains an empty key")
		}
		if _, err := homeassistant.NormalizeChargeStatus(trimmedValue); err != nil {
			return nil, fmt.Errorf("%q has invalid value %q: must be one of A, B, C", trimmedKey, trimmedValue)
		}
	}

	return statusMap, nil
}

func NewHomeAssistantVehicleFromConfig(other map[string]any) (api.Vehicle, error) {
	var cc struct {
		embed                `mapstructure:",squash"`
		homeassistant.Config `mapstructure:",squash"`
		Sensors struct {
			Soc        string // required
			Range      string // optional
			Status     string // optional
			StatusMap  map[string]string `mapstructure:"statusMap"` // optional
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

	if cc.Sensors.StatusMap != nil {
		if _, err := validateStatusMap(cc.Sensors.StatusMap); err != nil {
			return nil, fmt.Errorf("sensors.statusMap: %w", err)
		}
	}

	if cc.Sensors.Soc == "" {
		return nil, errors.New("missing soc sensor")
	}

	log := util.NewLogger("ha-vehicle")

	conn, err := cc.Config.NewConnection(log)
	if err != nil {
		return nil, err
	}

	statusMap := homeassistant.NormalizeStatusMap(cc.Sensors.StatusMap)

	res := &HomeAssistant{
		embed:     &cc.embed,
		Caps:      implement.New(),
		conn:      conn,
		soc:       cc.Sensors.Soc,
		statusMap: statusMap,
	}

	if cc.Sensors.LimitSoc != "" {
		implement.Has(res, implement.SocLimiter(func() (int64, error) {
			f, err := conn.GetFloatState(cc.Sensors.LimitSoc)
			return int64(f), err
		}))
	}
	if cc.Sensors.Status != "" {
		implement.Has(res, implement.ChargeState(func() (api.ChargeStatus, error) { return conn.GetChargeStatus(cc.Sensors.Status, res.statusMap) }))
	}
	if cc.Sensors.Range != "" {
		implement.Has(res, implement.VehicleRange(func() (int64, error) {
			f, err := conn.GetFloatState(cc.Sensors.Range)
			return int64(f), err
		}))
	}
	if cc.Sensors.Odometer != "" {
		implement.Has(res, implement.VehicleOdometer(func() (float64, error) { return conn.GetFloatState(cc.Sensors.Odometer) }))
	}
	if cc.Sensors.Climater != "" {
		implement.Has(res, implement.VehicleClimater(func() (bool, error) { return conn.GetBoolState(cc.Sensors.Climater) }))
	}
	if cc.Sensors.FinishTime != "" {
		implement.Has(res, implement.VehicleFinishTimer(func() (time.Time, error) { return conn.GetTimeState(cc.Sensors.FinishTime) }))
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
	implement.May(res, implement.ChargeController(enable))

	if cc.Services.Wakeup != "" {
		implement.Has(res, implement.Resurrector(func() error { return conn.CallSwitchService(cc.Services.Wakeup, true) }))
	}
	if cc.Services.SetMaxCurrent != "" {
		implement.Has(res, implement.CurrentController(func(current int64) error { return conn.CallNumberService(cc.Services.SetMaxCurrent, float64(current)) }))
	}

	return res, nil
}

func (v *HomeAssistant) Soc() (float64, error) {
	return v.conn.GetFloatState(v.soc)
}
