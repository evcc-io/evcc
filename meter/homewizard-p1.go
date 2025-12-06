package meter

import (
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/mluiten/evcc-homewizard-v2/device"
)

func init() {
	registry.Add("homewizard-p1", NewHomeWizardP1FromConfig)
}

// HomeWizardP1 is a wrapper for P1 meters using the common HomeWizardMeter base
type HomeWizardP1 struct {
	*HomeWizardMeter
	p1MeterDevice *device.P1MeterDevice // Keep reference for battery control
}

func NewHomeWizardP1FromConfig(other map[string]any) (api.Meter, error) {
	cc := struct {
		Host    string
		Token   string
		Phases  int // 1 or 3
		Timeout time.Duration
	}{
		Phases:  3,
		Timeout: device.DefaultTimeout,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	// Validate required parameters
	if cc.Host == "" || cc.Token == "" {
		return nil, fmt.Errorf("missing host or token - run 'evcc token homewizard'")
	}

	// Validate phases
	if cc.Phases != 1 && cc.Phases != 3 {
		return nil, fmt.Errorf("invalid phases value %d: must be 1 or 3", cc.Phases)
	}

	log := util.NewLogger("homewizard-p1")

	// Create P1MeterDevice (includes battery control)
	p1MeterDevice := device.NewP1MeterDevice(cc.Host, cc.Token, cc.Timeout)

	// Start device connection and wait for it to succeed
	if err := p1MeterDevice.StartAndWait(cc.Timeout); err != nil {
		return nil, err
	}

	log.INFO.Printf("configured P1 meter at %s (%d-phase)", cc.Host, cc.Phases)

	m := &HomeWizardP1{
		HomeWizardMeter: &HomeWizardMeter{
			log:    log,
			device: p1MeterDevice,
			usage:  "grid", // P1 meters are always grid meters
			phases: cc.Phases,
		},
		p1MeterDevice: p1MeterDevice,
	}

	return m, nil
}

// SetBatteryMode sets battery mode via P1 meter (for battery controller)
func (m *HomeWizardP1) SetBatteryMode(mode string) error {
	return m.p1MeterDevice.SetBatteryMode(mode)
}
