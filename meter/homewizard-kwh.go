package meter

import (
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/mluiten/evcc-homewizard-v2/device"
)

func init() {
	registry.Add("homewizard-kwh", NewHomeWizardKWHFromConfig)
}

// HomeWizardKWH is a wrapper for kWh meters using the common HomeWizardMeter base
type HomeWizardKWH struct {
	*HomeWizardMeter
}

func NewHomeWizardKWHFromConfig(other map[string]any) (api.Meter, error) {
	cc := struct {
		Host    string
		Token   string
		Usage   string // "pv" or "grid"
		Phases  int    // 1 or 3
		Timeout time.Duration
	}{
		Usage:   "pv",
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

	log := util.NewLogger("homewizard-kwh")
	kwhDevice := device.NewKWHMeterDevice(cc.Host, cc.Token, cc.Timeout)

	// Start device connection and wait for it to succeed
	if err := kwhDevice.StartAndWait(cc.Timeout); err != nil {
		return nil, err
	}

	log.INFO.Printf("configured kWh meter at %s (%d-phase, usage=%s)", cc.Host, cc.Phases, cc.Usage)

	m := &HomeWizardKWH{
		HomeWizardMeter: &HomeWizardMeter{
			log:    log,
			device: kwhDevice,
			usage:  cc.Usage,
			phases: cc.Phases,
		},
	}

	return m, nil
}
