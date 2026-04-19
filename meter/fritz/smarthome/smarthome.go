package smarthome

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/meter/fritz"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
)

// FRITZ! Smarthome REST API (FritzOS 8.2+)
// https://fritz.support/resources/SmarthomeRestApiFRITZOS82.html

// Connection implements the new REST API for Fritz smarthome devices
type Connection struct {
	*request.Helper
	*fritz.Settings
	SID     string
	UID     string // device UID (AIN with space)
	updated time.Time
	unitG   util.Cacheable[Unit]
}

// NewConnection creates a new REST API connection
func NewConnection(uri, ain, user, password string) (*Connection, error) {
	if uri == "" {
		uri = "https://fritz.box"
	}

	if ain == "" {
		return nil, errors.New("missing ain")
	}

	settings := &fritz.Settings{
		URI:      strings.TrimRight(uri, "/"),
		AIN:      ain,
		User:     user,
		Password: password,
	}

	log := util.NewLogger("fritzsmarthome").Redact(password)

	conn := &Connection{
		Helper:   request.NewHelper(log),
		Settings: settings,
		UID:      ainToUID(ain),
	}

	conn.Client.Transport = request.NewTripper(log, transport.Insecure())

	// cache unit data for 2 seconds to avoid excessive API calls
	conn.unitG = util.ResettableCached(func() (Unit, error) {
		return conn.getUnit()
	}, 2*time.Second)

	return conn, nil
}

// ainToUID converts AIN format to UID format by adding space
// AIN: "116300015376" -> UID: "11630 0015376"
func ainToUID(ain string) string {
	// Remove any existing spaces first
	ain = strings.ReplaceAll(ain, " ", "")
	if len(ain) >= 5 {
		return ain[:5] + " " + ain[5:]
	}
	return ain
}

// getUnit fetches unit data from REST API
func (c *Connection) getUnit() (Unit, error) {
	if err := c.refreshSession(); err != nil {
		return Unit{}, err
	}

	// Try to get the specific unit first
	uri := fmt.Sprintf("%s/api/v0/smarthome/overview/units/%s", c.URI, url.PathEscape(c.UID))

	req, _ := request.New("GET", uri, nil, map[string]string{
		"Authorization": "AVM-SID " + c.SID,
	}, request.AcceptJSON)

	var unit Unit
	if err := c.DoJSON(req, &unit); err != nil {
		// Fall back to getting all units and finding ours
		return c.findUnit()
	}

	if !unit.IsConnected {
		return unit, api.ErrNotAvailable
	}

	return unit, nil
}

// findUnit searches for our unit in the list of all units
func (c *Connection) findUnit() (Unit, error) {
	uri := fmt.Sprintf("%s/api/v0/smarthome/overview/units", c.URI)

	req, err := request.New("GET", uri, nil, map[string]string{
		"Authorization": "AVM-SID " + c.SID,
	}, request.AcceptJSON)
	if err != nil {
		return Unit{}, err
	}

	var units []Unit
	if err := c.DoJSON(req, &units); err != nil {
		return Unit{}, err
	}

	// Search for matching unit by UID or AIN
	for _, unit := range units {
		unitAIN := strings.ReplaceAll(unit.UID, " ", "")
		if unit.UID == c.UID || unitAIN == c.AIN {
			if !unit.IsConnected {
				return unit, api.ErrNotAvailable
			}
			return unit, nil
		}
	}

	return Unit{}, fmt.Errorf("unit not found: %s", c.AIN)
}

// CurrentPower implements the api.Meter interface
func (c *Connection) CurrentPower() (float64, error) {
	unit, err := c.unitG.Get()
	if err != nil {
		return 0, err
	}

	if unit.Statistics != nil && len(unit.Statistics.Powers) > 0 {
		if stats := unit.Statistics.Powers[0].Values; len(stats) > 0 {
			return (float64)(stats[0]) / 1000, nil
		}
	}

	return 0, api.ErrNotAvailable
}

var _ api.MeterEnergy = (*Connection)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (c *Connection) TotalEnergy() (float64, error) {
	unit, err := c.unitG.Get()
	if err != nil {
		return 0, err
	}

	if unit.Statistics != nil && len(unit.Statistics.Energies) > 0 {
		if stats := unit.Statistics.Energies[len(unit.Statistics.Energies)-1].Values; len(stats) > 0 {
			return (float64)(stats[0]) / 1000, nil
		}
	}

	return 0, api.ErrNotAvailable
}

// SwitchPresent checks if the device is connected
func (c *Connection) SwitchPresent() (bool, error) {
	unit, err := c.unitG.Get()
	if err != nil {
		if errors.Is(err, api.ErrNotAvailable) {
			return false, nil
		}
		return false, err
	}
	return unit.IsConnected, nil
}

// SwitchState returns the current switch state
func (c *Connection) SwitchState() (bool, error) {
	unit, err := c.unitG.Get()
	if err != nil {
		return false, err
	}

	if unit.Interfaces.OnOffInterface == nil {
		return false, errors.New("device has no switch")
	}

	return unit.Interfaces.OnOffInterface.State == "on", nil
}

// SwitchOn turns the switch on
func (c *Connection) SwitchOn() error {
	return c.setSwitch(true)
}

// SwitchOff turns the switch off
func (c *Connection) SwitchOff() error {
	return c.setSwitch(false)
}

// setSwitch sets the switch state via REST API
func (c *Connection) setSwitch(on bool) error {
	if err := c.refreshSession(); err != nil {
		return err
	}

	state := "off"
	if on {
		state = "on"
	}

	uri := fmt.Sprintf("%s/api/v0/smarthome/overview/units/%s", c.URI, url.PathEscape(c.UID))

	data := map[string]any{
		"onOffInterface": map[string]string{
			"state": state,
		},
	}

	req, _ := request.New("PUT", uri, request.MarshalJSON(data), map[string]string{
		"Authorization": "AVM-SID " + c.SID,
	}, request.JSONEncoding)

	var unit Unit
	if err := c.DoJSON(req, &unit); err != nil {
		return err
	}

	// Reset cache after state change
	c.unitG.Reset()

	// Verify state was changed
	if unit.Interfaces.OnOffInterface != nil {
		actualState := unit.Interfaces.OnOffInterface.State == "on"
		if actualState != on {
			return errors.New("switch state change failed")
		}
	}

	return nil
}

// refreshSession ensures we have a valid session ID
func (c *Connection) refreshSession() error {
	// refresh Fritzbox session id
	if time.Since(c.updated) >= fritz.SessionTimeout {
		sid, err := c.GetSessionID(c.Helper)
		if err != nil {
			return err
		}
		// update session timestamp
		c.SID = sid
		c.updated = time.Now()
	}

	return nil
}
