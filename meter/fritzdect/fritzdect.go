package fritzdect

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"golang.org/x/text/encoding/unicode"
)

// NewConnection creates FritzDECT connection
func NewConnection(uri, ain, user, password string) (*Connection, error) {
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
	if time.Since(c.updated) >= sessionTimeout {
		if err := c.getSessionID(); err != nil {
			return "", err
		}
		// update session timestamp
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
		// new logic for Fritz 250
		resp, err := c.ExecCmd("getbasicdevicestats")
		if err != nil {
			return 0, err
		}

		power, err := ParseFXml(resp, err)
		if err != nil {
			return 0, err
		}
		return (power * 10) / 1000, err // 1/100W ==> W
	}

	power, err := strconv.ParseFloat(resp, 64)

	return power / 1000, err // mW ==> W
}

var _ api.MeterEnergy = (*Connection)(nil)

// CurrentPower implements the api.MeterEnergy interface
func (c *Connection) TotalEnergy() (float64, error) {
	// Energy value in Wh (total switch energy, refresh approximately every 2 minutes)
	resp, err := c.ExecCmd("getswitchenergy")
	if err != nil {
		resp, err := c.ExecCmd("getbasicdevicestats")
		if err != nil {
			return 0, err
		}

		energy, err := ParseFXml2(resp, err)
		if err != nil {
			return 0, err
		}
		return energy / 1000, err // Wh ==> KWh
	}

	energy, err := strconv.ParseFloat(resp, 64)

	return energy / 1000, err // Wh ==> KWh
}

func ParseFXml(s string, err error) (float64, error) {
	var v Devicestats

	err2 := xml.Unmarshal([]byte(s), &v)
	if err2 != nil {
		return 0, err2
	}

	var csv = v.Power.Values[0]

	parts := strings.Split(csv, ",")
	if len(parts) == 0 {
		//
	}

	f, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0, err
	}

	return float64(f), nil
}

func ParseFXml2(s string, err error) (float64, error) {
	var v Devicestats

	err2 := xml.Unmarshal([]byte(s), &v)
	if err2 != nil {
		//
	}

	//fmt.Sprintln("%v", v)

	var csv = v.Energy.Values[0]

	parts := strings.Split(csv, ",")
	if len(parts) == 0 {
		//
	}

	f, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0, err
	}

	return float64(f), nil
}

// Fritzbox helpers (credits to https://github.com/rsdk/ahago)

// getSessionID fetches a session-id based on the username and password in the connection struct
func (c *Connection) getSessionID() error {
	uri := fmt.Sprintf("%s/login_sid.lua", c.URI)
	body, err := c.GetBody(uri)
	if err != nil {
		return err
	}

	var v struct {
		SID       string
		Challenge string
		BlockTime string
	}

	if err = xml.Unmarshal(body, &v); err == nil && v.SID == "0000000000000000" {
		var challresp string
		if challresp, err = createChallengeResponse(v.Challenge, c.Password); err == nil {
			params := url.Values{
				"username": {c.User},
				"response": {challresp},
			}

			if body, err = c.GetBody(uri + "?" + params.Encode()); err == nil {
				err = xml.Unmarshal(body, &v)
				if v.SID == "0000000000000000" {
					return errors.New("invalid user or password")
				}
				c.SID = v.SID
			}
		}
	}

	return err
}

// createChallengeResponse creates the Fritzbox challenge response string
func createChallengeResponse(challenge, pass string) (string, error) {
	encoder := unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewEncoder()
	utf16le, err := encoder.String(challenge + "-" + pass)
	if err != nil {
		return "", err
	}

	hash := md5.Sum([]byte(utf16le))
	md5hash := hex.EncodeToString(hash[:])

	return challenge + "-" + md5hash, nil
}
