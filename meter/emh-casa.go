// provides EMH CASA 1.1 Smart Meter Gateway support
// This is a wrapper around the emh-casa-go library:

package meter

import (
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	emhcasa "github.com/iseeberg79/emh-casa-go"
)

func init() {
	registry.Add("emh-casa", NewEMHCasaFromConfig)
}

// EMHCasa meter wrapper for EMH CASA 1.1 Smart Meter Gateways
type EMHCasa struct {
	client  *emhcasa.Client
	valuesG func() (map[string]float64, error)
	log     *util.Logger
}

// NewEMHCasaFromConfig creates meter from evcc config
func NewEMHCasaFromConfig(other map[string]any) (api.Meter, error) {
	cc := struct {
		URI      string
		User     string
		Password string
		MeterID  string
		Host     string // REQUIRED for most CASA gateways (example: 192.168.33.2)
		Refresh  time.Duration
	}{
		Refresh: 15 * time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.URI == "" {
		return nil, fmt.Errorf("missing uri")
	}

	if cc.User == "" || cc.Password == "" {
		return nil, api.ErrMissingCredentials
	}

	return NewEMHCasa(cc.URI, cc.User, cc.Password, cc.MeterID, cc.Host, cc.Refresh)
}

// NewEMHCasa creates an EMH CASA meter wrapper
func NewEMHCasa(uri, user, password, meterID, host string, refresh time.Duration) (api.Meter, error) {
	log := util.NewLogger("emh-casa").Redact(user, password)

	client, err := emhcasa.NewClient(uri, user, password, meterID, host)
	if err != nil {
		return nil, err
	}

	// Log discovered meter ID
	discoveredID := client.MeterID()
	prefix := discoveredID
	if len(prefix) > 4 {
		prefix = prefix[:4] + "..."
	}
	if meterID == "" {
		log.DEBUG.Printf("discovered meter ID: %s", prefix)
	} else {
		log.DEBUG.Printf("using configured meter ID: %s", prefix)
	}
	log.Redact(discoveredID)

	m := &EMHCasa{
		client: client,
		log:    log,
	}

	// Validate connection
	log.DEBUG.Printf("validating connection...")
	if _, err := m.getMeterValues(); err != nil {
		return nil, fmt.Errorf("failed to validate meter connection: %w", err)
	}
	log.DEBUG.Printf("connection validated successfully")

	m.valuesG = util.Cached(m.getMeterValues, refresh)

	return m, nil
}

// getMeterValues fetches meter values from the external library
func (m *EMHCasa) getMeterValues() (map[string]float64, error) {
	return m.client.GetMeterValues()
}

// CurrentPower implements api.Meter (OBIS 16.7.0 - Active power)
func (m *EMHCasa) CurrentPower() (float64, error) {
	return m.getOBISValue("16.7.0")
}

// TotalEnergy implements api.MeterEnergy (OBIS 1.8.0)
func (m *EMHCasa) TotalEnergy() (float64, error) {
	return m.getOBISValue("1.8.0")
}

// getOBISValue is a DRY helper for OBIS value extraction
func (m *EMHCasa) getOBISValue(obis string) (float64, error) {
	values, err := m.valuesG()
	if err != nil {
		return 0, err
	}

	v, ok := values[obis]
	if !ok {
		return 0, fmt.Errorf("OBIS value (%s) not found", obis)
	}

	return v, nil
}

// Currents implements api.PhaseCurrents
func (m *EMHCasa) Currents() (float64, float64, float64, error) {
	values, err := m.valuesG()
	if err != nil {
		return 0, 0, 0, err
	}

	// Return 0 for missing phases instead of error (gateway may not provide all)
	l1 := values["31.7.0"]
	l2 := values["51.7.0"]
	l3 := values["71.7.0"]

	return l1, l2, l3, nil
}

// Voltages implements api.PhaseVoltages
func (m *EMHCasa) Voltages() (float64, float64, float64, error) {
	values, err := m.valuesG()
	if err != nil {
		return 0, 0, 0, err
	}

	// Return 0 for missing phases instead of error (gateway may not provide all)
	l1 := values["32.7.0"]
	l2 := values["52.7.0"]
	l3 := values["72.7.0"]

	return l1, l2, l3, nil
}

// Powers implements api.PhasePowers
func (m *EMHCasa) Powers() (float64, float64, float64, error) {
	values, err := m.valuesG()
	if err != nil {
		return 0, 0, 0, err
	}

	// Return 0 for missing phases instead of error (gateway may not provide all)
	l1 := values["36.7.0"]
	l2 := values["56.7.0"]
	l3 := values["76.7.0"]

	return l1, l2, l3, nil
}

// GridProduction returns grid feed-in energy (OBIS 2.8.0)
func (m *EMHCasa) GridProduction() (float64, error) {
	return m.getOBISValue("2.8.0")
}

var _ api.Meter = (*EMHCasa)(nil)
var _ api.MeterEnergy = (*EMHCasa)(nil)
var _ api.PhaseCurrents = (*EMHCasa)(nil)
var _ api.PhaseVoltages = (*EMHCasa)(nil)
var _ api.PhasePowers = (*EMHCasa)(nil)
