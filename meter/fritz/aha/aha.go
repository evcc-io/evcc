package aha

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/meter/fritz"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
)

// FRITZ! FritzBox AHA interface and authentication specifications:
// https://fritz.com/fileadmin/user_upload/Global/Service/Schnittstellen/AHA-HTTP-Interface.pdf
// https://fritz.com/fileadmin/user_upload/Global/Service/Schnittstellen/AVM_Technical_Note_-_Session_ID.pdf

// FritzDECT connection
type Connection struct {
	*request.Helper
	*fritz.Settings
	SID     string
	updated time.Time
}

// NewConnection creates FritzDECT connection
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

	log := util.NewLogger("fritzdect").Redact(password)

	fritzdect := &Connection{
		Helper:   request.NewHelper(log),
		Settings: settings,
	}

	fritzdect.Client.Transport = request.NewTripper(log, transport.Insecure())

	return fritzdect, nil
}

// ExecCmd execautes an FritzDECT AHA-HTTP-Interface command
func (c *Connection) ExecCmd(function string) (string, error) {
	// refresh Fritzbox session id
	if time.Since(c.updated) >= fritz.SessionTimeout {
		sid, err := c.GetSessionID(c.Helper)
		if err != nil {
			return "", err
		}
		// update session timestamp
		c.SID = sid
		c.updated = time.Now()
	}

	parameters := url.Values{
		"sid":       {c.SID},
		"ain":       {c.AIN},
		"switchcmd": {function},
	}

	uri := fmt.Sprintf("%s/webservices/homeautoswitch.lua", c.URI)
	body, err := c.GetBody(uri + "?" + parameters.Encode())

	res := strings.TrimSpace(string(body))

	if err == nil && res == "inval" {
		err = api.ErrNotAvailable
	}

	return res, err
}

// CurrentPower implements the api.Meter interface
func (c *Connection) CurrentPower() (float64, error) {
	// power value in 0,001 W (current switch power, refresh approximately every 2 minutes)
	resp, err := c.ExecCmd("getswitchpower")
	if err != nil {
		return 0, err
	}

	power, err := strconv.ParseFloat(resp, 64)

	return power / 1000, err // mW ==> W
}

var _ api.MeterEnergy = (*Connection)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (c *Connection) TotalEnergy() (float64, error) {
	// Energy value in Wh (total switch energy, refresh approximately every 2 minutes)
	resp, err := c.ExecCmd("getswitchenergy")
	if err != nil {
		return 0, err
	}

	energy, err := strconv.ParseFloat(resp, 64)

	return energy / 1000, err // Wh ==> KWh
}

// SwitchPresent checks if the device is connected
func (c *Connection) SwitchPresent() (bool, error) {
	resp, err := c.ExecCmd("getswitchpresent")
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(resp)
}

// SwitchState returns the current switch state
func (c *Connection) SwitchState() (bool, error) {
	resp, err := c.ExecCmd("getswitchstate")
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(resp)
}

// SwitchOn turns the switch on
func (c *Connection) SwitchOn() error {
	resp, err := c.ExecCmd("setswitchon")
	if err != nil {
		return err
	}

	on, err := strconv.ParseBool(resp)
	if err == nil && !on {
		err = errors.New("switch on failed")
	}
	return err
}

// SwitchOff turns the switch off
func (c *Connection) SwitchOff() error {
	resp, err := c.ExecCmd("setswitchoff")
	if err != nil {
		return err
	}

	off, err := strconv.ParseBool(resp)
	if err == nil && off {
		err = errors.New("switch off failed")
	}
	return err
}
