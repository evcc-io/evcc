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

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/tplink"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/log"
)

// TPLink charger implementation
type TPLink struct {
	log          log.Logger
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
		log:          log.NewLogger("tplink"),
		uri:          net.JoinHostPort(uri, "9999"),
		standbypower: standbypower,
	}
	return c, nil
}

// Enabled implements the api.Charger interface
func (c *TPLink) Enabled() (bool, error) {
	var resp tplink.SystemResponse
	if err := c.execCmd(`{"system":{"get_sysinfo":null}}`, &resp); err != nil {
		return false, err
	}

	if err := resp.System.GetSysinfo.ErrCode; err != 0 {
		return false, fmt.Errorf("get_sysinfo error %d", err)
	}

	if !strings.Contains(resp.System.GetSysinfo.Feature, "ENE") {
		return false, errors.New(resp.System.GetSysinfo.Model + " not supported, energy meter feature missing")
	}

	return resp.System.GetSysinfo.RelayState == 1, nil
}

// Enable implements the api.Charger interface
func (c *TPLink) Enable(enable bool) error {
	cmd := `{"system":{"set_relay_state":{"state":0}}}`
	if enable {
		cmd = `{"system":{"set_relay_state":{"state":1}}}`
	}

	var resp tplink.SystemResponse
	if err := c.execCmd(cmd, &resp); err != nil {
		return err
	}

	if err := resp.System.SetRelayState.ErrCode; err != 0 {
		return fmt.Errorf("set_relay_state error %d", err)
	}

	return nil
}

// MaxCurrent implements the api.Charger interface
func (c *TPLink) MaxCurrent(current int64) error {
	return nil
}

// Status implements the api.Charger interface
func (c *TPLink) Status() (api.ChargeStatus, error) {
	res := api.StatusB

	// static mode
	if c.standbypower < 0 {
		on, err := c.Enabled()
		if on {
			res = api.StatusC
		}

		return res, err
	}

	// standby power mode
	power, err := c.CurrentPower()
	if power > c.standbypower {
		res = api.StatusC
	}

	return res, err
}

var _ api.Meter = (*TPLink)(nil)

// CurrentPower implements the api.Meter interface
func (c *TPLink) CurrentPower() (float64, error) {
	var resp tplink.EmeterResponse
	if err := c.execCmd(`{"emeter":{"get_realtime":null}}`, &resp); err != nil {
		return 0, err
	}

	if err := resp.Emeter.GetRealtime.ErrCode; err != 0 {
		return 0, fmt.Errorf("get_realtime error %d", err)
	}

	power := resp.Emeter.GetRealtime.PowerMw / 1000
	if power == 0 {
		power = resp.Emeter.GetRealtime.Power
	}

	// ignore standby power
	if power < c.standbypower {
		power = 0
	}

	return power, nil
}

var _ api.MeterEnergy = (*TPLink)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (c *TPLink) TotalEnergy() (float64, error) {
	var resp tplink.EmeterResponse
	if err := c.execCmd(`{"emeter":{"get_realtime":null}}`, &resp); err != nil {
		return 0, err
	}

	if err := resp.Emeter.GetRealtime.ErrCode; err != 0 {
		return 0, fmt.Errorf("get_realtime error %d", err)
	}

	energy := resp.Emeter.GetRealtime.TotalWh / 1000
	if energy == 0 {
		energy = resp.Emeter.GetRealtime.Total
	}

	return energy, nil
}

// execCmd executes an TP-Link Smart Home Protocol command and provides the response
func (c *TPLink) execCmd(cmd string, res interface{}) error {
	// encode command message
	buf := bytes.NewBuffer([]byte{0, 0, 0, 0})
	var key byte = 171 // initialization vector
	for i := 0; i < len(cmd); i++ {
		key = key ^ cmd[i]
		_ = buf.WriteByte(key)
	}

	// write 4 bytes command length to start of buffer
	binary.BigEndian.PutUint32(buf.Bytes(), uint32(buf.Len()-4))

	// open connection via TP-Link Smart Home Protocol
	conn, err := net.DialTimeout("tcp", c.uri, 5*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()

	// send command
	if _, err = buf.WriteTo(conn); err != nil {
		return err
	}

	// read response
	resp := make([]byte, 8192)
	len, err := conn.Read(resp)
	if err != nil {
		return err
	}

	// decode response message
	key = 171 // reset initialization vector
	for i := 4; i < len; i++ {
		dec := key ^ resp[i]
		key = resp[i]
		_ = buf.WriteByte(dec)
	}
	c.log.Trace("recv: %s", buf.String())

	return json.Unmarshal(buf.Bytes(), res)
}
