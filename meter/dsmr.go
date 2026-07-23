package meter

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/implement"
	"github.com/evcc-io/evcc/meter/obis"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

// Dsmr is a DSMR P1 meter. The raw P1 byte stream is provided by a pluggable
// transport: a raw TCP socket for classic P1-to-LAN gateways, or a WebSocket.
type Dsmr struct {
	implement.Caps
	mu      sync.Mutex
	log     *util.Logger
	dial    func() (io.ReadCloser, error)
	timeout time.Duration
	frame   map[string]string
	updated time.Time
	conn    io.ReadCloser
}

var (
	currentObis     = []string{obis.CurrentL1, obis.CurrentL2, obis.CurrentL3}
	voltageObis     = []string{obis.VoltageL1, obis.VoltageL2, obis.VoltageL3}
	powerImportObis = []string{obis.PowerImportL1, obis.PowerImportL2, obis.PowerImportL3}
	powerExportObis = []string{obis.PowerExportL1, obis.PowerExportL2, obis.PowerExportL3}
)

// objectRegexp matches a DSMR P1 object: OBIS reduced ID-code
// and its first value group, e.g. `1-0:1.7.0(00.330*kW)`.
var objectRegexp = regexp.MustCompile(`([0-9]+-[0-9]+:[0-9]+\.[0-9]+\.[0-9]+)\(([^)]*)\)`)

func init() {
	registry.AddCtx("dsmr", NewDsmrFromConfig)
}

// NewDsmrFromConfig creates a DSMR meter from generic config
func NewDsmrFromConfig(ctx context.Context, other map[string]any) (api.Meter, error) {
	cc := struct {
		URI     string
		Energy  string // TODO deprecated
		Timeout time.Duration
	}{
		Timeout: 15 * time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewDsmr(ctx, cc.URI, cc.Timeout)
}

// NewDsmr creates a DSMR meter. The transport is selected from the uri scheme:
// ws:// or wss:// uses a WebSocket, anything else a raw TCP socket (host:port).
func NewDsmr(ctx context.Context, uri string, timeout time.Duration) (api.Meter, error) {
	dial := dsmrDialer(ctx, uri)

	m, err := newDsmr(ctx, util.NewLogger("dsmr"), dial, timeout)
	if err != nil {
		return nil, err
	}

	m.decorateEnergy()
	m.decorateCurrents()
	m.decorateVoltages()
	m.decoratePowers()

	return m, nil
}

// dsmrDialer returns a transport dialer for the given uri.
func dsmrDialer(ctx context.Context, uri string) func() (io.ReadCloser, error) {
	if strings.HasPrefix(uri, "ws://") || strings.HasPrefix(uri, "wss://") {
		return wsDialer(ctx, uri)
	}

	return func() (io.ReadCloser, error) {
		dialer := net.Dialer{Timeout: request.Timeout}
		conn, err := dialer.DialContext(ctx, "tcp", uri)
		if err != nil {
			return nil, err
		}

		return conn, nil
	}
}

// newDsmr starts the read loop over the given transport and blocks until the
// first valid frame so callers can probe the available registers.
func newDsmr(ctx context.Context, log *util.Logger, dial func() (io.ReadCloser, error), timeout time.Duration) (*Dsmr, error) {
	conn, err := dial()
	if err != nil {
		return nil, err
	}

	m := &Dsmr{
		Caps:    implement.New(),
		log:     log,
		dial:    dial,
		timeout: timeout,
	}
	m.setConn(conn)

	// close the active connection when ctx is canceled, unblocking a pending
	// read so run observes ctx and returns
	go func() {
		<-ctx.Done()
		m.mu.Lock()
		if m.conn != nil {
			m.conn.Close()
		}
		m.mu.Unlock()
	}()

	done := make(chan struct{}, 1)
	go m.run(ctx, conn, done)

	// wait for initial value
	select {
	case <-done:
		return m, nil
	case <-time.After(timeout):
		return nil, os.ErrDeadlineExceeded
	}
}

// parseFrame maps each OBIS code in a P1 telegram to its raw value with the
// unit suffix stripped, e.g. `000084.276*kWh` -> `000084.276`.
func parseFrame(frame string) map[string]string {
	objects := make(map[string]string)
	for _, m := range objectRegexp.FindAllStringSubmatch(frame, -1) {
		value, _, _ := strings.Cut(m[2], "*")
		objects[m[1]] = value
	}

	return objects
}

// crc16ARC computes the CRC-16/ARC checksum (reflected, polynomial 0xA001,
// init 0x0000) used to verify DSMR P1 telegrams.
func crc16ARC(data []byte) uint16 {
	var crc uint16
	for _, b := range data {
		crc ^= uint16(b)
		for range 8 {
			if crc&1 != 0 {
				crc = crc>>1 ^ 0xA001
			} else {
				crc >>= 1
			}
		}
	}

	return crc
}

// setConn publishes the active connection so it can be closed on ctx cancel.
func (m *Dsmr) setConn(conn io.ReadCloser) {
	m.mu.Lock()
	m.conn = conn
	m.mu.Unlock()
}

func (m *Dsmr) run(ctx context.Context, conn io.ReadCloser, done chan struct{}) {
	bo := backoff.NewExponentialBackOff(backoff.WithMaxInterval(5*time.Minute), backoff.WithMaxElapsedTime(0))

	reader := bufio.NewReader(conn)

	// close whatever connection is current when the loop exits
	defer func() {
		if conn != nil {
			conn.Close()
		}
	}()

	for {
		if ctx.Err() != nil {
			return
		}

		if conn == nil {
			var err error
			if conn, err = m.dial(); err != nil {
				m.log.ERROR.Printf("connect: %v", err)
				select {
				case <-time.After(max(bo.NextBackOff(), time.Second)):
				case <-ctx.Done():
					return
				}
				continue
			}
			m.setConn(conn)
			reader.Reset(conn)
			bo.Reset()
		}

		objects, err := m.readFrame(reader)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			m.log.ERROR.Printf("read: %v", err)
			conn.Close()
			conn = nil
			continue
		}

		m.mu.Lock()
		m.frame = objects
		m.updated = time.Now()
		m.mu.Unlock()

		select {
		case done <- struct{}{}:
		default:
		}
	}
}

