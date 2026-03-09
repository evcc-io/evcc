package chargercontroller

import (
	"fmt"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

// PowerController controls power-native chargers (heatpumps, water heaters, SG Ready)
// that implement api.PowerController. It sends watts directly without current/phase conversion.
type PowerController struct {
	log     *util.Logger
	clock   clock.Clock
	charger api.Charger
	power   api.PowerController
	limiter api.PowerLimiter // optional
	circuit api.Circuit      // optional

	offeredPower    float64
	chargePower     float64 // measured charge power
	enabled         bool
	chargerSwitched time.Time

	// callbacks
	setEnabled func(bool)
	publish    func(key string, val any)
}

// NewPowerController creates a PowerController for a charger implementing api.PowerController.
func NewPowerController(log *util.Logger, clock clock.Clock, charger api.Charger, power api.PowerController, circuit api.Circuit, setEnabled func(bool), publish func(string, any)) *PowerController {
	c := &PowerController{
		log:        log,
		clock:      clock,
		charger:    charger,
		power:      power,
		circuit:    circuit,
		setEnabled: setEnabled,
		publish:    publish,
	}

	if limiter, ok := charger.(api.PowerLimiter); ok {
		c.limiter = limiter
	}

	return c
}

func (c *PowerController) MinPower() float64 {
	if c.limiter != nil {
		if minP, _, err := c.limiter.GetMinMaxPower(); err == nil && minP > 0 {
			return minP
		}
	}
	return 0
}

func (c *PowerController) MaxPower() float64 {
	if c.limiter != nil {
		if _, maxP, err := c.limiter.GetMinMaxPower(); err == nil && maxP > 0 {
			return maxP
		}
	}
	return 0
}

func (c *PowerController) EffectiveChargePower() float64 {
	return c.chargePower
}

// SyncState synchronizes the controller's enabled state with the initial charger state.
func (c *PowerController) SyncState(enabled bool) {
	c.enabled = enabled
}


// UpdateChargePower updates the measured charge power from the meter.
func (c *PowerController) UpdateChargePower(power float64) {
	c.chargePower = power
}

func (c *PowerController) SetMaxPower() error {
	return c.SetOfferedPower(c.MaxPower())
}

func (c *PowerController) SetOfferedPower(power float64) error {
	// circuit power limits
	if c.circuit != nil {
		power = c.circuit.ValidatePower(c.chargePower, power)
	}

	effMinPower := c.MinPower()
	if effMaxPower := c.MaxPower(); effMinPower > effMaxPower && effMaxPower > 0 {
		return fmt.Errorf("invalid config: min power %.0fW exceeds max power %.0fW", effMinPower, effMaxPower)
	}

	// set power on charger
	if power != c.offeredPower && power >= effMinPower {
		if err := c.power.MaxPower(power); err != nil {
			return fmt.Errorf("set charge power limit %.0fW: %w", power, err)
		}

		c.log.DEBUG.Printf("set charge power limit: %.0fW", power)
		c.offeredPower = power
	}

	// enable/disable
	if enabled := power >= effMinPower; enabled != c.enabled {
		if err := c.charger.Enable(enabled); err != nil {
			return fmt.Errorf("charger %s: %w", enabledStatus[enabled], err)
		}

		c.enabled = enabled
		c.setEnabled(enabled)
		c.chargerSwitched = c.clock.Now()

		if !enabled {
			c.offeredPower = 0
		}
	}

	return nil
}
