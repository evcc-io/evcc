package charger

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

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/request"
	"golang.org/x/text/encoding/unicode"
)

// AVM FritzBox AHA interface and authentification specifications:
// https://avm.de/fileadmin/user_upload/Global/Service/Schnittstellen/AHA-HTTP-Interface.pdf
// https://avm.de/fileadmin/user_upload/Global/Service/Schnittstellen/AVM_Technical_Note_-_Session_ID.pdf

// FritzDECT charger implementation
type FritzDECT struct {
	*request.Helper
	uri, ain, user, password, sid string
	standbypower                  float64
	updated                       time.Time
}

func init() {
	registry.Add("fritzdect", NewFritzDECTFromConfig)
}

// NewFritzDECTFromConfig creates a fritzdect charger from generic config
func NewFritzDECTFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		URI          string
		AIN          string
		User         string
		Password     string
		SID          string
		StandbyPower float64
		Updated      time.Time
	}{}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.URI == "" {
		cc.URI = "https://fritz.box"
	}

	if cc.AIN == "" {
		return nil, errors.New("missing ain")
	}

	return NewFritzDECT(cc.URI, cc.AIN, cc.User, cc.Password, cc.SID, cc.StandbyPower, cc.Updated)
}

// NewFritzDECT creates FritzDECT charger
func NewFritzDECT(uri, ain, user, password, sid string, standbypower float64, updated time.Time) (*FritzDECT, error) {
	log := util.NewLogger("fritzdect")

	c := &FritzDECT{
		Helper:       request.NewHelper(log),
		uri:          strings.TrimRight(uri, "/"),
		ain:          ain,
		user:         user,
		password:     password,
		standbypower: standbypower,
		sid:          sid,
	}

	c.Client.Transport = request.NewTripper(log, request.InsecureTransport())

	return c, nil
}

func (c *FritzDECT) execFritzDectCmd(function string) (string, error) {
	// Refresh Fritzbox session id
	if time.Since(c.updated).Minutes() >= 10 {
		err := c.getSessionID()
		if err != nil {
			return "", err
		}
		// Update session timestamp
		c.updated = time.Now()
	}

	parameters := url.Values{
		"sid":       []string{c.sid},
		"ain":       []string{c.ain},
		"switchcmd": []string{function},
	}

	uri := fmt.Sprintf("%s/webservices/homeautoswitch.lua", c.uri)
	response, err := c.GetBody(uri + "?" + parameters.Encode())
	return strings.TrimSpace(string(response)), err
}

// Status implements the Charger.Status interface
func (c *FritzDECT) Status() (api.ChargeStatus, error) {

	// present 0/1 - DECT Switch connected to fritzbox (no/yes)
	var present int64
	resp, err := c.execFritzDectCmd("getswitchpresent")
	if err == nil {
		present, err = strconv.ParseInt(resp, 10, 64)
	}

	// power value in 0,001 W (current switch power, refresh aproximately every 2 minutes)
	var power float64
	if err == nil {
		if resp, err = c.execFritzDectCmd("getswitchpower"); err == nil {
			power, err = strconv.ParseFloat(resp, 64)
		}
	}

	power = power / 1000 // mW ==> W
	switch {
	case present == 1 && power <= c.standbypower:
		return api.StatusB, err
	case present == 1 && power > c.standbypower:
		return api.StatusC, err
	default:
		return api.StatusNone, errors.New("switch absent")
	}
}

// Enabled implements the Charger.Enabled interface
func (c *FritzDECT) Enabled() (bool, error) {
	// state 0/1 - DECT Switch state off/on (empty if unkown or error)
	resp, err := c.execFritzDectCmd("getswitchstate")

	var state int64
	if err == nil {
		state, err = strconv.ParseInt(resp, 10, 32)
	}

	return state == 1, err
}

// Enable implements the Charger.Enable interface
func (c *FritzDECT) Enable(enable bool) error {
	cmd := "setswitchoff"
	if enable {
		cmd = "setswitchon"
	}

	// state 0/1 - DECT Switch state off/on (empty if unkown or error)
	resp, err := c.execFritzDectCmd(cmd)

	var state int64
	if err == nil {
		state, err = strconv.ParseInt(resp, 10, 32)
	}

	switch {
	case err != nil:
		return err
	case enable && state == 0:
		return errors.New("switchOn failed")
	case !enable && state == 1:
		return errors.New("switchOff failed")
	default:
		return nil
	}
}

// MaxCurrent implements the Charger.MaxCurrent interface
func (c *FritzDECT) MaxCurrent(current int64) error {
	return nil
}

// CurrentPower implements the Meter interface.
func (c *FritzDECT) CurrentPower() (float64, error) {
	// power value in 0,001 W (current switch power, refresh aproximately every 2 minutes)
	resp, err := c.execFritzDectCmd("getswitchpower")

	var power float64
	if err == nil {
		power, err = strconv.ParseFloat(resp, 64)
	}

	// ignore standby power
	power = power / 1000 // mW ==> W
	if power < c.standbypower {
		power = 0
	}

	return power, err
}

// Fritzbox helpers (based on ideas of https://github.com/rsdk/ahago)

// getSessionID fetches a session-id based on the username and password in the connection struct
func (c *FritzDECT) getSessionID() error {
	uri := fmt.Sprintf("%s/login_sid.lua", c.uri)
	body, err := c.GetBody(uri)
	if err != nil {
		return err
	}

	v := struct {
		SID       string
		Challenge string
		BlockTime string
	}{}

	if err = xml.Unmarshal(body, &v); err == nil && v.SID == "0000000000000000" {
		var challresp string
		if challresp, err = createChallengeResponse(v.Challenge, c.password); err == nil {
			params := url.Values{
				"username": []string{c.user},
				"response": []string{challresp},
			}

			if body, err = c.GetBody(uri + "?" + params.Encode()); err == nil {
				err = xml.Unmarshal(body, &v)
				if v.SID == "0000000000000000" {
					return errors.New("invalid username (" + c.user + ") or password")
				}
				c.sid = v.SID
			}
		}
	}

	return err
}

// createChallengeResponse creates the Fritzbox challenge response string
func createChallengeResponse(challenge string, pass string) (string, error) {
	encoder := unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewEncoder()
	utf16le, err := encoder.String(challenge + "-" + pass)
	if err != nil {
		return "", err
	}

	hash := md5.New()
	if _, err = hash.Write([]byte(utf16le)); err != nil {
		return "", err
	}

	md5hash := hex.EncodeToString(hash.Sum(nil))
	return challenge + "-" + md5hash, nil
}
