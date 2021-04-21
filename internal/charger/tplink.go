package charger

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

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

// NewTPLinkFromConfig creates a TP-Link charger from generic config
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
		uri:          net.JoinHostPort(uri, "9999"),
		standbypower: standbypower,
	}
	return c, nil
}

// Enabled implements the Charger.Enabled interface
func (c *TPLink) Enabled() (bool, error) {
	sysResp, err := c.execCmd(`{"system":{"get_sysinfo":null}}`)
	if err != nil {
		return false, err
	}

	var systemResponse tplink.SystemResponse
	if err := json.Unmarshal(sysResp, &systemResponse); err != nil {
		return false, err
	}

	if err := systemResponse.System.GetSysinfo.ErrCode; err != 0 {
		return false, fmt.Errorf("get_sysinfo error %d", err)
	}

	if !strings.Contains(systemResponse.System.GetSysinfo.Feature, "ENE") {
		return false, errors.New(systemResponse.System.GetSysinfo.Model + " not supported, energy meter feature missing")
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

	if err := systemResponse.System.SetRelayState.ErrCode; err != 0 {
		return fmt.Errorf("set_relay_state error %d", err)
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
	emeResp, err := c.execCmd(`{"emeter":{"get_realtime":null}}`)
	if err != nil {
		return 0, err
	}

	var emeterResponse tplink.EmeterResponse
	if err := json.Unmarshal(emeResp, &emeterResponse); err != nil {
		return 0, err
	}
	if err := emeterResponse.Emeter.GetRealtime.ErrCode; err != 0 {
		return 0, fmt.Errorf("get_realtime error %d", err)
	}

	power := emeterResponse.Emeter.GetRealtime.PowerMw / 1000
	if power == 0 {
		power = emeterResponse.Emeter.GetRealtime.Power
	}

	// ignore standby power
	if power < c.standbypower {
		power = 0
	}

	return power, err
}

// execCmd executes an TP-Link Smart Home Protocol command and provides the response
func (c *TPLink) execCmd(cmd string) ([]byte, error) {
	// encode command message
	buf := bytes.NewBuffer([]byte{0, 0, 0, 0})
	var ekey byte = 171 // initialization vector
	for i := 0; i < len(cmd); i++ {
		ekey = ekey ^ cmd[i]
		_ = buf.WriteByte(ekey)
	}

	// write 4 bytes to start of buffer with command length
	binary.BigEndian.PutUint32(buf.Bytes(), uint32(buf.Len()-4))

	// open connection via TP-Link Smart Home Protocol
	conn, err := net.DialTimeout("tcp", c.uri, 5*time.Second)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	// send command
	if _, err = buf.WriteTo(conn); err != nil {
		return nil, err
	}

	// read response
	resp := make([]byte, 2048)
	n, err := conn.Read(resp)
	if err != nil {
		return nil, err
	}

	// decode response message
	var dkey byte = 171 // initialization vector
	for i := 4; i < n; i++ {
		dec := dkey ^ resp[i]
		dkey = resp[i]
		_ = buf.WriteByte(dec)
	}

	return buf.Bytes(), nil
}
