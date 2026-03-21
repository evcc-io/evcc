package charger

// LICENSE

// Copyright (c) evcc.io (andig, naltatis, premultiply)

// This module is NOT covered by the MIT license. All rights reserved.

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

// EVSE Master UDP charger integration.
// Protocol credit: https://github.com/johnwoo-nl/emproto (reverse-engineering)
// Reference implementation: https://github.com/Oniric75/evsemasterudp (Home Assistant)
//
// Key protocol insight: the EVSE sends FROM its own port (e.g. 11938) TO the
// app's port 28376.  All replies must go back to the EVSE's source address
// (ip:11938), NOT to ip:28376.  The EVSE's source port is therefore learned
// from its Login broadcast and stored; no URI/IP is needed in the config.

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/evsemaster"
	"github.com/evcc-io/evcc/util"
)

const evsemasterTimeout = 60 * time.Second

// EVSEMaster implements api.Charger (and api.Meter / api.MeterEnergy /
// api.PhaseCurrents / api.PhaseVoltages) for charging stations that use the
// EVSE Master UDP protocol – e.g. Sync EV and generic Chinese EVSE devices.
//
// The device is auto-discovered: its IP and ephemeral port are learned from
// its periodic Login broadcast, so only serial and password are required.
//
// Configuration:
//
//	type: evsemaster-udp
//	serial:   0906252400004617   # 16-char hex serial printed on the device
//	password: 123456             # password set in the EVSE Master mobile app
type EVSEMaster struct {
	log  *util.Logger
	conn *evsemaster.Connection

	mu      sync.RWMutex
	data    *util.Monitor[*evsemaster.ACStatus]
	current int // last value set by MaxCurrent

	// evseAddr is the EVSE's source address (e.g. 192.168.1.100:11938).
	// It is learned from the first Login broadcast and used for all sends.
	evseAddr *net.UDPAddr
}

func init() {
	registry.AddCtx("evsemaster-udp", NewEVSEMasterFromConfig)
}

// NewEVSEMasterFromConfig creates an EVSEMaster charger from a generic config map.
func NewEVSEMasterFromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
	var cc struct {
		Serial   string
		Password string
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewEVSEMaster(ctx, cc.Serial, cc.Password)
}

// NewEVSEMaster creates a new EVSEMaster charger and blocks until the first
// ACStatus is received. Returns api.ErrTimeout if the EVSE does not respond
// within 60 seconds – check serial, password, and that the charger is on the
// same network segment (UDP broadcast does not cross VLANs).
func NewEVSEMaster(ctx context.Context, serial, password string) (*EVSEMaster, error) {
	log := util.NewLogger("evsemaster")

	if len(serial) != 16 {
		return nil, fmt.Errorf("serial must be a 16-character hex string, got %q", serial)
	}

	conn, err := evsemaster.NewConnection(log, serial, password)
	if err != nil {
		return nil, err
	}

	wb := &EVSEMaster{
		log:     log,
		conn:    conn,
		current: 6,
		data:    util.NewMonitor[*evsemaster.ACStatus](evsemasterTimeout),
	}

	done := make(chan struct{})
	go wb.run(ctx, done)

	select {
	case <-done:
		return wb, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(evsemasterTimeout):
		return nil, api.ErrTimeout
	}
}

// send writes a command datagram to the EVSE's stored source address.
func (wb *EVSEMaster) send(cmd uint16, payload []byte) error {
	wb.mu.RLock()
	addr := wb.evseAddr
	wb.mu.RUnlock()

	if addr == nil {
		// EVSE has not broadcast yet; silently drop.
		return nil
	}

	return wb.conn.Send(cmd, payload, addr)
}

