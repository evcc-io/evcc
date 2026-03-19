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
// EVSE Master UDP protocol – e.g. Morec and generic Chinese EVSE devices.
//
// The device is auto-discovered: its IP and ephemeral port are learned from
// its periodic Login broadcast, so only serial and password are required.
//
// Configuration:
//
//	type: evsemaster-udp
//	serial:   0906252400004617   # 16-char hex serial printed on the device
//	password: 100219             # password set in the EVSE Master mobile app
type EVSEMaster struct {
	log      *util.Logger
	serial   string
	password string

	mu       sync.RWMutex
	status   *evsemaster.ACStatus
	loggedIn bool
	maxAmps  int // last value set by MaxCurrent

	// evseAddr is the EVSE's source address (e.g. 10.123.10.99:11938).
	// It is learned from the first Login broadcast and used for all sends.
	evseAddr *net.UDPAddr

	recv  chan *evsemaster.ReceivedPacket
	ready chan struct{} // closed once the first ACStatus is received

	done chan struct{}
}

func init() {
	registry.Add("evsemaster-udp", NewEVSEMasterFromConfig)
}

// NewEVSEMasterFromConfig creates an EVSEMaster charger from a generic config map.
func NewEVSEMasterFromConfig(other map[string]any) (api.Charger, error) {
	cc := struct {
		Serial   string
		Password string
	}{}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewEVSEMaster(cc.Serial, cc.Password)
}

// NewEVSEMaster creates a new EVSEMaster charger and blocks until the first
// status update arrives (or 60 s elapses waiting for the Login broadcast).
func NewEVSEMaster(serial, password string) (*EVSEMaster, error) {
	log := util.NewLogger("evsemaster")

	if len(serial) != 16 {
		return nil, fmt.Errorf("evsemaster: serial must be a 16-character hex string, got %q", serial)
	}

	wb := &EVSEMaster{
		log:      log,
		serial:   serial,
		password: password,
		maxAmps:  6,
		recv:     make(chan *evsemaster.ReceivedPacket, 32),
		ready:    make(chan struct{}),
		done:     make(chan struct{}),
	}

	lst, err := evsemaster.Instance(log)
	if err != nil {
		return nil, fmt.Errorf("evsemaster: listener: %w", err)
	}
	lst.Subscribe(serial, wb.recv)

	go wb.run()

	// Block until the EVSE has broadcast, we've logged in, and the first
	// SingleACStatus has been received.  Typical wait is < 1 broadcast cycle.
	select {
	case <-wb.ready:
	case <-time.After(60 * time.Second):
		lst.Unsubscribe(serial)
		return nil, fmt.Errorf("evsemaster: device with serial %s not found within 60s – check serial, password, and that the EVSE Master app is not connected", serial)
	}

	return wb, nil
}

// send packs and writes a datagram to the EVSE's stored source address.
func (wb *EVSEMaster) send(cmd uint16, payload []byte) error {
	wb.mu.RLock()
	addr := wb.evseAddr
	wb.mu.RUnlock()

	if addr == nil {
		// EVSE has not broadcast yet; silently drop – run() will retry once the address is known.
		return nil
	}

	pkt := &evsemaster.Packet{
		Serial:   wb.serial,
		Password: wb.password,
		Command:  cmd,
		Payload:  payload,
	}
	buf, err := pkt.Pack()
	if err != nil {
		return err
	}

	lst, err := evsemaster.Instance(wb.log)
	if err != nil {
		return err
	}
	return lst.WriteTo(buf, addr)
}

