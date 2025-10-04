package meter

//go:generate go tool decorate -f decorateHomeAssistant -b *HomeAssistant -r api.Meter -t "api.MeterEnergy,TotalEnergy,func() (float64, error)" -t "api.PhaseCurrents,Currents,func() (float64, float64, float64, error)" -t "api.PhaseVoltages,Voltages,func() (float64, float64, float64, error)"

import (
	"errors"
	"fmt"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/homeassistant"
)

// HomeAssistant meter implementation
type HomeAssistant struct {
	conn            *homeassistant.Connection
	power           string
	energy          string
	currentEntities [3]string
	voltageEntities [3]string
}

// parsePhases helper to turn a []string into a [3]string or error
func parsePhases(name string, cfg []string) ([3]string, error) {
	var arr [3]string
	if len(cfg) == 0 {
		return arr, nil
	}
	if len(cfg) != 1 && len(cfg) != 3 {
		return arr, fmt.Errorf("%s must contain either 1 entity (single-phase) or 3 entities (three-phase L1, L2, L3), got %d", name, len(cfg))
	}
	if len(cfg) == 1 {
		arr[0] = cfg[0]
	} else {
		copy(arr[:], cfg)
	}
	return arr, nil
}

func init() {
	registry.Add("homeassistant", NewHomeAssistantFromConfig)
}

// NewHomeAssistantFromConfig creates a HomeAssistant meter from generic config
func NewHomeAssistantFromConfig(other map[string]interface{}) (api.Meter, error) {
	cc := struct {
		URI      string   `mapstructure:"uri"`
		Token    string   `mapstructure:"token"`
		Power    string   `mapstructure:"power"`
		Energy   string   `mapstructure:"energy"`
		Currents []string `mapstructure:"currents"`
		Voltages []string `mapstructure:"voltages"`
	}{}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.Power == "" {
		return nil, errors.New("missing power sensor entity")
	}

	conn, err := homeassistant.NewConnection(cc.URI, cc.Token)
	if err != nil {
		return nil, err
	}

	m := &HomeAssistant{
		conn:   conn,
		power:  cc.Power,
		energy: cc.Energy,
	}

	// Set up phase currents (optional)
	currents, err := parsePhases("currents", cc.Currents)
	if err != nil {
		return nil, err
	}
	m.currentEntities = currents

	// Set up phase voltages (optional)
	voltages, err := parsePhases("voltages", cc.Voltages)
	if err != nil {
		return nil, err
	}
	m.voltageEntities = voltages

	// decorators for optional interfaces
	var meterEnergy func() (float64, error)
	var phaseCurrents func() (float64, float64, float64, error)
	var phaseVoltages func() (float64, float64, float64, error)

	if m.energy != "" {
		meterEnergy = m.totalEnergy
	}
	if m.currentEntities[0] != "" {
		phaseCurrents = m.currents
	}
	if m.voltageEntities[0] != "" {
		phaseVoltages = m.voltages
	}

	return decorateHomeAssistant(m, meterEnergy, phaseCurrents, phaseVoltages), nil
}

// NewHomeAssistant creates HomeAssistant meter
func NewHomeAssistant(uri, token, power, energy string, currents, voltages [3]string) (*HomeAssistant, error) {
	if power == "" {
		return nil, errors.New("missing power sensor entity")
	}

	conn, err := homeassistant.NewConnection(uri, token)
	if err != nil {
		return nil, err
	}

	m := &HomeAssistant{
		conn:            conn,
		power:           power,
		energy:          energy,
		currentEntities: currents,
		voltageEntities: voltages,
	}

	return m, nil
}

var _ api.Meter = (*HomeAssistant)(nil)

// CurrentPower implements the api.Meter interface
func (m *HomeAssistant) CurrentPower() (float64, error) {
	return m.conn.GetFloatState(m.power)
}

// totalEnergy implements the api.MeterEnergy interface (private for decorator)
func (m *HomeAssistant) totalEnergy() (float64, error) {
	if m.energy == "" {
		return 0, api.ErrNotAvailable
	}
	return m.conn.GetFloatState(m.energy)
}

// currents implements the api.PhaseCurrents interface (private for decorator)
func (m *HomeAssistant) currents() (float64, float64, float64, error) {
	if m.currentEntities[0] == "" {
		return 0, 0, 0, api.ErrNotAvailable
	}
	return m.conn.GetPhaseStates(m.currentEntities)
}

// voltages implements the api.PhaseVoltages interface (private for decorator)
func (m *HomeAssistant) voltages() (float64, float64, float64, error) {
	if m.voltageEntities[0] == "" {
		return 0, 0, 0, api.ErrNotAvailable
	}
	return m.conn.GetPhaseStates(m.voltageEntities)
}