// readFrame consumes a single P1 telegram and returns its parsed OBIS values.
// CRC and parse errors are logged and skipped; it only returns on a transport
// error (to trigger a reconnect).
func (m *Dsmr) readFrame(reader *bufio.Reader) (map[string]string, error) {
	for {
		b, err := reader.Peek(1)
		if err != nil {
			return nil, err
		}
		if b[0] != '/' {
			m.log.DEBUG.Printf("ignoring garbage character: %c", b[0])
			_, _ = reader.ReadByte()
			continue
		}

		frame, err := reader.ReadBytes('!')
		if err != nil {
			return nil, err
		}

		bcrc, err := reader.ReadBytes('\n')
		if err != nil {
			return nil, err
		}

		m.log.TRACE.Printf("read: %s", frame)

		// Check CRC
		mcrc := strings.ToUpper(strings.TrimSpace(string(bcrc)))
		if crc := fmt.Sprintf("%04X", crc16ARC(frame)); mcrc != crc {
			m.log.ERROR.Printf("crc mismatch: %q != %q", mcrc, crc)
			continue
		}

		return parseFrame(string(frame)), nil
	}
}

// hasObis reports whether every given OBIS code exists in the last frame
func (m *Dsmr) hasObis(obis ...string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, o := range obis {
		if _, ok := m.frame[o]; !ok {
			return false
		}
	}

	return true
}

// decorateEnergy registers MeterEnergy/MeterReturnEnergy when import/export
// energy is available, either as the combined register or summed tariffs.
func (m *Dsmr) decorateEnergy() {
	if fn := m.energyFunc(obis.EnergyImport, obis.EnergyImportT1, obis.EnergyImportT2); fn != nil {
		implement.Has(m, implement.MeterEnergy(fn))
	}

	if fn := m.energyFunc(obis.EnergyExport, obis.EnergyExportT1, obis.EnergyExportT2); fn != nil {
		implement.Has(m, implement.MeterReturnEnergy(fn))
	}
}