// run is the background goroutine that maintains the EVSE session.
func (wb *EVSEMaster) run() {
	var lastHeartbeat time.Time
	keepaliveTick := time.NewTicker(15 * time.Second)
	reloginTick := time.NewTicker(30 * time.Second)
	defer keepaliveTick.Stop()
	defer reloginTick.Stop()

	for {
		select {
		case <-wb.done:
			return

		case pkt := <-wb.recv:
			switch pkt.Command {

			case evsemaster.CmdLoginBroadcast:
				// Learn (or refresh) the EVSE's source address
				wb.mu.Lock()
				wb.evseAddr = pkt.From
				wb.mu.Unlock()

				if err := wb.send(evsemaster.CmdLoginConfirm, []byte{0x00}); err != nil {
					wb.log.WARN.Printf("evsemaster: LoginConfirm: %v", err)
					continue
				}
				if err := wb.send(evsemaster.CmdHeading, nil); err != nil {
					wb.log.WARN.Printf("evsemaster: initial Heading: %v", err)
				}
				wb.mu.Lock()
				wb.loggedIn = true
				wb.mu.Unlock()
				lastHeartbeat = time.Now()
				wb.log.DEBUG.Printf("evsemaster: logged in, EVSE at %s", pkt.From)

			case evsemaster.CmdHeadingFromEVSE:
				if err := wb.send(evsemaster.CmdHeadingResp, nil); err != nil {
					wb.log.WARN.Printf("evsemaster: HeadingResp: %v", err)
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
					wb.log.WARN.Printf("evsemaster: ACStatus parse: %v", err)
				}
				_ = wb.send(evsemaster.CmdStatusAck, []byte{0x01})

			case evsemaster.CmdChargeStatus:
				_ = wb.send(evsemaster.CmdChargingAck, []byte{0x00})
			}

		case <-keepaliveTick.C:
			wb.mu.RLock()
			li := wb.loggedIn
			wb.mu.RUnlock()
			if li {
				if err := wb.send(evsemaster.CmdHeading, nil); err != nil {
					wb.log.WARN.Printf("evsemaster: keepalive: %v", err)
				}
			}

		case <-reloginTick.C:
			if !lastHeartbeat.IsZero() && time.Since(lastHeartbeat) > 90*time.Second {
				wb.log.DEBUG.Printf("evsemaster: heartbeat timeout – marking offline, waiting for next broadcast")
				wb.mu.Lock()
				wb.loggedIn = false
				wb.mu.Unlock()
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
		return api.StatusNone, fmt.Errorf("evsemaster: no status received yet")
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
		return false, nil
	}
	return wb.status.OutputState == 1, nil
}

// Enable implements the api.Charger interface.
func (wb *EVSEMaster) Enable(enable bool) error {
	wb.mu.RLock()
	li := wb.loggedIn
	maxAmps := wb.maxAmps
	wb.mu.RUnlock()

	if !li {
		return fmt.Errorf("evsemaster: not logged in")
	}

	if enable {
		payload, err := evsemaster.PackChargeStart(maxAmps)
		if err != nil {
			return err
		}
		return wb.send(evsemaster.CmdChargeStart, payload)
	}

	return wb.send(evsemaster.CmdChargeStop, nil)
}

// MaxCurrent implements the api.Charger interface.
func (wb *EVSEMaster) MaxCurrent(current int64) error {
	if current < 6 || current > 32 {
		return fmt.Errorf("evsemaster: current %dA out of range (6-32A)", current)
	}

	wb.mu.Lock()
	wb.maxAmps = int(current)
	li := wb.loggedIn
	wb.mu.Unlock()

	if !li {
		return nil
	}

	return wb.send(evsemaster.CmdSetCurrent, evsemaster.PackSetCurrent(int(current)))
}

var _ api.Meter = (*EVSEMaster)(nil)

// CurrentPower implements the api.Meter interface.
func (wb *EVSEMaster) CurrentPower() (float64, error) {
	wb.mu.RLock()
	defer wb.mu.RUnlock()
	if wb.status == nil {
		return 0, nil
	}
	return wb.status.Power, nil
}

var _ api.MeterEnergy = (*EVSEMaster)(nil)

// TotalEnergy implements the api.MeterEnergy interface.
func (wb *EVSEMaster) TotalEnergy() (float64, error) {
	wb.mu.RLock()
	defer wb.mu.RUnlock()
	if wb.status == nil {
		return 0, nil
	}
	return wb.status.TotalEnergy, nil
}

var _ api.PhaseCurrents = (*EVSEMaster)(nil)

// Currents implements the api.PhaseCurrents interface.
func (wb *EVSEMaster) Currents() (float64, float64, float64, error) {
	wb.mu.RLock()
	defer wb.mu.RUnlock()
	if wb.status == nil {
		return 0, 0, 0, nil
	}
	return wb.status.L1Current, wb.status.L2Current, wb.status.L3Current, nil
}

var _ api.PhaseVoltages = (*EVSEMaster)(nil)

// Voltages implements the api.PhaseVoltages interface.
func (wb *EVSEMaster) Voltages() (float64, float64, float64, error) {
	wb.mu.RLock()
	defer wb.mu.RUnlock()
	if wb.status == nil {
		return 0, 0, 0, nil
	}
	return wb.status.L1Voltage, wb.status.L2Voltage, wb.status.L3Voltage, nil
}
