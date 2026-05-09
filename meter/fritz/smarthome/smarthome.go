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
	UID   string // unitUid resolved from /devices
	unitG util.Cacheable[Unit]
}

// NewConnection creates a new REST API connection
func NewConnection(uri, ain, user, password string, unit int) (*Connection, error) {
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
		Unit:     unit,
	}

	log := util.NewLogger("fritzsmarthome").Redact(password)

	conn := &Connection{
		Helper:   request.NewHelper(log),
		Settings: settings,
	}

	conn.Client.Transport = request.NewTripper(log, transport.Insecure())

	uid, err := conn.resolveUnitUID(unit)
	if err != nil {
		return nil, err
	}
	conn.UID = uid

	// cache unit data for 2 seconds to avoid excessive API calls
	conn.unitG = util.ResettableCached(func() (Unit, error) {
		return conn.getUnit()
	}, 2*time.Second)

	return conn, nil
}

// resolveUnitUID looks up the device by AIN and returns its unitUid at the given index
func (c *Connection) resolveUnitUID(unit int) (string, error) {
	sid, err := c.GetSessionID(c.Helper)
	if err != nil {
		return "", err
	}

	uri := fmt.Sprintf("%s/api/v0/smarthome/overview/devices", c.URI)

	req, _ := request.New("GET", uri, nil, map[string]string{
		"Authorization": "AVM-SID " + sid,
	}, request.AcceptJSON)

	var devices []Device
	if err := c.DoJSON(req, &devices); err != nil {
		return "", err
	}

	for _, d := range devices {
		if d.AIN != c.AIN {
			continue
		}
		if len(d.UnitUids) < unit {
			return "", fmt.Errorf("invalid unit %d, got %v", unit, d.UnitUids)
		}
		return d.UnitUids[unit-1], nil
	}

	return "", fmt.Errorf("ain not found: %s", c.AIN)
}

// getUnit fetches unit data from REST API
func (c *Connection) getUnit() (Unit, error) {
	sid, err := c.GetSessionID(c.Helper)
	if err != nil {
		return Unit{}, err
	}

	uri := fmt.Sprintf("%s/api/v0/smarthome/overview/units/%s", c.URI, url.PathEscape(c.UID))

	req, _ := request.New("GET", uri, nil, map[string]string{
		"Authorization": "AVM-SID " + sid,
	}, request.AcceptJSON)

	var unit Unit
	if err := c.DoJSON(req, &unit); err != nil {
		return Unit{}, err
	}

	if !unit.IsConnected {
		return unit, api.ErrNotAvailable
	}

	return unit, nil
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

var _ api.MeterImport = (*Connection)(nil)

// ImportEnergy implements the api.MeterImport interface
func (c *Connection) ImportEnergy() (float64, error) {
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
	sid, err := c.GetSessionID(c.Helper)
	if err != nil {
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
		"Authorization": "AVM-SID " + sid,
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
