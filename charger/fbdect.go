package charger

import (
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/request"
)

// AVM FritzBox AHA interface and authentification specifications:
// https://avm.de/fileadmin/user_upload/Global/Service/Schnittstellen/AHA-HTTP-Interface.pdf
// https://avm.de/fileadmin/user_upload/Global/Service/Schnittstellen/AVM_Technical_Note_-_Session_ID.pdf

// fbdect AHA-HTTP StatusResponse is the API response if status not OK
type fbdectStatusResponse struct {
	Name        string  // DECT Switch name
	Present     int64   // 0/1 - DECT Switch connected to fritzbox (no/yes)
	State       int64   // 0/1 - DECT Switch state off/on (empty if unkown or error)
	Temperature float64 // Wert in 0,1 °C, negative und positive Werte möglich
	Power       float64 // Wert in 0,001 W (aktuelle Leistung, wird etwa alle 2 Minuten aktualisiert)
	Energy      float64 // Wert in 1.0 Wh (absoluter Verbrauch seit Inbetriebnahme)
}

// FbDect charger implementation
type FbDect struct {
	*request.Helper
	uri, ain, user, password, sid string
	updated                       time.Time
}

func init() {
	registry.Add("fbdect", NewFbDectFromConfig)
}

// NewFbDectFromConfig creates a fbdect charger from generic config
func NewFbDectFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		URI      string
		AIN      string
		User     string
		Password string
		SID      string
		Cache    time.Duration
	}{
		URI:      "",
		AIN:      "",
		User:     "",
		Password: "",
		SID:      "",
		Cache:    0,
	}
	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.URI == "" || cc.AIN == "" {
		return nil, errors.New("fbdect config: must have uri and ain of AVM FritzDECT switch")
	}

	return NewFbDect(cc.URI, cc.AIN, cc.User, cc.Password, cc.SID, cc.Cache)
}

// NewFbDect creates FbDect charger
func NewFbDect(uri, ain, user, password, sid string, cache time.Duration) (*FbDect, error) {
	c := &FbDect{
		Helper:   request.NewHelper(util.NewLogger("fbdect")),
		uri:      strings.TrimRight(uri, "/"),
		ain:      strings.TrimSpace(ain),
		user:     strings.TrimSpace(user),
		password: strings.TrimSpace(password),
		sid:      strings.TrimSpace(sid),
	}

	return c, nil
}

func (c *FbDect) fbResponse(function string) string {

	loginURL := fmt.Sprintf("%s/webservices/homeautoswitch.lua", c.uri)
	parameters := make(map[string]string)
	// Refresh Fritzbox session id
	if time.Since(c.updated).Minutes() >= 10 {
		c.getFbSessionID()
		// Update session timestamp
		c.updated = time.Now()
	}

	parameters["sid"] = c.sid
	if c.ain != "" {
		parameters["ain"] = c.ain
	}
	parameters["switchcmd"] = function

	return strings.TrimSpace(string(sendFbRequest(loginURL, parameters)))
}

func (c *FbDect) apiStatus() (status fbdectStatusResponse, err error) {
	status.Name = c.fbResponse("getswitchname")
	status.Present, err = strconv.ParseInt(c.fbResponse("getswitchpresent"), 10, 64)
	if err != nil {
		return status, err
	}
	status.State, err = strconv.ParseInt(c.fbResponse("getswitchstate"), 10, 32)
	if err != nil {
		return status, err
	}
	status.Temperature, err = strconv.ParseFloat(c.fbResponse("gettemperature"), 64)
	if err != nil {
		return status, err
	}
	status.Temperature = status.Temperature / 10
	if err != nil {
		return status, err
	}
	status.Power, err = strconv.ParseFloat(c.fbResponse("getswitchpower"), 64)
	if err != nil {
		return status, err
	}
	status.Power = status.Power / 1000 // mW ==> W
	status.Energy, err = strconv.ParseFloat(c.fbResponse("getswitchenergy"), 64)
	if err != nil {
		return status, err
	}
	status.Energy = status.Energy / 1000 // Wh ==> kWh

	return status, err
}

// apiUpdate invokes fbdect api
func (c *FbDect) apiUpdate(function string) (status fbdectStatusResponse, err error) {
	if function == "SetSwitchOn" {
		status.State, err = strconv.ParseInt(strings.TrimSpace(c.fbResponse("setswitchon")), 10, 32)
	}
	if function == "SetSwitchOff" {
		status.State, err = strconv.ParseInt(strings.TrimSpace(c.fbResponse("setswitchoff")), 10, 32)
	}

	return status, err
}

// isFbResponseValid checks is fritz DECT status response is local
func isFbResponseValid(status fbdectStatusResponse) bool {
	return status.Present != 0
}

// Status implements the Charger.Status interface
func (c *FbDect) Status() (api.ChargeStatus, error) {
	status, err := c.apiStatus()
	if err != nil {
		return api.StatusNone, err
	}

	if status.Present == 1 {

		if status.Power == 0 {
			return api.StatusA, nil
		}

		if status.Power > 0 && status.Power < 8 {
			return api.StatusB, nil
		}

		if status.Power > 3 {
			return api.StatusC, nil
		}

	}

	return api.StatusNone, fmt.Errorf("DECT Switch not present")

}

