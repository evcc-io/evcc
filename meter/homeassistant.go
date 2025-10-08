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
	currentEntities []string
	voltageEntities []string
}

func init() {
	registry.Add("homeassistant", NewHomeAssistantFromConfig)
}

// NewHomeAssistantFromConfig creates a HomeAssistant meter from generic config
func NewHomeAssistantFromConfig(other map[string]interface{}) (api.Meter, error) {
	cc := struct {
		URI      string
		Token    string
		Power    string
		Energy   string
		Currents []string
		Voltages []string
	}{}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.Power == "" {
		return nil, errors.New("missing power sensor entity")
	}

	log := util.NewLogger("ha-meter")
	conn, err := homeassistant.NewConnection(log, cc.URI, cc.Token)
	if err != nil {
		return nil, err
	}

	m := &HomeAssistant{
		conn:   conn,
		power:  cc.Power,
		energy: cc.Energy,
	}

	// Set up phase currents (optional)
	if len(cc.Currents) > 0 {
		currents, err := homeassistant.ValidatePhaseEntities(cc.Currents)
		if err != nil {
			return nil, fmt.Errorf("currents: %w", err)
		}
		m.currentEntities = currents
	}

	// Set up phase voltages (optional)
	if len(cc.Voltages) > 0 {
		voltages, err := homeassistant.ValidatePhaseEntities(cc.Voltages)
		if err != nil {
			return nil, fmt.Errorf("voltages: %w", err)
		}
		m.voltageEntities = voltages
	}

	// decorators for optional interfaces
	var energy func() (float64, error)
	var currents, voltages func() (float64, float64, float64, error)

	if m.energy != "" {
		energy = m.totalEnergy
	}
	if m.currentEntities[0] != "" {
		currents = m.currents
	}
	if m.voltageEntities[0] != "" {
		voltages = m.voltages
	}

	return decorateHomeAssistant(m, energy, currents, voltages), nil
}

var _ api.Meter = (*HomeAssistant)(nil)

// CurrentPower implements the api.Meter interface
func (m *HomeAssistant) CurrentPower() (float64, error) {
	return m.conn.GetFloatState(m.power)
}

// totalEnergy implements the api.MeterEnergy interface
func (m *HomeAssistant) totalEnergy() (float64, error) {
	return m.conn.GetFloatState(m.energy)
}

// currents implements the api.PhaseCurrents interface
func (m *HomeAssistant) currents() (float64, float64, float64, error) {
	return m.conn.GetPhaseFloatStates(m.currentEntities)
}

// voltages implements the api.PhaseVoltages interface
func (m *HomeAssistant) voltages() (float64, float64, float64, error) {
	return m.conn.GetPhaseFloatStates(m.voltageEntities)
}
