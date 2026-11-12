package charger

import (
	"context"
	"encoding/binary"
	"fmt"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/grid"
	"github.com/evcc-io/evcc/util/modbus"
)

// Policharger charger implementation for the Ulartec Policharger 8.1 series.
//
// On the tested unit (NW-T2, software 9.30) a session only starts with reg 80 >= 16 A,
// about 205 s after reg 80 was set, and reg 80 = 0 takes about 55 s to stop. The
// charger is therefore kept in normally locked mode (reg 30 = 1): while idle it stays
// armed (reg 80 = 16) with reg 44 capping the current, and reg 88 starts and stops
// charging within seconds. Booleans are 1/0, writing 0xFF00 returns exception 03.
type Policharger struct {
	log      *util.Logger
	conn     *modbus.Connection
	cache    *modbus.Cache
	mu       sync.Mutex
	phases   int
	current  uint16    // last current requested by evcc
	armed    bool      // reg 44 and reg 80 are known to be set for the next start
	cfgOK    time.Time // last time the lock mode was verified
	pvWarned bool
}

const (
	polRegModel        = 1  // 1 IN, 2 PRO, 3 OUT, 4 NW, 5 ÖN
	polRegConnector    = 2  // 1 T1, 2 T2, 3 T23F, 4..7 and 10, 11 dual connector models
	polRegSwVersion    = 3  // two registers: major, minor
	polRegThreePhase   = 25 // EsTrifasico
	polRegLockType     = 30 // TipoDeBloqueo: 0 normal, 1 normally locked
	polRegCurrent      = 37 // Intensidad sensor carga [A]
	polRegConnected    = 39 // Conectado (0 while locked)
	polRegCharging     = 41 // Cargando
	polRegManualLimit  = 44 // Intensidad límite carga manual [A]
	polRegPvGridLimit  = 73 // Intensidad permitida red en carga FV [A]
	polRegSetpoint     = 80 // IntRepartida [A]
	polRegPvProgram    = 85 // Programa FV licence, read-only
	polRegPinConnected = 86 // ConectadoPin1 (valid in normally locked mode)
	polRegLock         = 88 // BloqueoCarga1: 0 charging allowed, 1 locked

	// register blocks read together, gateways become unreliable above about 14 registers
	polBlockStatus, polBlockStatusLen   = 36, 9
	polBlockControl, polBlockControlLen = 80, 10

	polArmCurrent = 16 // minimum reg 80 value that lets the charger start a session
	polMinCurrent = 6
	polMaxCurrent = 32

	polHeartbeat  = time.Minute
	polCfgCheck   = 10 * time.Minute // periodic verification of the lock mode
	polArmCheck   = time.Minute      // the lock mode must be this fresh before arming
	polCacheTTL   = 5 * time.Second  // share reads between the calls of one evcc cycle
	polRetries    = 4
	polRetryDelay = 300 * time.Millisecond
)

// polRegs maps register addresses to values
type polRegs map[uint16]uint16

func init() {
	registry.AddCtx("policharger", NewPolichargerFromConfig)
}

// NewPolichargerFromConfig creates a Policharger charger from generic config
func NewPolichargerFromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
	cc := modbus.Settings{
		ID:       1,
		Baudrate: 9600,
		Comset:   "8N1",
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewPolicharger(ctx, cc)
}