// run is the background goroutine that maintains the EVSE session.
func (wb *EVSEMaster) run(ctx context.Context, done chan struct{}) {
	var once sync.Once

	recv := make(chan *evsemaster.ReceivedPacket, 32)
	wb.conn.Subscribe(recv)
	defer wb.conn.Unsubscribe()

	for tick := time.NewTicker(15 * time.Second); ; {
		select {
		case <-ctx.Done():
			return

		case <-tick.C:
			wb.mu.RLock()
			addr := wb.evseAddr
			wb.mu.RUnlock()
			if addr != nil {
				if err := wb.send(evsemaster.CmdHeading, nil); err != nil {
					wb.log.WARN.Printf("keepalive: %v", err)
				}
			}

		case pkt := <-recv:
			switch pkt.Command {
			case evsemaster.CmdLoginBroadcast:
				// Learn (or refresh) the EVSE's source address.
				wb.mu.Lock()
				wb.evseAddr = pkt.From
				wb.mu.Unlock()

				if err := wb.send(evsemaster.CmdLoginConfirm, []byte{0x00}); err != nil {
					wb.log.WARN.Printf("CmdLoginConfirm: %v", err)
					continue
				}
				if err := wb.send(evsemaster.CmdHeading, nil); err != nil {
					wb.log.WARN.Printf("CmdHeading: %v", err)
				}
				wb.log.DEBUG.Printf("logged in, EVSE at %s", pkt.From)

			case evsemaster.CmdHeadingFromEVSE:
				if err := wb.send(evsemaster.CmdHeadingResp, nil); err != nil {
					wb.log.WARN.Printf("HeadingResp: %v", err)
				}

			case evsemaster.CmdACStatus:
				if s, err := evsemaster.ParseACStatus(pkt.Payload); err == nil {
					wb.data.Set(s)
					once.Do(func() { close(done) })
				} else {
					wb.log.WARN.Printf("ACStatus parse: %v", err)
				}
				if err := wb.send(evsemaster.CmdStatusAck, []byte{0x01}); err != nil {
					wb.log.WARN.Printf("ack: %v", err)
				}

			case evsemaster.CmdChargeStatus:
				if err := wb.send(evsemaster.CmdChargingAck, []byte{0x00}); err != nil {
					wb.log.WARN.Printf("ack: %v", err)
				}
			}
		}
	}
}

// Status implements the api.Charger interface.
//
// GunState (TypeScript ref): 0=unknown, 1=disconnected, 2=connected_unlocked,
// 3=negotiating, 4=connected_locked
// OutputState: 0=idle, 1=charging, 2+=other active state
func (wb *EVSEMaster) Status() (api.ChargeStatus, error) {
	res, err := wb.data.Get()
	if err != nil {
		return api.StatusNone, err
	}

	switch {
	case res.OutputState == 1:
		return api.StatusC, nil
	case res.GunState >= 2:
		return api.StatusB, nil
	default:
		return api.StatusA, nil
	}
}

// Enabled implements the api.Charger interface.
func (wb *EVSEMaster) Enabled() (bool, error) {
	res, err := wb.data.Get()
	if err != nil {
		return false, err
	}

	return res.OutputState == 1, nil
}

// Enable implements the api.Charger interface.
func (wb *EVSEMaster) Enable(enable bool) error {
	var err error
	if enable {
		var b []byte
		if b, err = evsemaster.PackChargeStart(wb.current); err != nil {
			return err
		}
		err = wb.send(evsemaster.CmdChargeStart, b)
	} else {
		err = wb.send(evsemaster.CmdChargeStop, nil)
	}

	if err == nil {
		_ = wb.send(evsemaster.CmdHeading, nil) // request immediate status update
	}

	return err
}

// MaxCurrent implements the api.Charger interface.
func (wb *EVSEMaster) MaxCurrent(current int64) error {
	if err := wb.send(evsemaster.CmdSetCurrent, evsemaster.PackSetCurrent(int(current))); err != nil {
		return err
	}
	_ = wb.send(evsemaster.CmdHeading, nil) // request immediate status update

	wb.current = int(current)

	return nil
}

var _ api.Meter = (*EVSEMaster)(nil)

// CurrentPower implements the api.Meter interface.
func (wb *EVSEMaster) CurrentPower() (float64, error) {
	res, err := wb.data.Get()
	if err != nil {
		return 0, err
	}

	return res.Power, nil
}

var _ api.MeterEnergy = (*EVSEMaster)(nil)

// TotalEnergy implements the api.MeterEnergy interface.
func (wb *EVSEMaster) TotalEnergy() (float64, error) {
	res, err := wb.data.Get()
	if err != nil {
		return 0, err
	}

	return res.TotalEnergy, nil
}

var _ api.PhaseCurrents = (*EVSEMaster)(nil)

// Currents implements the api.PhaseCurrents interface.
func (wb *EVSEMaster) Currents() (float64, float64, float64, error) {
	res, err := wb.data.Get()
	if err != nil {
		return 0, 0, 0, err
	}

	return res.L1Current, res.L2Current, res.L3Current, nil
}

var _ api.PhaseVoltages = (*EVSEMaster)(nil)

// Voltages implements the api.PhaseVoltages interface.
func (wb *EVSEMaster) Voltages() (float64, float64, float64, error) {
	res, err := wb.data.Get()
	if err != nil {
		return 0, 0, 0, err
	}

	return res.L1Voltage, res.L2Voltage, res.L3Voltage, nil
}
