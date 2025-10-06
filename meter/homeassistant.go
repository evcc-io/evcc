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
		BaseURL  string   `mapstructure:"baseurl"`
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

	conn, err := homeassistant.NewConnection(cc.BaseURL, cc.Token)
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
	var meterEnergy func() (float64, error)
	var phaseCurrents func() (float64, float64, float64, error)
	var phaseVoltages func() (float64, float64, float64, error)

	if m.energy != "" {
		meterEnergy = m.TotalEnergy
	}
	if m.currentEntities[0] != "" {
		phaseCurrents = m.Currents
	}
	if m.voltageEntities[0] != "" {
		phaseVoltages = m.Voltages
	}

	return decorateHomeAssistant(m, meterEnergy, phaseCurrents, phaseVoltages), nil
}

// NewHomeAssistant creates HomeAssistant meter
func NewHomeAssistant(baseURL, token, power, energy string, currents, voltages []string) (*HomeAssistant, error) {
	if power == "" {
		return nil, errors.New("missing power sensor entity")
	}

	conn, err := homeassistant.NewConnection(baseURL, token)
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

// TotalEnergy implements the api.MeterEnergy interface
func (m *HomeAssistant) TotalEnergy() (float64, error) {
	if m.energy == "" {
		return 0, api.ErrNotAvailable
	}
	return m.conn.GetFloatState(m.energy)
}

// Currents implements the api.PhaseCurrents interface
func (m *HomeAssistant) Currents() (float64, float64, float64, error) {
	if m.currentEntities[0] == "" {
		return 0, 0, 0, api.ErrNotAvailable
	}
	return m.conn.GetPhaseStates(m.currentEntities)
}

// Voltages implements the api.PhaseVoltages interface
func (m *HomeAssistant) Voltages() (float64, float64, float64, error) {
	if m.voltageEntities[0] == "" {
		return 0, 0, 0, api.ErrNotAvailable
	}
	return m.conn.GetPhaseStates(m.voltageEntities)
}