// NewPolicharger creates a Policharger charger
func NewPolicharger(ctx context.Context, settings modbus.Settings) (api.Charger, error) {
	conn, err := settings.Connection(ctx)
	if err != nil {
		return nil, err
	}

	log := util.NewLogger("policharger")
	conn.Logger(log.TRACE)

	wb := newPolicharger(conn, log, polCacheTTL)

	id, err := wb.read(polRegModel, 4)
	if err != nil {
		return nil, fmt.Errorf("identify: %w", err)
	}

	log.DEBUG.Printf("model %d, connector %d, software %d.%02d",
		id[polRegModel], id[polRegConnector], id[polRegSwVersion], id[polRegSwVersion+1])

	switch id[polRegConnector] {
	case 4, 5, 6, 7, 10, 11:
		log.WARN.Println("dual connector model: only the first connector is controlled")
	}

	cfg, err := wb.read(polRegThreePhase, 1)
	if err != nil {
		return nil, fmt.Errorf("phases: %w", err)
	}
	if cfg[polRegThreePhase] != 0 {
		wb.phases = 3
	}

	r, err := wb.state()
	if err != nil {
		return nil, err
	}

	if err := wb.ensureConfig(r, 0); err != nil {
		return nil, fmt.Errorf("lock mode: %w", err)
	}

	go wb.heartbeat(ctx)

	return wb, nil
}

// newPolicharger wires the struct without device probe (also used by tests)
func newPolicharger(conn *modbus.Connection, log *util.Logger, ttl time.Duration) *Policharger {
	return &Policharger{
		log:    log,
		conn:   conn,
		cache:  modbus.NewCache(ttl),
		phases: 1,
	}
}

// heartbeat keeps the charger in locked mode and armed while idle, including after
// a session the car ended by itself
func (wb *Policharger) heartbeat(ctx context.Context) {
	tick := time.NewTicker(polHeartbeat)
	defer tick.Stop()

	for {
		if err := wb.maintain(); err != nil {
			wb.log.WARN.Println("heartbeat:", err)
		}

		select {
		case <-tick.C:
		case <-ctx.Done():
			return
		}
	}
}

// maintain verifies the lock mode and arms the charger if it is not charging
func (wb *Policharger) maintain() error {
	wb.mu.Lock()
	defer wb.mu.Unlock()

	wb.cache.Clear()

	r, err := wb.state()
	if err != nil {
		return err
	}

	if err := wb.ensureConfig(r, polCfgCheck); err != nil {
		return err
	}

	if r[polRegCharging] == 0 && (!wb.armed || r[polRegSetpoint] != polArmCurrent) {
		return wb.arm(r, wb.target())
	}

	return nil
}

// retry repeats op on transport errors, the link to the charger loses single requests
func (wb *Policharger) retry(op func() error) error {
	bo := backoff.WithMaxRetries(backoff.NewConstantBackOff(polRetryDelay), polRetries-1)

	return backoff.Retry(func() error {
		err := op()
		if modbus.IsException(err) {
			return backoff.Permanent(err)
		}
		return err
	}, bo)
}

// read reads a block of holding registers
func (wb *Policharger) read(start, count uint16) (polRegs, error) {
	b, _, err := wb.cache.Fetch(fmt.Sprintf("%d:%d", start, count), func() ([]byte, error) {
		var b []byte
		err := wb.retry(func() (err error) {
			b, err = wb.conn.ReadHoldingRegisters(start, count)
			return err
		})
		return b, err
	})
	if err != nil {
		return nil, err
	}

	if len(b) != 2*int(count) {
		return nil, fmt.Errorf("invalid response length: %d", len(b))
	}

	res := make(polRegs, count)
	for i := range count {
		res[start+i] = binary.BigEndian.Uint16(b[2*i:])
	}

	return res, nil
}

// state reads the status and control register blocks
func (wb *Policharger) state() (polRegs, error) {
	res, err := wb.read(polBlockStatus, polBlockStatusLen)
	if err != nil {
		return nil, err
	}

	ctrl, err := wb.read(polBlockControl, polBlockControlLen)
	if err != nil {
		return nil, err
	}

	for k, v := range ctrl {
		res[k] = v
	}

	return res, nil
}

