package meter

//go:generate go tool decorate -f decorateHomeAssistant -b *HomeAssistant -r api.Meter -t "api.MeterEnergy,TotalEnergy,func() (float64, error)" -t "api.PhaseCurrents,Currents,func() (float64, float64, float64, error)" -t "api.PhaseVoltages,Voltages,func() (float64, float64, float64, error)" -t "api.Battery,Soc,func() (float64, error)"

import (
	"errors"
	"fmt"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/homeassistant"
)

// HomeAssistant meter implementation
type HomeAssistant struct {
	conn  *homeassistant.Connection
	power string
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
		Soc      string
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
		conn:  conn,
		power: cc.Power,
	}

	// decorators for optional interfaces
	var energy func() (float64, error)
	var currents, voltages func() (float64, float64, float64, error)
	var soc func() (float64, error)

	if cc.Energy != "" {
		energy = func() (float64, error) { return conn.GetFloatState(cc.Energy) }
	}

	if len(cc.Currents) > 0 {
		phases, err := homeassistant.ValidatePhaseEntities(cc.Currents)
		if err != nil {
			return nil, fmt.Errorf("currents: %w", err)
		}
		currents = func() (float64, float64, float64, error) { return conn.GetPhaseFloatStates(phases) }
	}

	// Set up phase voltages (optional)
	if len(cc.Voltages) > 0 {
		phases, err := homeassistant.ValidatePhaseEntities(cc.Voltages)
		if err != nil {
			return nil, fmt.Errorf("voltages: %w", err)
		}
		voltages = func() (float64, float64, float64, error) { return conn.GetPhaseFloatStates(phases) }
	}

	if cc.Soc != "" {
		soc = func() (float64, error) { return conn.GetFloatState(cc.Soc) }
	}

	return decorateHomeAssistant(m, energy, currents, voltages, soc), nil
}

var _ api.Meter = (*HomeAssistant)(nil)

// CurrentPower implements the api.Meter interface
func (m *HomeAssistant) CurrentPower() (float64, error) {
	return m.conn.GetFloatState(m.power)
}
