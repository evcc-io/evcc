package charger

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

	mu       sync.RWMutex
	status   *evsemaster.ACStatus
	loggedIn bool
	current  int // last value set by MaxCurrent

	// evseAddr is the EVSE's source address (e.g. 192.168.1.100:11938).
	// It is learned from the first Login broadcast and used for all sends.
	evseAddr *net.UDPAddr

	recv  chan *evsemaster.ReceivedPacket
	ready chan struct{} // closed once the first ACStatus is received
}

func init() {
	registry.AddCtx("evsemaster-udp", NewEVSEMasterFromConfig)
}

// NewEVSEMasterFromConfig creates an EVSEMaster charger from a generic config map.
func NewEVSEMasterFromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
	var cc = struct {
		Serial   string
		Password string
	}{}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewEVSEMaster(ctx, cc.Serial, cc.Password)
}

// NewEVSEMaster creates a new EVSEMaster charger and returns once the first
// ACStatus is received (or ctx / 60 s timeout elapses).
func NewEVSEMaster(ctx context.Context, serial, password string) (*EVSEMaster, error) {
	log := util.NewLogger("evsemaster")

	if len(serial) != 16 {
		return nil, fmt.Errorf("serial must be a 16-character hex string, got %q", serial)
	}

	conn, err := evsemaster.NewConnection(log, serial, password)
	if err != nil {
		return nil, fmt.Errorf("listener: %w", err)
	}

	wb := &EVSEMaster{
		log:     log,
		conn:    conn,
		current: 6,
		recv:    make(chan *evsemaster.ReceivedPacket, 32),
		ready:   make(chan struct{}),
	}

	conn.Subscribe(wb.recv)

	go wb.run(ctx)

	select {
	case <-wb.ready:
	case <-ctx.Done():
		conn.Unsubscribe()
		return nil, api.ErrTimeout
	case <-time.After(60 * time.Second):
		conn.Unsubscribe()
		return nil, api.ErrTimeout
	}

	return wb, nil
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
func (wb *EVSEMaster) run(ctx context.Context) {
	var lastHeartbeat time.Time
	keepaliveTick := time.NewTicker(15 * time.Second)
	defer keepaliveTick.Stop()
	defer wb.conn.Unsubscribe()

	for {
		select {
		case <-ctx.Done():
			return

		case pkt := <-wb.recv:
			switch pkt.Command {
			case evsemaster.CmdLoginBroadcast:
				// Learn (or refresh) the EVSE's source address.
				wb.mu.Lock()
				wb.evseAddr = pkt.From
				wb.mu.Unlock()

				if err := wb.send(evsemaster.CmdLoginConfirm, []byte{0x00}); err != nil {
					wb.log.WARN.Printf("LoginConfirm: %v", err)
					continue
				}
				if err := wb.send(evsemaster.CmdHeading, nil); err != nil {
					wb.log.WARN.Printf("initial Heading: %v", err)
				}
				wb.mu.Lock()
				wb.loggedIn = true
				wb.mu.Unlock()
				lastHeartbeat = time.Now()
				wb.log.DEBUG.Printf("logged in, EVSE at %s", pkt.From)

			case evsemaster.CmdHeadingFromEVSE:
				if err := wb.send(evsemaster.CmdHeadingResp, nil); err != nil {
					wb.log.WARN.Printf("HeadingResp: %v", err)
				}
				lastHeartbeat = time.Now()

			case evsemaster.CmdACStatus:
				if s, err := evsemaster.ParseACStatus(pkt.Payload); err == nil {
					wb.mu.Lock()
					firstStatus := wb.status == nil
					wb.status = s
					wb.mu.Unlock()
					if firstStatus {
						close(wb.ready)
					}
				} else {
					wb.log.WARN.Printf("ACStatus parse: %v", err)
				}
				_ = wb.send(evsemaster.CmdStatusAck, []byte{0x01})

			case evsemaster.CmdChargeStatus:
				_ = wb.send(evsemaster.CmdChargingAck, []byte{0x00})
			}

		case <-keepaliveTick.C:
			// Detect heartbeat timeout and mark offline.
			if !lastHeartbeat.IsZero() && time.Since(lastHeartbeat) > 90*time.Second {
				wb.log.DEBUG.Printf("heartbeat timeout – marking offline, waiting for next broadcast")
				wb.mu.Lock()
				wb.loggedIn = false
				wb.mu.Unlock()
				break
			}
			wb.mu.RLock()
			li := wb.loggedIn
			wb.mu.RUnlock()
			if li {
				if err := wb.send(evsemaster.CmdHeading, nil); err != nil {
					wb.log.WARN.Printf("keepalive: %v", err)
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
	wb.mu.RLock()
	defer wb.mu.RUnlock()

	if wb.status == nil {
		return api.StatusNone, api.ErrTimeout
	}

	switch {
	case wb.status.OutputState == 1:
		return api.StatusC, nil
	case wb.status.GunState >= 2:
		return api.StatusB, nil
	default:
		return api.StatusA, nil
	}
}

// Enabled implements the api.Charger interface.
func (wb *EVSEMaster) Enabled() (bool, error) {
	wb.mu.RLock()
	defer wb.mu.RUnlock()

	if wb.status == nil {
		return false, api.ErrTimeout
	}
	return wb.status.OutputState == 1, nil
}

// Enable implements the api.Charger interface.
func (wb *EVSEMaster) Enable(enable bool) error {
	wb.mu.RLock()
	li := wb.loggedIn
	current := wb.current
	wb.mu.RUnlock()

	if !li {
		return api.ErrTimeout
	}

	if enable {
		payload, err := evsemaster.PackChargeStart(current)
		if err != nil {
			return err
		}
		return wb.send(evsemaster.CmdChargeStart, payload)
	}

	return wb.send(evsemaster.CmdChargeStop, nil)
}

// MaxCurrent implements the api.Charger interface.
func (wb *EVSEMaster) MaxCurrent(current int64) error {
	wb.mu.RLock()
	li := wb.loggedIn
	wb.mu.RUnlock()

	if li {
		if err := wb.send(evsemaster.CmdSetCurrent, evsemaster.PackSetCurrent(int(current))); err != nil {
			return err
		}
	}

	wb.mu.Lock()
	wb.current = int(current)
	wb.mu.Unlock()

	return nil
}

var _ api.Meter = (*EVSEMaster)(nil)

// CurrentPower implements the api.Meter interface.
func (wb *EVSEMaster) CurrentPower() (float64, error) {
	wb.mu.RLock()
	defer wb.mu.RUnlock()
	if wb.status == nil {
		return 0, api.ErrTimeout
	}
	return wb.status.Power, nil
}

var _ api.MeterEnergy = (*EVSEMaster)(nil)

// TotalEnergy implements the api.MeterEnergy interface.
func (wb *EVSEMaster) TotalEnergy() (float64, error) {
	wb.mu.RLock()
	defer wb.mu.RUnlock()
	if wb.status == nil {
		return 0, api.ErrTimeout
	}
	return wb.status.TotalEnergy, nil
}

var _ api.PhaseCurrents = (*EVSEMaster)(nil)

// Currents implements the api.PhaseCurrents interface.
func (wb *EVSEMaster) Currents() (float64, float64, float64, error) {
	wb.mu.RLock()
	defer wb.mu.RUnlock()
	if wb.status == nil {
		return 0, 0, 0, api.ErrTimeout
	}
	return wb.status.L1Current, wb.status.L2Current, wb.status.L3Current, nil
}

var _ api.PhaseVoltages = (*EVSEMaster)(nil)

// Voltages implements the api.PhaseVoltages interface.
func (wb *EVSEMaster) Voltages() (float64, float64, float64, error) {
	wb.mu.RLock()
	defer wb.mu.RUnlock()
	if wb.status == nil {
		return 0, 0, 0, api.ErrTimeout
	}
	return wb.status.L1Voltage, wb.status.L2Voltage, wb.status.L3Voltage, nil
}