// energyFunc returns an accessor for total energy: the combined register if
// present, otherwise the sum of whichever tariff registers are available, or
// nil if none are present.
func (m *Dsmr) energyFunc(total string, tariffs ...string) func() (float64, error) {
	if m.hasObis(total) {
		return func() (float64, error) {
			return m.get(total)
		}
	}

	var present []string
	for _, t := range tariffs {
		if m.hasObis(t) {
			present = append(present, t)
		}
	}

	if len(present) == 0 {
		return nil
	}

	return func() (float64, error) {
		return m.sum(present...)
	}
}

// decorateCurrents registers PhaseCurrents only when all three phases are present.
func (m *Dsmr) decorateCurrents() {
	if m.hasObis(currentObis...) {
		implement.Has(m, implement.PhaseCurrents(m.currents))
	}
}

// decorateVoltages registers PhaseVoltages only when all three phases are present.
func (m *Dsmr) decorateVoltages() {
	if m.hasObis(voltageObis...) {
		implement.Has(m, implement.PhaseVoltages(m.voltages))
	}
}

// decoratePowers registers PhasePowers only when per-phase import and export
// power are present for all three phases.
func (m *Dsmr) decoratePowers() {
	if m.hasObis(powerImportObis...) && m.hasObis(powerExportObis...) {
		implement.Has(m, implement.PhasePowers(m.powers))
	}
}

func (m *Dsmr) get(id string) (float64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if time.Since(m.updated) > m.timeout {
		return 0, os.ErrDeadlineExceeded
	}

	res, ok := m.frame[id]
	if !ok {
		return 0, fmt.Errorf("%w: %s", api.ErrNotAvailable, id)
	}

	return strconv.ParseFloat(res, 64)
}

// sum adds the values of the given OBIS codes, failing if any is unavailable.
func (m *Dsmr) sum(obis ...string) (float64, error) {
	var total float64
	for _, o := range obis {
		f, err := m.get(o)
		if err != nil {
			return 0, err
		}
		total += f
	}

	return total, nil
}

// totalOrSum returns the total OBIS value, falling back to the sum of the given
// per-phase OBIS codes when the total is not available.
func (m *Dsmr) totalOrSum(total string, phases ...string) (float64, error) {
	v, err := m.get(total)
	if !errors.Is(err, api.ErrNotAvailable) {
		return v, err
	}

	if sum, sumErr := m.sum(phases...); sumErr == nil {
		return sum, nil
	}

	return 0, err
}

// CurrentPower implements the api.Meter interface
func (m *Dsmr) CurrentPower() (float64, error) {
	importPower, err := m.totalOrSum(obis.PowerImport, obis.PowerImportL1, obis.PowerImportL2, obis.PowerImportL3)
	if err != nil {
		return 0, err
	}

	// export is optional
	exportPower, err := m.totalOrSum(obis.PowerExport, obis.PowerExportL1, obis.PowerExportL2, obis.PowerExportL3)
	if err != nil && !errors.Is(err, api.ErrNotAvailable) {
		return 0, err
	}

	return (importPower - exportPower) * 1e3, nil
}

// currents implements the api.PhaseCurrents interface
func (m *Dsmr) currents() (float64, float64, float64, error) {
	var res [3]float64

	for i := range res {
		var err error
		if res[i], err = m.get(currentObis[i]); err != nil {
			return 0, 0, 0, err
		}

		// correct import/export sign
		if f, err := m.get(powerExportObis[i]); err != nil {
			return 0, 0, 0, err
		} else if f > 0 {
			res[i] = -res[i]
		}
	}

	return res[0], res[1], res[2], nil
}

// voltages implements the api.PhaseVoltages interface
func (m *Dsmr) voltages() (float64, float64, float64, error) {
	var res [3]float64

	for i := range res {
		var err error
		if res[i], err = m.get(voltageObis[i]); err != nil {
			return 0, 0, 0, err
		}
	}

	return res[0], res[1], res[2], nil
}

// powers implements the api.PhasePowers interface
func (m *Dsmr) powers() (float64, float64, float64, error) {
	var res [3]float64

	for i := range res {
		importPower, err := m.get(powerImportObis[i])
		if err != nil {
			return 0, 0, 0, err
		}

		exportPower, err := m.get(powerExportObis[i])
		if err != nil {
			return 0, 0, 0, err
		}

		res[i] = (importPower - exportPower) * 1e3
	}

	return res[0], res[1], res[2], nil
}
