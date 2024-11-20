package tapo

import (
	"fmt"
	"net/netip"
	"net/url"
	"strings"

	"github.com/evcc-io/evcc/util"
	"github.com/insomniacslk/tapo"
)

// Connection is the Tapo connection
type Connection struct {
	log             *util.Logger
	plug            tapo.Plug
	lasttodayenergy int64
	energy          int64
}

// NewConnection creates a new Tapo device connection.
// User is encoded by using MessageDigest of SHA1 which is afterwards B64 encoded.
// Password is directly B64 encoded.
func NewConnection(uri, user, password string) (*Connection, error) {
	url, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("invalid url: %s", uri)
	}

	addr, err := netip.ParseAddr(url.Hostname())
	if err != nil {
		return nil, fmt.Errorf("invalid ip address: %s", uri)
	}

	if user == "" || password == "" {
		return nil, fmt.Errorf("missing user or password")
	}

	log := util.NewLogger("tapo").Redact(user, password)

	plug := tapo.NewPlug(addr, nil)
	if err := plug.Handshake(user, password); err != nil {
		return nil, fmt.Errorf("login failed: %w", err)
	}

	conn := &Connection{
		log:  log,
		plug: *plug,
	}

	res, err := conn.plug.GetDeviceInfo()
	if err != nil {
		return nil, err
	}
	conn.log.DEBUG.Printf("%s %s connected (fw:%s,hw:%s,mac:%s)", res.Type, res.Model, res.FWVersion, res.HWVersion, res.MAC)

	return conn, err
}

// Enable implements the api.Charger interface
func (c *Connection) Enable(enable bool) error {
	return c.plug.SetDeviceInfo(enable)
}

// Enabled implements the api.Charger interface
func (c *Connection) Enabled() (bool, error) {
	resp, err := c.plug.GetDeviceInfo()
	if err != nil {
		return false, err
	}

	return resp.DeviceON, nil
}

// CurrentPower provides current power consuption
func (c *Connection) CurrentPower() (float64, error) {
	resp, err := c.plug.GetEnergyUsage()
	if err != nil {
		if strings.Contains(err.Error(), "-1001") {
			c.log.DEBUG.Printf("meter not available")
			return 0, nil
		} else {
			return 0, err
		}
	}

	return float64(resp.CurrentPower) / 1e3, nil
}

// ChargedEnergy collects the daily charged energy
func (c *Connection) ChargedEnergy() (float64, error) {
	resp, err := c.plug.GetEnergyUsage()
	if err != nil {
		if strings.Contains(err.Error(), "-1001") {
			c.log.DEBUG.Printf("meter not available")
			return 0, nil
		} else {
			return 0, err
		}
	}

	if int64(resp.TodayEnergy) > c.lasttodayenergy {
		c.energy += (int64(resp.TodayEnergy) - c.lasttodayenergy)
	}
	c.lasttodayenergy = int64(resp.TodayEnergy)

	return float64(c.energy) / 1000, nil
}