// write writes a single register. If the echo is lost the value is read back
// before writing again, since the charger may already have applied it.
func (wb *Policharger) write(reg, val uint16) error {
	wb.cache.Clear()
	defer wb.cache.Clear()

	return wb.retry(func() error {
		_, err := wb.conn.WriteSingleRegister(reg, val)
		if err != nil && !modbus.IsException(err) {
			if b, rerr := wb.conn.ReadHoldingRegisters(reg, 1); rerr == nil && len(b) == 2 && binary.BigEndian.Uint16(b) == val {
				return nil
			}
		}
		return err
	})
}

// ensureConfig puts the charger in normally locked mode unless verified within maxAge
func (wb *Policharger) ensureConfig(r polRegs, maxAge time.Duration) error {
	if maxAge > 0 && time.Since(wb.cfgOK) < maxAge {
		return nil
	}

	cfg, err := wb.read(polRegLockType, 1)
	if err != nil {
		return err
	}

	if cfg[polRegLockType] != 1 {
		if r[polRegLock] != 1 {
			// lock before changing mode
			if err := wb.write(polRegLock, 1); err != nil {
				return err
			}
			r[polRegLock] = 1
		}

		if err := wb.write(polRegLockType, 1); err != nil {
			return err
		}
	}

	// with the PV programme licence, a low grid limit blocks charging from the grid
	if r[polRegPvProgram] != 0 && !wb.pvWarned {
		pv, err := wb.read(polRegPvGridLimit, 1)
		if err != nil {
			return err
		}

		if limit := pv[polRegPvGridLimit]; limit < polArmCurrent {
			wb.log.WARN.Printf("PV programme active with grid current limit %d A: sessions may not start, set it to %d A or more in the charger menu", limit, polArmCurrent)
		}
		wb.pvWarned = true
	}

	wb.cfgOK = time.Now()

	return nil
}

// target returns the current to arm the charger with
func (wb *Policharger) target() uint16 {
	return max(wb.current, polMinCurrent)
}

// arm sets the current cap and arms the charger while it is not charging.
// In normal mode reg 80 = 16 would start charging on its own, so the lock mode
// is verified first.
func (wb *Policharger) arm(r polRegs, amps uint16) error {
	if r[polRegManualLimit] != amps {
		if err := wb.write(polRegManualLimit, amps); err != nil {
			return err
		}
		r[polRegManualLimit] = amps
	}

	if r[polRegSetpoint] != polArmCurrent {
		if err := wb.ensureConfig(r, polArmCheck); err != nil {
			return err
		}

		if err := wb.write(polRegSetpoint, polArmCurrent); err != nil {
			return err
		}
		r[polRegSetpoint] = polArmCurrent
	}

	wb.armed = true

	return nil
}

// Status implements the api.Charger interface
func (wb *Policharger) Status() (api.ChargeStatus, error) {
	wb.mu.Lock()
	defer wb.mu.Unlock()

	r, err := wb.state()
	if err != nil {
		return api.StatusNone, err
	}

	switch {
	case r[polRegCharging] != 0:
		return api.StatusC, nil
	case r[polRegPinConnected] != 0 || r[polRegConnected] != 0:
		return api.StatusB, nil
	default:
		return api.StatusA, nil
	}
}

// Enabled implements the api.Charger interface
func (wb *Policharger) Enabled() (bool, error) {
	wb.mu.Lock()
	defer wb.mu.Unlock()

	r, err := wb.read(polBlockControl, polBlockControlLen)
	if err != nil {
		return false, err
	}

	return r[polRegLock] == 0, nil
}

// Enable implements the api.Charger interface
func (wb *Policharger) Enable(enable bool) error {
	wb.mu.Lock()
	defer wb.mu.Unlock()

	if !enable {
		wb.armed = false

		// stop first, without prior reads
		if err := wb.write(polRegLock, 1); err != nil {
			return err
		}

		// arm for the next start, the heartbeat completes this if it fails here
		r, err := wb.state()
		if err == nil {
			err = wb.arm(r, wb.target())
		}
		if err != nil {
			wb.log.DEBUG.Println("arm:", err)
		}

		return nil
	}

	r, err := wb.state()
	if err != nil {
		return err
	}

	if err := wb.ensureConfig(r, polCfgCheck); err != nil {
		return err
	}

	// the cap must be in place before unlocking
	if r[polRegCharging] == 0 {
		if err := wb.arm(r, wb.target()); err != nil {
			return err
		}
	}

	if r[polRegLock] != 0 {
		return wb.write(polRegLock, 0)
	}

	return nil
}

