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
	"github.com/evcc-io/evcc/util/logx"
)

// TPLink charger implementation
type TPLink struct {
	log          logx.Logger
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
		log:          logx.NewModule("tplink"),
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

var _ api.ChargeRater = (*TPLink)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (c *TPLink) ChargedEnergy() (float64, error) {
	var resp tplink.DayStatResponse
	year, month, day := time.Now().Date()
	cmd := fmt.Sprintf(`{"emeter":{"get_daystat":{"day":%v,"month":%v,"year":%v}}}`, day, int(month), year)
	if err := c.execCmd(cmd, &resp); err != nil {
		return 0, err
	}

	if err := resp.Emeter.GetDaystat.ErrCode; err != 0 {
		return 0, fmt.Errorf("get_daystat error %d", err)
	}

	energy := resp.Emeter.GetDaystat.DayList[len(resp.Emeter.GetDaystat.DayList)-1].EnergyWh / 1000
	if energy == 0 {
		energy = resp.Emeter.GetDaystat.DayList[len(resp.Emeter.GetDaystat.DayList)-1].Energy
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
	logx.Trace(c.log, "recv", buf.String())

	return json.Unmarshal(buf.Bytes(), res)
}
