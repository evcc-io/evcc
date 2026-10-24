package charger

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/ecoflow"
	"github.com/evcc-io/evcc/util"
)

// EcoflowPowerPulse is an EcoFlow PowerPulse wallbox behind a PowerOcean
// gateway, controlled through the EcoFlow app MQTT broker.
type EcoflowPowerPulse struct {
	wb *ecoflow.Wallbox
}

func init() {
	registry.AddCtx("ecoflow-powerpulse", NewEcoflowPowerPulseFromConfig)
}

// NewEcoflowPowerPulseFromConfig creates an EcoFlow PowerPulse charger from generic config
func NewEcoflowPowerPulseFromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
	cc := struct {
		User, Password    string
		AppKey, AppSecret string
		Serial, Gateway   string
		Timeout           time.Duration
	}{
		Timeout: 3 * time.Minute,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.User == "" || cc.Password == "" {
		return nil, api.ErrMissingCredentials
	}

	if cc.AppKey == "" || cc.AppSecret == "" {
		return nil, errors.New("missing appkey or appsecret")
	}

	if cc.Serial == "" {
		return nil, errors.New("missing serial")
	}

	conn, err := ecoflow.NewSharedConnection(ctx, cc.User, cc.Password, ecoflow.AppKey{Key: cc.AppKey, Secret: cc.AppSecret})
	if err != nil {
		return nil, err
	}

	if cc.Gateway == "" {
		if cc.Gateway, err = conn.Gateway(); err != nil {
			return nil, err
		}
		util.NewLogger("ecoflow").DEBUG.Printf("%s: detected gateway %s", cc.Serial, cc.Gateway)
	}

	return NewEcoflowPowerPulse(conn, cc.Serial, cc.Gateway, cc.Timeout)
}

// NewEcoflowPowerPulse creates an EcoFlow PowerPulse charger on a connection.
// Telemetry older than timeout is reported as outdated.
func NewEcoflowPowerPulse(conn *ecoflow.Connection, serial, gateway string, timeout time.Duration) (*EcoflowPowerPulse, error) {
	wb, err := conn.Wallbox(serial, gateway, timeout)
	if err != nil {
		return nil, err
	}

	return &EcoflowPowerPulse{wb: wb}, nil
}

// Status implements the api.Charger interface
func (c *EcoflowPowerPulse) Status() (api.ChargeStatus, error) {
	hb, err := c.wb.Heartbeat()
	if err != nil {
		return api.StatusNone, err
	}

	switch hb.State {
	case ecoflow.StateUnplugged:
		return api.StatusA, nil
	case ecoflow.StatePlugged, ecoflow.StatePaused, ecoflow.StatePausedByVehicle, ecoflow.StateStoppedByWallbox:
		return api.StatusB, nil
	case ecoflow.StateCharging:
		return api.StatusC, nil
	case ecoflow.StateError:
		// evcc has no fault status, other chargers report faults as error too
		return api.StatusNone, errors.New("wallbox fault (state 9)")
	default:
		return api.StatusNone, fmt.Errorf("unknown state: %d", hb.State)
	}
}

// Enabled implements the api.Charger interface
func (c *EcoflowPowerPulse) Enabled() (bool, error) {
	return c.wb.Enabled()
}

// Enable implements the api.Charger interface
func (c *EcoflowPowerPulse) Enable(enable bool) error {
	return c.wb.StartStop(enable)
}

// MaxCurrent implements the api.Charger interface
func (c *EcoflowPowerPulse) MaxCurrent(current int64) error {
	// 32 A as in 241/44; a lower limit of the wallbox shows in GetMaxCurrent
	if current < 6 || current > 32 {
		return fmt.Errorf("invalid current %d A, must be 6-32 A", current)
	}

	return c.wb.SetMaxCurrent(uint64(current * 10))
}

var _ api.CurrentGetter = (*EcoflowPowerPulse)(nil)

// GetMaxCurrent implements the api.CurrentGetter interface
func (c *EcoflowPowerPulse) GetMaxCurrent() (float64, error) {
	raw, err := c.wb.MaxCurrent()
	if err != nil {
		return 0, err
	}

	return float64(raw) / 10, nil
}

var _ api.PhaseSwitcher = (*EcoflowPowerPulse)(nil)

// Phases1p3p implements the api.PhaseSwitcher interface
func (c *EcoflowPowerPulse) Phases1p3p(phases int) error {
	var mode uint64
	switch phases {
	case 1:
		mode = ecoflow.PhaseOne
	case 3:
		mode = ecoflow.PhaseThree
	default:
		return fmt.Errorf("invalid phases: %d", phases)
	}

	// while charging the wallbox stops, switches and starts again
	return c.wb.SetPhaseMode(mode)
}

var _ api.Meter = (*EcoflowPowerPulse)(nil)

// CurrentPower implements the api.Meter interface
func (c *EcoflowPowerPulse) CurrentPower() (float64, error) {
	hb, err := c.wb.Heartbeat()
	if err != nil {
		return 0, err
	}

	return float64(hb.PowerW), nil
}

var _ api.PhaseCurrents = (*EcoflowPowerPulse)(nil)

// Currents implements the api.PhaseCurrents interface
func (c *EcoflowPowerPulse) Currents() (float64, float64, float64, error) {
	hb, err := c.wb.Heartbeat()
	if err != nil {
		return 0, 0, 0, err
	}

	return threePhases("currents", hb.Amps)
}

var _ api.PhaseVoltages = (*EcoflowPowerPulse)(nil)

// Voltages implements the api.PhaseVoltages interface
func (c *EcoflowPowerPulse) Voltages() (float64, float64, float64, error) {
	hb, err := c.wb.Heartbeat()
	if err != nil {
		return 0, 0, 0, err
	}

	return threePhases("voltages", hb.Volts)
}

func threePhases(name string, v []float32) (float64, float64, float64, error) {
	if len(v) != 3 {
		return 0, 0, 0, fmt.Errorf("%s: expected 3 phases, got %d", name, len(v))
	}

	return float64(v[0]), float64(v[1]), float64(v[2]), nil
}

var _ api.MeterEnergy = (*EcoflowPowerPulse)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (c *EcoflowPowerPulse) TotalEnergy() (float64, error) {
	hb, err := c.wb.Heartbeat()
	if err != nil {
		return 0, err
	}

	if hb.TotalEnergyWh == 0 {
		return 0, errors.New("energy counter not reported yet")
	}

	return float64(hb.TotalEnergyWh) / 1e3, nil
}
