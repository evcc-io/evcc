package charger

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"math"
	"net"
	"strings"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/internal/charger/tplink"
	"github.com/andig/evcc/util"
)

// TPLink charger implementation
type TPLink struct {
	uri          string
	standbypower float64
}

func init() {
	registry.Add("tplink", NewTPLinkFromConfig)
}

// NewTasmotaFromConfig creates a TP-Link charger from generic config
func NewTPLinkFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		URI          string
		StandbyPower float64
	}{}
	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}
	if cc.URI == "" {
		return nil, errors.New("missing uri")
	}
	return NewTPLink(cc.URI, cc.StandbyPower)
}

// NewTPLink creates TP-Link charger
func NewTPLink(uri string, standbypower float64) (*TPLink, error) {
	c := &TPLink{
		uri:          strings.TrimRight(uri, "/"),
		standbypower: standbypower,
	}
	return c, nil
}

// Enabled implements the Charger.Enabled interface
func (c *TPLink) Enabled() (bool, error) {

	// Execute TP-Link get_sysinfo command
	sysResp, err := c.execCmd(`{ "system":{ "get_sysinfo":null } }`)
	if err != nil {
		return false, err
	}
	var systemResponse tplink.SystemResponse
	if err := json.Unmarshal(sysResp, &systemResponse); err != nil {
		return false, err
	}
	if systemResponse.System.GetSysinfo.ErrCode != 0 {
		return false, errors.New("System.GetSysinfo failed")
	}
	if !strings.Contains(systemResponse.System.GetSysinfo.Feature, "ENE") {
		return false, errors.New(systemResponse.System.GetSysinfo.Model + " plug not supported. energy meter feature missing")
	}
	return int(1) == systemResponse.System.GetSysinfo.RelayState, err
}

// Enable implements the Charger.Enable interface
func (c *TPLink) Enable(enable bool) error {

	cmd := `{"system":{"set_relay_state":{"state":0}}}`
	if enable {
		cmd = `{"system":{"set_relay_state":{"state":1}}}`
	}

	// Execute TP-Link set_relay_state command
	sysResp, err := c.execCmd(cmd)
	if err != nil {
		return err
	}
	var systemResponse tplink.SystemResponse
	if err := json.Unmarshal(sysResp, &systemResponse); err != nil {
		return err
	}
	if systemResponse.System.SetRelayState.ErrCode != 0 {
		return errors.New("System.SetRelayState failed")
	}
	return nil
}

// MaxCurrent implements the Charger.MaxCurrent interface
func (c *TPLink) MaxCurrent(current int64) error {
	return nil
}

// Status implements the Charger.Status interface
func (c *TPLink) Status() (api.ChargeStatus, error) {
	power, err := c.CurrentPower()

	switch {
	case power > 0:
		return api.StatusC, err
	default:
		return api.StatusB, err
	}
}

var _ api.Meter = (*TPLink)(nil)

// CurrentPower implements the api.Meter interface
func (c *TPLink) CurrentPower() (float64, error) {
	// Execute TP-Link emeter command
	emeResp, err := c.execCmd(`{ "emeter":{ "get_realtime":null } }`)
	if err != nil {
		return math.NaN(), err
	}
	var emeterResponse tplink.EmeterResponse
	if err := json.Unmarshal(emeResp, &emeterResponse); err != nil {
		return math.NaN(), err
	}
	if emeterResponse.Emeter.GetRealtime.ErrCode != 0 {
		return math.NaN(), errors.New("Emeter.GetRealtime failed")
	}

	power := emeterResponse.Emeter.GetRealtime.Power

	// ignore standby power
	if power < c.standbypower {
		power = 0
	}

	return power, err
}

// execCmd executes an TP-Link Smart Home Protocol command and provides the response
func (c *TPLink) execCmd(cmd string) ([]byte, error) {
	hexHeader := []byte{0, 0, 0, 0} // BigEndian, unsigned integer

	// Encode command message
	// encResult provides the encrypted plug command
	encCommand := hexHeader
	var enc int
	key := 171 // Encryption initialization vector
	for i := 0; i < len(cmd); i++ {
		enc = key ^ int(cmd[i])
		key = enc
		encCommand = append(encCommand, byte(enc))
	}

	// Send command message on port 9999 to plug in local network
	// encResponse receives the encrypted plug response
	var encResponse []byte
	// Open connection via TP-Link Smart Home Protocol port 9999
	conn, err := net.Dial("tcp", c.uri)
	if err != nil {
		return nil, err
	}
	_, err = conn.Write(encCommand)
	if err != nil {
		return nil, err
	}
	encResponse, err = ioutil.ReadAll(conn)
	if err != nil {
		return nil, err
	}

	// Decode response message
	// decResponse provides the decrypted smart plug response
	var decResponse []byte
	var dec int
	key = 171 // Reset initialization vector
	for i := 0; i < len(encResponse); i++ {
		if i > 3 {
			dec = key ^ int(encResponse[i])
			key = int(encResponse[i])
			decResponse = append(decResponse, byte(dec))
		}
	}

	return decResponse, nil
}