// Enabled implements the Charger.Enabled interface
func (c *FbDect) Enabled() (bool, error) {
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
		return false, fmt.Errorf("State unknown result: %d", status.State)
	}
}

// Enable implements the Charger.Enable interface
func (c *FbDect) Enable(enable bool) (err error) {
	var status fbdectStatusResponse

	switch enable {
	case true:
		status, err = c.apiUpdate("SetSwitchOn")
		if err != nil {
			return err
		}

	case false:
		status, err = c.apiUpdate("SetSwitchOff")
		if err != nil {
			return err
		}
	}

	if enable && isFbResponseValid(status) && status.State == 0 {
		return fmt.Errorf("Wasn't able to switchOn: %d", status.State)
	}

	if !enable && isFbResponseValid(status) && status.State == 1 {
		return fmt.Errorf("Wasn't able to switchOff: %d", status.State)
	}

	return nil
}

// MaxCurrent implements the Charger.MaxCurrent interface (Dummy function)
func (c *FbDect) MaxCurrent(current int64) error {
	// FritzDECT switch has no option to set MaxCurrent
	return nil
}

// CurrentPower implements the Meter interface.
func (c *FbDect) CurrentPower() (float64, error) {
	status, err := c.apiStatus()
	return float64(status.Power), err
}

// ChargedEnergy implements the ChargeRater interface
func (c *FbDect) ChargedEnergy() (float64, error) {
	status, err := c.apiStatus()

	energy := float64(status.Energy) / 1000

	return energy, err
}

// Currents implements the MeterCurrent interface
func (c *FbDect) Currents() (float64, float64, float64, error) {
	status, err := c.apiStatus()
	// FritzDECT switch provides no ampere meter
	return float64(status.Power) / 230, 0, 0, err
}

// Fritzbox helpers (thx to ahago)

//getFbSessionID fetches a session-id based on the username and password in the connection struct
func (c *FbDect) getFbSessionID() {
	loginURL := fmt.Sprintf("%s/login_sid.lua", c.uri)
	parameters := make(map[string]string)
	type Result struct {
		SID       string
		Challenge string
		BlockTime string
	}

	body := sendFbRequest(loginURL, parameters)

	v := Result{SID: "none", Challenge: "none", BlockTime: "none"}
	err := xml.Unmarshal(body, &v)
	if err != nil {
		fmt.Printf("Fehler bei getFbSessionID.Unmarshalling")
	}

	if v.SID == "0000000000000000" {
		parameters["username"] = c.user
		parameters["response"] = createFbChallengeResponse(v.Challenge, c.password)
		body = sendFbRequest(loginURL, parameters)
		err = xml.Unmarshal(body, &v)
		if err != nil {
			fmt.Printf("Fehler bei Unmarshalling2")
		}
	}
	c.sid = v.SID
}

// sendFbRequest sends a HTTP Request to the FritzBox based on the given URL and returns the answer as a bytearray
func sendFbRequest(baseURL string, parametersIn map[string]string) []byte {
	var URL *url.URL
	URL, err := url.Parse(baseURL)
	if err != nil {
		fmt.Printf("Fb URL parse error: %s", baseURL)
	}
	parameters := url.Values{}
	for key, value := range parametersIn {
		parameters.Add(key, value)
	}
	URL.RawQuery = parameters.Encode()

	resp, err := http.Get(URL.String())
	if err != nil {
		fmt.Printf("Fb Get URL error: %s", URL.String())
	}
	if resp.StatusCode != 200 {
		fmt.Printf("Response: %o", resp.StatusCode)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Fb response body read error: %s", body)
	}

	return body
}

// createFbChallengeResponse creates the Fritzbox challenge response string
func createFbChallengeResponse(challenge string, pass string) string {
	// Create Response string to be crypted
	utf8 := []byte(challenge + "-" + pass)
	utf16le := encFbUTF8ToUTF16le(utf8)
	hash := md5.New()
	x, err := hash.Write(utf16le)
	if err != nil {
		fmt.Printf("Error in hash.Write: %b", x)
	}
	md5hash := hex.EncodeToString(hash.Sum(nil))
	return challenge + "-" + md5hash
}

// encFbUTF8ToUTF16le encodes an UTF8 byte array to an UTF16LE byte array
func encFbUTF8ToUTF16le(in []byte) []byte {
	var ucps []rune
	var utf16uint []uint16
	var utf16le []byte
	for len(in) > 0 {
		r, size := utf8.DecodeRune(in)
		ucps = append(ucps, r)
		in = in[size:]
	}
	utf16uint = utf16.Encode(ucps)
	b := make([]byte, 2)
	for val := range utf16uint {
		binary.LittleEndian.PutUint16(b, utf16uint[val])
		utf16le = append(utf16le, b[0], b[1])
	}
	return utf16le
}
