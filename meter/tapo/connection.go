package tapo

import (
	"fmt"
	"net/netip"
	"net/url"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

// Tapo homepage + api reverse engineering results
// https://www.tapo.com/de/
// Credits to & inspired by:
// https://k4czp3r.xyz/reverse-engineering/tp-link/tapo/2020/10/15/reverse-engineering-tp-link-tapo.html
// https://github.com/fishbigger/TapoP100
// https://github.com/artemvang/p100-go

const Timeout = time.Second * 15

// Connection is the Tapo connection
type Connection struct {
	*request.Helper
	log             *util.Logger
	plug            Plug
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

	log := util.NewLogger("tapo")

	plug := NewPlug(addr, log)
	if err := plug.Handshake(user, password); err != nil {
		return nil, fmt.Errorf("login failed: %w", err)
	}

	conn := &Connection{
		log:    log,
		Helper: request.NewHelper(log),
		plug:   *plug,
	}

	conn.Client.Timeout = Timeout

	return conn, err
}

// Enable implements the api.Charger interface
func (d *Connection) Enable(enable bool) error {
	return d.plug.SetDeviceInfo(enable)
}

// Enabled implements the api.Charger interface
func (d *Connection) Enabled() (bool, error) {
	resp, err := d.plug.GetDeviceInfo()
	if err != nil {
		return false, err
	}

	return resp.DeviceON, nil
}

// CurrentPower provides current power consuption
func (d *Connection) CurrentPower() (float64, error) {
	resp, err := d.plug.GetEnergyUsage()
	if err != nil {
		return 0, err
	}

	return float64(resp.CurrentPower) / 1e3, nil
}

// ChargedEnergy collects the daily charged energy
func (d *Connection) ChargedEnergy() (float64, error) {
	resp, err := d.plug.GetEnergyUsage()
	if err != nil {
		return 0, err
	}

	if resp.TodayEnergy > d.lasttodayenergy {
		d.energy = d.energy + (resp.TodayEnergy - d.lasttodayenergy)
	}
	d.lasttodayenergy = resp.TodayEnergy

	return float64(d.energy) / 1000, nil
}