// MaxCurrent implements the api.Charger interface
func (wb *Policharger) MaxCurrent(current int64) error {
	if current < polMinCurrent || current > polMaxCurrent {
		return fmt.Errorf("invalid current %d", current)
	}

	wb.mu.Lock()
	defer wb.mu.Unlock()

	amps := uint16(current)
	wb.current = amps

	r, err := wb.state()
	if err != nil {
		return err
	}

	if r[polRegCharging] == 0 {
		return wb.arm(r, amps)
	}

	// regulate with reg 80, reg 44 is only raised to keep its writes rare
	if r[polRegSetpoint] != amps {
		wb.armed = false
		if err := wb.write(polRegSetpoint, amps); err != nil {
			return err
		}
	}

	if r[polRegManualLimit] < amps {
		wb.armed = false
		return wb.write(polRegManualLimit, amps)
	}

	return nil
}

// chargeCurrent returns the measured charging current
func (wb *Policharger) chargeCurrent() (float64, error) {
	wb.mu.Lock()
	defer wb.mu.Unlock()

	r, err := wb.read(polBlockStatus, polBlockStatusLen)
	if err != nil {
		return 0, err
	}

	// the sensor may show a residual value while not charging
	if r[polRegCharging] == 0 {
		return 0, nil
	}

	return float64(r[polRegCurrent]), nil
}

var _ api.Meter = (*Policharger)(nil)

// CurrentPower implements the api.Meter interface
//
// The charger only reports current (1 A resolution), so power uses the nominal voltage.
func (wb *Policharger) CurrentPower() (float64, error) {
	i, err := wb.chargeCurrent()
	return grid.CurrentToPower(i, wb.phases), err
}

var _ api.PhaseCurrents = (*Policharger)(nil)

// Currents implements the api.PhaseCurrents interface
func (wb *Policharger) Currents() (float64, float64, float64, error) {
	i, err := wb.chargeCurrent()
	if wb.phases == 3 {
		return i, i, i, err
	}
	return i, 0, 0, err
}

var _ api.Diagnosis = (*Policharger)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *Policharger) Diagnose() {
	wb.mu.Lock()
	defer wb.mu.Unlock()

	wb.cache.Clear()

	if id, err := wb.read(polRegModel, 4); err == nil {
		fmt.Printf("Model: %d, connector: %d, software: %d.%02d\n",
			id[polRegModel], id[polRegConnector], id[polRegSwVersion], id[polRegSwVersion+1])
	}

	if cfg, err := wb.read(polRegLockType, 1); err == nil {
		fmt.Printf("Lock type (reg %d): %d (1 = normally locked)\n", polRegLockType, cfg[polRegLockType])
	}

	if pv, err := wb.read(polRegPvGridLimit, 1); err == nil {
		fmt.Printf("PV grid current limit (reg %d): %d A\n", polRegPvGridLimit, pv[polRegPvGridLimit])
	}

	if r, err := wb.state(); err == nil {
		fmt.Printf("Charging: %d, connected: %d/%d, current: %d A\n",
			r[polRegCharging], r[polRegPinConnected], r[polRegConnected], r[polRegCurrent])
		fmt.Printf("Lock (reg %d): %d, setpoint (reg %d): %d A, cap (reg %d): %d A, PV licence: %d\n",
			polRegLock, r[polRegLock], polRegSetpoint, r[polRegSetpoint], polRegManualLimit, r[polRegManualLimit], r[polRegPvProgram])
	}
}
