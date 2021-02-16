package charger

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
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
		Cache        time.Duration
	}{}
	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}
	if cc.URI == "" || cc.AIN == "" {
		return nil, errors.New("fritzdect config: must have uri and ain of AVM FritzDECT switch")
	}
	return NewFritzDECT(cc.URI, cc.AIN, cc.User, cc.Password, cc.SID, cc.StandbyPower, cc.Cache)
}

// NewFritzDECT creates FritzDECT charger
func NewFritzDECT(uri, ain, user, password, sid string, standbypower float64, cache time.Duration) (*FritzDECT, error) {
	c := &FritzDECT{
		Helper:       request.NewHelper(util.NewLogger("fritzdect")),
		uri:          strings.TrimRight(uri, "/"),
		ain:          ain,
		user:         user,
		password:     password,
		standbypower: standbypower,
		sid:          sid,
	}
	return c, nil
}

func (c *FritzDECT) execFritzDectCmd(function string) string {
	// Refresh Fritzbox session id
	if time.Since(c.updated).Minutes() >= 10 {
		err := c.getFritzBoxSessionID()
		if err != nil {
			log.Printf("error in getFritzBoxSessionID: %v", err)
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
	response, _ := sendFritzBoxRequest(uri, parameters)
	return strings.TrimSpace(string(response))
}

// Status implements the Charger.Status interface
func (c *FritzDECT) Status() (api.ChargeStatus, error) {
	// present 0/1 - DECT Switch connected to fritzbox (no/yes)
	var present int64
	// power value in 0,001 W (current switch power, refresh aproximately every 2 minutes)
	var power float64
	var err error
	present, err = strconv.ParseInt(c.execFritzDectCmd("getswitchpresent"), 10, 64)
	if err != nil {
		return api.StatusNone, err
	}
	power, err = strconv.ParseFloat(c.execFritzDectCmd("getswitchpower"), 64)
	if err != nil {
		return api.StatusNone, err
	}
	power = power / 1000 // mW ==> W
	switch present {
	case 1:
		switch {
		case power == 0:
			return api.StatusA, nil
		case power > 0 && power <= c.standbypower:
			return api.StatusB, nil
		case power > c.standbypower:
			return api.StatusC, nil
		}
	}
	return api.StatusNone, fmt.Errorf("DECT switch not present")
}

// Enabled implements the Charger.Enabled interface
func (c *FritzDECT) Enabled() (bool, error) {
	// state 0/1 - DECT Switch state off/on (empty if unkown or error)
	state, err := strconv.ParseInt(c.execFritzDectCmd("getswitchstate"), 10, 32)
	if err != nil {
		return false, err
	}
	return state == 1, nil
}

// Enable implements the Charger.Enable interface
func (c *FritzDECT) Enable(enable bool) error {
	// state 0/1 - DECT Switch state off/on (empty if unkown or error)
	var state int64
	var err error
	if enable {
		state, err = strconv.ParseInt(c.execFritzDectCmd("setswitchon"), 10, 32)
	} else {
		state, err = strconv.ParseInt(c.execFritzDectCmd("setswitchoff"), 10, 32)
	}
	switch {
	case err != nil:
		return err
	case enable && state == 0:
		return fmt.Errorf("wasn't able to switchOn: %d", state)
	case !enable && state == 1:
		return fmt.Errorf("wasn't able to switchOff: %d", state)
	default:
		return nil
	}
}

// MaxCurrent implements the Charger.MaxCurrent interface (Dummy function)
func (c *FritzDECT) MaxCurrent(current int64) error {
	// FritzDECT switch has no option to set MaxCurrent
	return nil
}

// CurrentPower implements the Meter interface.
func (c *FritzDECT) CurrentPower() (float64, error) {
	// power value in 0,001 W (current switch power, refresh aproximately every 2 minutes)
	power, err := strconv.ParseFloat(c.execFritzDectCmd("getswitchpower"), 64)
	if err != nil {
		return 0, err
	}
	power = power / 1000 // mW ==> W
	if power < c.standbypower {
		return 0, err
	}
	return power, err
}

// ChargedEnergy implements the ChargeRater interface
func (c *FritzDECT) ChargedEnergy() (float64, error) {
	// energy in 1.0 Wh (total energy since first activation or last manual reset)
	energy, err := strconv.ParseFloat(c.execFritzDectCmd("getswitchenergy"), 64)
	if err != nil {
		return 0, err
	}
	energy = energy / 1000 // Wh ==> kWh
	return energy, err
}

// Fritzbox helpers (based on ideas of https://github.com/rsdk/ahago)

//getFritzBoxSessionID fetches a session-id based on the username and password in the connection struct
func (c *FritzDECT) getFritzBoxSessionID() error {
	uri := fmt.Sprintf("%s/login_sid.lua", c.uri)
	var parameters url.Values
	type result struct {
		SID       string
		Challenge string
		BlockTime string
	}
	body, err := sendFritzBoxRequest(uri, parameters)
	if err != nil {
		return err
	}
	v := result{SID: "none", Challenge: "none", BlockTime: "none"}
	err = xml.Unmarshal(body, &v)
	if err != nil {
		return err
	}
	if v.SID == "0000000000000000" {
		var challresp string
		challresp, err = createFbChallengeResponse(v.Challenge, c.password)
		if err != nil {
			return err
		}
		parameters = url.Values{
			"username": []string{c.user},
			"response": []string{challresp},
		}
		body, err = sendFritzBoxRequest(uri, parameters)
		if err != nil {
			return err
		}
		err = xml.Unmarshal(body, &v)
		if err != nil {
			return err
		}
	}
	c.sid = v.SID
	return nil
}

// sendFritzBoxRequest sends a HTTP Request to the FritzBox based on the given URL and returns the answer as a bytearray
func sendFritzBoxRequest(uri string, parameters url.Values) ([]byte, error) {
	var URL *url.URL
	URL, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}
	URL.RawQuery = parameters.Encode()
	resp, err := http.Get(URL.String())
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, errors.New("sendFritzBoxRequest status code != 200")
	}
	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}

// createFbChallengeResponse creates the Fritzbox challenge response string
func createFbChallengeResponse(challenge string, pass string) (string, error) {
	encoder := unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewEncoder()
	utf16le, err := encoder.String(challenge + "-" + pass)
	if err != nil {
		log.Printf("error in unicode.UTF16 encoder: %v", err)
		return "", err
	}
	hash := md5.New()
	n, err := hash.Write([]byte(utf16le))
	if err != nil {
		log.Printf("error in createFbChallengeResponse md5 hash creation: %b - %v", n, err)
		return "", err
	}
	md5hash := hex.EncodeToString(hash.Sum(nil))
	return challenge + "-" + md5hash, nil
}
