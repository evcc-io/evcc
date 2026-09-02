package shelly

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
)

type Generation interface {
	Enabled() (bool, error)
	Enable(bool) error
	// relay is the switch id the channel resolves to, which may differ from the configured one
	relay() int
	api.Meter
	api.MeterEnergy
	api.MeterReturnEnergy
	IsThreePhase() bool
	IsReversed() bool
	HasReturnEnergy() bool
}

type Phases interface {
	api.PhaseCurrents
	api.PhaseVoltages
	api.PhasePowers
}

// Connection is the Shelly connection. It aggregates one Generation per configured channel.
type Connection struct {
	Generation // first channel, decides the device capabilities
	gens       []Generation
	gen        int
}

var _ Phases = (*Connection)(nil)

// SignedPower reports whether the device returns directional (signed) power.
func (c *Connection) SignedPower() bool {
	return c.gen >= 3
}

// NewConnection creates a new Shelly device connection.
func NewConnection(uri, user, password string, channels []int, cache time.Duration) (*Connection, error) {
	if uri == "" {
		return nil, errors.New("missing uri")
	}

	if len(channels) == 0 {
		return nil, errors.New("missing channel")
	}

	for _, suffix := range []string{"/", "/rcp", "/shelly"} {
		uri = strings.TrimSuffix(uri, suffix)
	}
	uri = util.DefaultScheme(uri, "http")

	log := util.NewLogger("shelly")
	client := request.NewHelper(log)

	// Shelly Gen1 and Gen2 families expose the /shelly endpoint
	var resp DeviceInfo
	if err := client.GetJSON(fmt.Sprintf("%s/shelly", uri), &resp); err != nil {
		return nil, err
	}

	if (resp.Auth || resp.AuthEn) && (user == "" || password == "") {
		return nil, fmt.Errorf("%s (%s) missing user/password", resp.Model, resp.Mac)
	}

	model, _, _ := strings.Cut(resp.Type+resp.Model, "-")

	client.Transport = request.NewTripper(log, transport.Insecure())

	// authentication is wrapped once, the client is shared by all channels
	if user != "" {
		if resp.Gen < 2 {
			// https://shelly-api-docs.shelly.cloud/gen1/#authentication
			log.Redact(transport.BasicAuthHeader(user, password))
			client.Transport = transport.BasicAuth(user, password, client.Transport)
		} else {
			// Shelly gen 2 rfc7616 authentication
			// https://shelly-api-docs.shelly.cloud/gen2/General/Authentication
			client.Transport = transport.Digest(user, password, client.Transport)
		}
	}

	gens := make([]Generation, 0, len(channels))
	for _, channel := range channels {
		if resp.Gen < 2 {
			// Shelly GEN 1 API
			// https://shelly-api-docs.shelly.cloud/gen1/#shelly-family-overview
			gens = append(gens, newGen1(client, uri, model, channel, cache))
			continue
		}

		// Shelly GEN 2+ API
		// https://shelly-api-docs.shelly.cloud/gen2/
		gen, err := newGen2(client, uri, model, channel, cache)
		if err != nil {
			return nil, err
		}
		gens = append(gens, gen)
	}

	// channels are compared after construction- the Pro output add-on can remap
	// multiple configured channels onto the same relay
	used := make(map[int]bool, len(gens))
	for _, gen := range gens {
		if used[gen.relay()] {
			return nil, fmt.Errorf("duplicate channel: %d", gen.relay())
		}
		used[gen.relay()] = true
	}

	conn := &Connection{Generation: gens[0], gens: gens, gen: resp.Gen}

	return conn, nil
}

// Enabled reports whether all configured channels are switched on
func (c *Connection) Enabled() (bool, error) {
	for _, gen := range c.gens {
		enabled, err := gen.Enabled()
		if err != nil || !enabled {
			return false, err
		}
	}
	return true, nil
}

// Enable switches all configured channels. Every channel is attempted, a channel
// left in the wrong state would otherwise go unnoticed by the Enabled readback.
func (c *Connection) Enable(enable bool) error {
	var errs []error
	for _, gen := range c.gens {
		if err := gen.Enable(enable); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// CurrentPower implements the api.Meter interface
func (c *Connection) CurrentPower() (float64, error) {
	return c.sum(Generation.CurrentPower)
}

// TotalEnergy implements the api.MeterEnergy interface
func (c *Connection) TotalEnergy() (float64, error) {
	return c.sum(Generation.TotalEnergy)
}

// ReturnEnergy implements the api.MeterReturnEnergy interface
func (c *Connection) ReturnEnergy() (float64, error) {
	return c.sum(Generation.ReturnEnergy)
}

// sum accumulates a reading across all configured channels
func (c *Connection) sum(fun func(Generation) (float64, error)) (float64, error) {
	var total float64
	for _, gen := range c.gens {
		val, err := fun(gen)
		if err != nil {
			return 0, err
		}
		total += val
	}
	return total, nil
}

// HasPhases reports whether per-phase readings are available. A single channel uses the
// device readings, three channels are mapped to L1..L3 in configuration order.
func (c *Connection) HasPhases() bool {
	_, ok := c.Generation.(Phases)
	return ok && (len(c.gens) == 1 || len(c.gens) == 3)
}

// Currents implements the api.PhaseCurrents interface
func (c *Connection) Currents() (float64, float64, float64, error) {
	return c.phaseValues(Phases.Currents)
}

// Voltages implements the api.PhaseVoltages interface
func (c *Connection) Voltages() (float64, float64, float64, error) {
	return c.phaseValues(Phases.Voltages)
}

// Powers implements the api.PhasePowers interface
func (c *Connection) Powers() (float64, float64, float64, error) {
	return c.phaseValues(Phases.Powers)
}

func (c *Connection) phaseValues(fun func(Phases) (float64, float64, float64, error)) (float64, float64, float64, error) {
	if !c.HasPhases() {
		return 0, 0, 0, api.ErrNotAvailable
	}

	if len(c.gens) == 1 {
		return fun(c.Generation.(Phases))
	}

	var res [3]float64
	for i, gen := range c.gens {
		l1, _, _, err := fun(gen.(Phases))
		if err != nil {
			return 0, 0, 0, err
		}
		res[i] = l1
	}

	return res[0], res[1], res[2], nil
}
