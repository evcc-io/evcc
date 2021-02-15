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

// fritzdect AHA-HTTP StatusResponse is the API response if status not OK
type fritzdectStatusResponse struct {
	Name    string  // DECT Switch name
	Present int64   // 0/1 - DECT Switch connected to fritzbox (no/yes)
	State   int64   // 0/1 - DECT Switch state off/on (empty if unkown or error)
	Power   float64 // Wert in 0,001 W (aktuelle Leistung, wird etwa alle 2 Minuten aktualisiert)
	Energy  float64 // Wert in 1.0 Wh (absoluter Verbrauch seit Inbetriebnahme)
}

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

func (c *FritzDECT) getFritzBoxResponse(function string) string {
	uri := fmt.Sprintf("%s/webservices/homeautoswitch.lua", c.uri)
	parameters := make(map[string]string)
	// Refresh Fritzbox session id
	if time.Since(c.updated).Minutes() >= 10 {
		err := c.getFritzBoxSessionID()
		if err != nil {
			log.Printf("error in getFritzBoxSessionID: %v", err)
		}
		// Update session timestamp
		c.updated = time.Now()
	}
	parameters["sid"] = c.sid
	if c.ain != "" {
		parameters["ain"] = c.ain
	}
	parameters["switchcmd"] = function
	response, _ := sendFritzBoxRequest(uri, parameters)
	return strings.TrimSpace(string(response))
}

func (c *FritzDECT) apiStatus() (status fritzdectStatusResponse, err error) {
	status.Name = c.getFritzBoxResponse("getswitchname")
	status.Present, err = strconv.ParseInt(c.getFritzBoxResponse("getswitchpresent"), 10, 64)
	if err != nil {
		return status, err
	}
	status.State, err = strconv.ParseInt(c.getFritzBoxResponse("getswitchstate"), 10, 32)
	if err != nil {
		return status, err
	}
	if err != nil {
		return status, err
	}
	status.Power, err = strconv.ParseFloat(c.getFritzBoxResponse("getswitchpower"), 64)
	if err != nil {
		return status, err
	}
	status.Power = status.Power / 1000 // mW ==> W
	status.Energy, err = strconv.ParseFloat(c.getFritzBoxResponse("getswitchenergy"), 64)
	if err != nil {
		return status, err
	}
	status.Energy = status.Energy / 1000 // Wh ==> kWh
	return status, err
}

// apiUpdate invokes fritzdect api
func (c *FritzDECT) apiUpdate(function string) (status fritzdectStatusResponse, err error) {
	if function == "SetSwitchOn" {
		status.State, err = strconv.ParseInt(c.getFritzBoxResponse("setswitchon"), 10, 32)
	}
	if function == "SetSwitchOff" {
		status.State, err = strconv.ParseInt(c.getFritzBoxResponse("setswitchoff"), 10, 32)
	}
	return status, err
}

// validFritzBoxResponse checks is fritz DECT status response is local
func validFritzBoxResponse(status fritzdectStatusResponse) bool {
	return status.Present != 0
}

// Status implements the Charger.Status interface
func (c *FritzDECT) Status() (api.ChargeStatus, error) {
	status, err := c.apiStatus()
	if err != nil {
		return api.StatusNone, err
	}
	switch status.Present {
	case 1:
		if status.Power == 0 {
			return api.StatusA, nil
		}
		if status.Power > 0 && status.Power <= c.standbypower {
			return api.StatusB, nil
		}
		if status.Power > c.standbypower {
			return api.StatusC, nil
		}
	}
	return api.StatusNone, fmt.Errorf("DECT switch not present")
}

// Enabled implements the Charger.Enabled interface
func (c *FritzDECT) Enabled() (bool, error) {
	status, err := c.apiStatus()
	if err != nil {
		return false, err
	}
	switch status.State {
	case 0:
		return false, nil
	case 1:
		return true, nil
	default:
		return false, fmt.Errorf("state unknown result: %d", status.State)
	}
}

// Enable implements the Charger.Enable interface
func (c *FritzDECT) Enable(enable bool) (err error) {
	var status fritzdectStatusResponse
	if enable {
		status, err = c.apiUpdate("SetSwitchOn")
	} else {
		status, err = c.apiUpdate("SetSwitchOff")
	}
	if err != nil {
		return err
	}
	if enable && validFritzBoxResponse(status) && status.State == 0 {
		return fmt.Errorf("wasn't able to switchOn: %d", status.State)
	}
	if !enable && validFritzBoxResponse(status) && status.State == 1 {
		return fmt.Errorf("wasn't able to switchOff: %d", status.State)
	}
	return nil
}

// MaxCurrent implements the Charger.MaxCurrent interface (Dummy function)
func (c *FritzDECT) MaxCurrent(current int64) error {
	// FritzDECT switch has no option to set MaxCurrent
	return nil
}

// CurrentPower implements the Meter interface.
func (c *FritzDECT) CurrentPower() (float64, error) {
	status, err := c.apiStatus()
	if status.Power < c.standbypower {
		return 0, err
	}
	return status.Power, err
}

// ChargedEnergy implements the ChargeRater interface
func (c *FritzDECT) ChargedEnergy() (float64, error) {
	status, err := c.apiStatus()
	energy := status.Energy / 1000
	return energy, err
}

// Fritzbox helpers (based on ideas of https://github.com/rsdk/ahago)

//getFritzBoxSessionID fetches a session-id based on the username and password in the connection struct
func (c *FritzDECT) getFritzBoxSessionID() error {
	uri := fmt.Sprintf("%s/login_sid.lua", c.uri)
	parameters := make(map[string]string)
	type Result struct {
		SID       string
		Challenge string
		BlockTime string
	}
	body, err := sendFritzBoxRequest(uri, parameters)
	if err != nil {
		return err
	}
	v := Result{SID: "none", Challenge: "none", BlockTime: "none"}
	err = xml.Unmarshal(body, &v)
	if err != nil {
		return err
	}
	if v.SID == "0000000000000000" {
		parameters["username"] = c.user
		parameters["response"] = createFbChallengeResponse(v.Challenge, c.password)
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
func sendFritzBoxRequest(uri string, parametersIn map[string]string) ([]byte, error) {
	var URL *url.URL
	URL, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}
	parameters := url.Values{}
	for key, value := range parametersIn {
		parameters.Add(key, value)
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
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

// createFbChallengeResponse creates the Fritzbox challenge response string
func createFbChallengeResponse(challenge string, pass string) string {
	encoder := unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewEncoder()
	utf16le, err := encoder.String(challenge + "-" + pass)
	if err != nil {
		log.Printf("error in unicode.UTF16 encoder: %v", err)
	}
	hash := md5.New()
	n, err := hash.Write([]byte(utf16le))
	if err != nil {
		log.Printf("error in createFbChallengeResponse md5 hash creation: %b - %v", n, err)
	}
	md5hash := hex.EncodeToString(hash.Sum(nil))
	return challenge + "-" + md5hash
}
