package meter

import (
	"errors"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/homeassistant"
)

// HomeAssistant meter implementation
type HomeAssistant struct {
	conn     *homeassistant.Connection
	power    string
	energy   string
	currents [3]string
	voltages [3]string
}

func init() {
	registry.Add("homeassistant", NewHomeAssistantFromConfig)
}

// NewHomeAssistantFromConfig creates a HomeAssistant meter from generic config
func NewHomeAssistantFromConfig(other map[string]interface{}) (api.Meter, error) {
	cc := struct {
		BaseURL  string
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
		currents, err := homeassistant.ValidatePhaseEntities(cc.Currents, "currents")
		if err != nil {
			return nil, err
		}
		m.currents = currents
	}

	// Set up phase voltages (optional)
	if len(cc.Voltages) > 0 {
		voltages, err := homeassistant.ValidatePhaseEntities(cc.Voltages, "voltages")
		if err != nil {
			return nil, err
		}
		m.voltages = voltages
	}

	return m, nil
}

// NewHomeAssistant creates HomeAssistant meter
func NewHomeAssistant(baseURL, token, power, energy string, currents, voltages [3]string) (*HomeAssistant, error) {
	if power == "" {
		return nil, errors.New("missing power sensor entity")
	}

	conn, err := homeassistant.NewConnection(baseURL, token)
	if err != nil {
		return nil, err
	}

	m := &HomeAssistant{
		conn:     conn,
		power:    power,
		energy:   energy,
		currents: currents,
		voltages: voltages,
	}

	return m, nil
}

var _ api.Meter = (*HomeAssistant)(nil)

// CurrentPower implements the api.Meter interface
func (m *HomeAssistant) CurrentPower() (float64, error) {
	return m.conn.GetFloatState(m.power)
}

var _ api.MeterEnergy = (*HomeAssistant)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (m *HomeAssistant) TotalEnergy() (float64, error) {
	if m.energy == "" {
		return 0, api.ErrNotAvailable
	}
	return m.conn.GetFloatState(m.energy)
}

var _ api.PhaseCurrents = (*HomeAssistant)(nil)

// Currents implements the api.PhaseCurrents interface
func (m *HomeAssistant) Currents() (float64, float64, float64, error) {
	if m.currents[0] == "" {
		return 0, 0, 0, api.ErrNotAvailable
	}
	return m.conn.GetPhaseStates(m.currents)
}

var _ api.PhaseVoltages = (*HomeAssistant)(nil)

// Voltages implements the api.PhaseVoltages interface
func (m *HomeAssistant) Voltages() (float64, float64, float64, error) {
	if m.voltages[0] == "" {
		return 0, 0, 0, api.ErrNotAvailable
	}
	return m.conn.GetPhaseStates(m.voltages)
}
