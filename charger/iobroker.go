package charger

import (
	"errors"
	"fmt"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/implement"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/iobroker"
)

// HomeAssistant charger implementation
type Iobroker struct {
	implement.Caps
	conn       *iobroker.Connection
	status     string
	enabled    string
	enable     string
	maxcurrent string
}

func init() {
	registry.Add("iobroker", NewIobrokerFromConfig)
}

// NewIobrokerFromConfig creates an Iobroker charger from generic config
func NewIobrokerFromConfig(other map[string]any) (api.Charger, error) {
	var cc struct {
		Name       string   `mapstructure:"iobroker"`
		Status     string   // required - sensor for charge status
		Enabled    string   // required - sensor for enabled state
		Enable     string   // required - switch/input_boolean for enable/disable
		MaxCurrent string   // required - number entity for setting max current
		Power      string   // optional - power sensor
		Energy     string   // optional - energy sensor
		Currents   []string // optional - current sensors for L1, L2, L3
		Voltages   []string // optional - voltage sensors for L1, L2, L3
		Phases     string   // optional - select entity for 1p/3p phase switching
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.Status == "" {
		return nil, errors.New("missing status sensor entity")
	}
	if cc.Enabled == "" {
		return nil, errors.New("missing enabled sensor entity")
	}
	if cc.Enable == "" {
		return nil, errors.New("missing enable switch entity")
	}
	if cc.MaxCurrent == "" {
		return nil, errors.New("missing maxcurrent number entity")
	}

	conn := iobroker.GetConnection(cc.Name)
	if conn == nil {
		return nil, errors.New("Iobroker instance unknown: " + cc.Name)
	}

	c := &Iobroker{
		Caps:       implement.New(),
		conn:       conn,
		status:     cc.Status,
		enabled:    cc.Enabled,
		enable:     cc.Enable,
		maxcurrent: cc.MaxCurrent,
	}

	if cc.Power != "" {
		implement.Has(c, implement.Meter(func() (float64, error) { return conn.GetFloatState(cc.Power) }))
	}
	if cc.Energy != "" {
		implement.Has(c, implement.MeterEnergy(func() (float64, error) { return conn.GetFloatState(cc.Energy) }))
	}

	// phase currents (optional)
	if phases, err := iobroker.ValidatePhaseEntities(cc.Currents); len(phases) > 0 {
		implement.Has(c, implement.PhaseCurrents(func() (float64, float64, float64, error) { return conn.GetPhaseFloatStates(phases) }))
	} else if err != nil {
		return nil, fmt.Errorf("currents: %w", err)
	}

	// phase voltages (optional)
	if phases, err := iobroker.ValidatePhaseEntities(cc.Voltages); len(phases) > 0 {
		implement.Has(c, implement.PhaseVoltages(func() (float64, float64, float64, error) { return conn.GetPhaseFloatStates(phases) }))
	} else if err != nil {
		return nil, fmt.Errorf("voltages: %w", err)
	}

	// phase switching (optional)
	if cc.Phases != "" {
		implement.Has(c, implement.PhaseSwitcher(func(phases int) error {
			return conn.SetIntState(cc.Phases, int64(phases))
		}))

		implement.Has(c, implement.PhaseGetter(func() (int, error) {
			val, err := conn.GetIntState(cc.Phases)
			if err != nil {
				return 0, err
			}
			return int(val), nil
		}))
	}

	return c, nil
}

var _ api.Charger = (*HomeAssistant)(nil)

// Status implements the api.ChargeState interface
func (c *Iobroker) Status() (api.ChargeStatus, error) {
	return c.conn.GetChargeStatus(c.status)
}

// Enabled implements the api.Charger interface
func (c *Iobroker) Enabled() (bool, error) {
	return c.conn.GetBoolState(c.enabled)
}

// Enable implements the api.Charger interface
func (c *Iobroker) Enable(enable bool) error {
	return c.conn.SetBoolState(c.enable, enable)
}

// MaxCurrent implements the api.Charger interface
func (c *Iobroker) MaxCurrent(current int64) error {
	return c.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*HomeAssistant)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (c *Iobroker) MaxCurrentMillis(current float64) error {
	return c.conn.SetFloatState(c.maxcurrent, current)
}
