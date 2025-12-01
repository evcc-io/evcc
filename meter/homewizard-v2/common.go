package homewizard

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/mluiten/evcc-homewizard-v2/device"
)

// DefaultTimeout is the default connection timeout
const DefaultTimeout = device.DefaultTimeout

// Config holds common configuration for all HomeWizard meters
type Config struct {
	Host    string
	Token   string
	Usage   string
	Timeout time.Duration
}

// HomeWizardMeter is a minimal base that only stores common fields
type HomeWizardMeter struct {
	log    *util.Logger
	phases int // 1 or 3
}

var _ api.PhaseGetter = (*HomeWizardMeter)(nil)

// GetPhases implements the api.PhaseGetter interface
func (m *HomeWizardMeter) GetPhases() (int, error) {
	return m.phases, nil
}
