package fritzdect

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
)

// FRITZ! Smarthome REST API (FritzOS 8.2+)
// https://fritz.support/resources/SmarthomeRestApiFRITZOS82.html

// RestConnection implements the new REST API for Fritz smarthome devices
type RestConnection struct {
	*request.Helper
	*Settings
	SID     string
	UID     string // device UID (AIN with space)
	updated time.Time
	unitG   util.Cacheable[Unit]
}

// Overview response from /smarthome/overview
type Overview struct {
	Units []Unit `json:"units"`
}

// Unit represents a smarthome unit with its interfaces
type Unit struct {
	UID                 string               `json:"UID"`
	DeviceUID           string               `json:"deviceUid"`
	UnitType            string               `json:"unitType"`
	IsConnected         bool                 `json:"isConnected"`
	MultimeterInterface *MultimeterInterface `json:"multimeterInterface,omitempty"`
	OnOffInterface      *OnOffInterface      `json:"onOffInterface,omitempty"`
}

// MultimeterInterface contains power/energy measurements
type MultimeterInterface struct {
	State   string `json:"state"`
	Power   int    `json:"power"`   // mW
	Voltage int    `json:"voltage"` // mV
	Current int    `json:"current"` // mA
	Energy  int    `json:"energy"`  // Wh
}

// OnOffInterface contains switch state
type OnOffInterface struct {
	State string `json:"state"` // "on" or "off"
}

// NewRestConnection creates a new REST API connection
func NewRestConnection(uri, ain, user, password string) (*RestConnection, error) {
	if uri == "" {
		uri = "https://fritz.box"
	}

	if ain == "" {
		return nil, errors.New("missing ain")
	}

	settings := &Settings{
		URI:      strings.TrimRight(uri, "/"),
		AIN:      ain,
		User:     user,
		Password: password,
	}

	log := util.NewLogger("fritzrest").Redact(password)

	conn := &RestConnection{
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

// refreshSession ensures we have a valid session ID
func (c *RestConnection) refreshSession() error {
	if time.Since(c.updated) < sessionTimeout {
		return nil
	}

	// Use the same session mechanism as legacy connection
	uri := fmt.Sprintf("%s/login_sid.lua", c.URI)
	body, err := c.GetBody(uri)
	if err != nil {
		return err
	}

	var v struct {
		SID       string `xml:"SID"`
		Challenge string `xml:"Challenge"`
	}

	if err = parseXML(body, &v); err != nil {
		return err
	}

	if v.SID == "0000000000000000" {
		challresp, err := createChallengeResponse(v.Challenge, c.Password)
		if err != nil {
			return err
		}

		params := url.Values{
			"username": {c.User},
			"response": {challresp},
		}

		body, err = c.GetBody(uri + "?" + params.Encode())
		if err != nil {
			return err
		}

		if err = parseXML(body, &v); err != nil {
			return err
		}

		if v.SID == "0000000000000000" {
			return errors.New("invalid user or password")
		}
	}

	c.SID = v.SID
	c.updated = time.Now()

	return nil
}

// getUnit fetches unit data from REST API
func (c *RestConnection) getUnit() (Unit, error) {
	if err := c.refreshSession(); err != nil {
		return Unit{}, err
	}

	// Try to get the specific unit first
	uri := fmt.Sprintf("%s/smarthome/units/%s?sid=%s", c.URI, url.PathEscape(c.UID), c.SID)

	var unit Unit
	err := c.GetJSON(uri, &unit)
	if err != nil {
		// Fall back to getting all units and finding ours
		return c.findUnit()
	}

	if !unit.IsConnected {
		return unit, api.ErrNotAvailable
	}

	return unit, nil
}

// findUnit searches for our unit in the overview
func (c *RestConnection) findUnit() (Unit, error) {
	uri := fmt.Sprintf("%s/smarthome/units?sid=%s", c.URI, c.SID)

	var units []Unit
	if err := c.GetJSON(uri, &units); err != nil {
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
func (c *RestConnection) CurrentPower() (float64, error) {
	unit, err := c.unitG.Get()
	if err != nil {
		return 0, err
	}

	if unit.MultimeterInterface == nil {
		return 0, errors.New("device has no power meter")
	}

	if unit.MultimeterInterface.State != "valid" {
		return 0, api.ErrNotAvailable
	}

	// Power is in mW, convert to W
	return float64(unit.MultimeterInterface.Power) / 1000, nil
}

var _ api.MeterEnergy = (*RestConnection)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (c *RestConnection) TotalEnergy() (float64, error) {
	unit, err := c.unitG.Get()
	if err != nil {
		return 0, err
	}

	if unit.MultimeterInterface == nil {
		return 0, errors.New("device has no energy meter")
	}

	// Energy is in Wh, convert to kWh
	return float64(unit.MultimeterInterface.Energy) / 1000, nil
}

// SwitchPresent checks if the device is connected
func (c *RestConnection) SwitchPresent() (bool, error) {
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
func (c *RestConnection) SwitchState() (bool, error) {
	unit, err := c.unitG.Get()
	if err != nil {
		return false, err
	}

	if unit.OnOffInterface == nil {
		return false, errors.New("device has no switch")
	}

	return unit.OnOffInterface.State == "on", nil
}

// SwitchOn turns the switch on
func (c *RestConnection) SwitchOn() error {
	return c.setSwitch(true)
}

// SwitchOff turns the switch off
func (c *RestConnection) SwitchOff() error {
	return c.setSwitch(false)
}

// setSwitch sets the switch state via REST API
func (c *RestConnection) setSwitch(on bool) error {
	if err := c.refreshSession(); err != nil {
		return err
	}

	state := "off"
	if on {
		state = "on"
	}

	uri := fmt.Sprintf("%s/smarthome/units/%s?sid=%s", c.URI, url.PathEscape(c.UID), c.SID)

	data := map[string]any{
		"onOffInterface": map[string]string{
			"state": state,
		},
	}

	req, err := request.New("PUT", uri, request.MarshalJSON(data), request.JSONEncoding)
	if err != nil {
		return err
	}

	var unit Unit
	if err := c.DoJSON(req, &unit); err != nil {
		return err
	}

	// Reset cache after state change
	c.unitG.Reset()

	// Verify state was changed
	if unit.OnOffInterface != nil {
		actualState := unit.OnOffInterface.State == "on"
		if actualState != on {
			return errors.New("switch state change failed")
		}
	}

	return nil
}
