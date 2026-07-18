package plugin

// modbushttp: generic transport for devices that only expose Modbus-RTU
// tunneled through a vendor-specific HTTP/JSON endpoint (instead of real
// Modbus TCP/RTU). Originally built for the Solplanet/AISWEI "AI dongle"
// (fdbg.cgi on port 8484), but the framing/CRC/HTTP logic is generic and
// should work for any device using the same "raw RTU frame as hex, POSTed
// as JSON" pattern.
//
// Unlike a fixed set of pre-computed hex commands, this plugin computes the
// Modbus RTU frame (address, function code, register, value, CRC16) at
// runtime, so it supports arbitrary values - which is what makes variable
// charge/discharge power control possible.

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/evcc-io/evcc/util"
)

// modbus function codes we support
const (
	fcReadHoldingRegisters byte = 0x03
	fcWriteSingleRegister  byte = 0x06
)

// decode/encode types for the register value
type registerType string

const (
	typeUint16 registerType = "uint16"
	typeInt16  registerType = "int16"
)

// Register describes a single Modbus holding register on the device.
type Register struct {
	Address uint16       `mapstructure:"address"` // plain register address, e.g. 1152 (NOT the 4xxxx Modicon notation)
	Decode  registerType `mapstructure:"decode"`  // uint16 (default) or int16
	Scale   float64      `mapstructure:"scale"`   // applied before encoding / after decoding, default 1
}

// ModbusHttp is the modbushttp plugin.
type ModbusHttp struct {
	*http.Client
	log         *util.Logger
	uri         string // e.g. http://192.168.1.50:8484/fdbg.cgi
	host        string // scheme+host+port, used as rate-limit key
	slaveID     byte
	register    Register
	timeout     time.Duration // overall budget for FloatGetter/FloatSetter, incl. retries
	minInterval time.Duration
}

func init() {
	registry.AddCtx("modbushttp", NewModbusHttpFromConfig)
}

// modbusHttpConfig is the raw YAML/mapstructure configuration.
type modbusHttpConfig struct {
	URI            string        `mapstructure:"uri"` // http://<host>:<port>/fdbg.cgi
	ID             uint8         `mapstructure:"id"`  // Modbus slave id, evcc convention default 1
	Register       Register      `mapstructure:"register"`
	RequestTimeout time.Duration `mapstructure:"requestTimeout"` // per single HTTP request, default 5s
	Timeout        time.Duration `mapstructure:"timeout"`        // overall budget incl. retries, default 30s
	MinInterval    time.Duration `mapstructure:"minInterval"`    // minimum spacing between requests to the same dongle, default 2s
}

// NewModbusHttpFromConfig creates a modbushttp plugin.
func NewModbusHttpFromConfig(ctx context.Context, other map[string]any) (Plugin, error) {
	var cc modbusHttpConfig
	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.URI == "" {
		return nil, fmt.Errorf("modbushttp: uri is required")
	}
	if cc.ID == 0 {
		cc.ID = 1
	}
	if cc.Register.Scale == 0 {
		cc.Register.Scale = 1
	}
	if cc.RequestTimeout == 0 {
		cc.RequestTimeout = 20 * time.Second // observed real-world responses take up to ~11s; give more headroom to reduce retry frequency
	}
	if cc.Timeout == 0 {
		cc.Timeout = 50 * time.Second // covers: request (up to 20s) + short pause (3s) + retry (up to 20s) + margin
	}
	if cc.MinInterval == 0 {
		cc.MinInterval = 5 * time.Second // community reports: this dongle needs ~5s between any requests
	}

	log := util.ContextLoggerWithDefault(ctx, util.NewLogger("modbushttp"))

	m := &ModbusHttp{
		Client:      &http.Client{Timeout: cc.RequestTimeout},
		log:         log,
		uri:         cc.URI,
		host:        hostKey(cc.URI),
		slaveID:     cc.ID,
		register:    cc.Register,
		timeout:     cc.Timeout,
		minInterval: cc.MinInterval,
	}

	return m, nil
}

// --- rate limiting ---
//
// The Solplanet AI dongle (and likely similar ESP32-based dongles using the
// same fdbg.cgi pattern) is fragile under concurrent/rapid requests: multiple
// field reports describe the dongle becoming unresponsive for several
// minutes after being hit with requests in short succession - both via
// fdbg.cgi and the getdevdata.cgi JSON endpoint used for reads elsewhere in
// the template. Since evcc may run several modbushttp instances against the
// same dongle (e.g. one for the mode register, one for the setpoint
// register, invoked back-to-back from a "sequence" plugin), we serialize and
// throttle requests per host so the effective request rate to a given dongle
// never exceeds one request per minInterval, regardless of how many
// registers/instances are configured against it.

var dongleGate = struct {
	mu   sync.Mutex
	last map[string]time.Time
}{last: make(map[string]time.Time)}

// throttle blocks until at least minInterval has passed since the last
// request to the same host, then reserves the current slot. host should be
// the scheme+host+port part of the URI (e.g. "http://192.168.1.50:8484") so
// that all registers/instances talking to the same physical dongle share the
// same budget.
func throttle(ctx context.Context, host string, minInterval time.Duration) error {
	if minInterval <= 0 {
		return nil
	}

	for {
		dongleGate.mu.Lock()
		last, ok := dongleGate.last[host]
		now := time.Now()
		wait := time.Duration(0)
		if ok {
			if elapsed := now.Sub(last); elapsed < minInterval {
				wait = minInterval - elapsed
			}
		}
		if wait == 0 {
			dongleGate.last[host] = now
			dongleGate.mu.Unlock()
			return nil
		}
		dongleGate.mu.Unlock()

		select {
		case <-time.After(wait):
			// retry - another goroutine may have taken the slot meanwhile
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// hostKey extracts scheme+host+port from a full URI, e.g.
// "http://192.168.1.50:8484/fdbg.cgi" -> "http://192.168.1.50:8484"
func hostKey(rawURI string) string {
	u, err := url.Parse(rawURI)
	if err != nil {
		return rawURI // fall back to full URI - still correct, just less shared
	}
	return u.Scheme + "://" + u.Host
}

func crc16modbus(data []byte) uint16 {
	var crc uint16 = 0xFFFF
	for _, b := range data {
		crc ^= uint16(b)
		for i := 0; i < 8; i++ {
			if crc&1 != 0 {
				crc = (crc >> 1) ^ 0xA001
			} else {
				crc >>= 1
			}
		}
	}
	return crc
}

// frame builds a full Modbus RTU frame (incl. CRC) as raw bytes.
func (m *ModbusHttp) frame(fc byte, address uint16, value uint16) []byte {
	buf := make([]byte, 0, 8)
	buf = append(buf, m.slaveID, fc)
	buf = append(buf, byte(address>>8), byte(address&0xff))
	buf = append(buf, byte(value>>8), byte(value&0xff))

	crc := crc16modbus(buf)
	// CRC is transmitted low byte first
	buf = append(buf, byte(crc&0xff), byte(crc>>8))

	return buf
}

// encode converts a float64 evcc value into the raw uint16 register payload,
// respecting sign (S16) and scale.
func encodeValue(v float64, scale float64, decode registerType) uint16 {
	scaled := v / scale
	if decode == typeInt16 {
		return uint16(int16(scaled))
	}
	return uint16(scaled)
}

// decodeValue converts a raw uint16 register payload back into a float64,
// respecting sign (S16) and scale.
func decodeValue(raw uint16, scale float64, decode registerType) float64 {
	if decode == typeInt16 {
		return float64(int16(raw)) * scale
	}
	return float64(raw) * scale
}

// dongleRequest is the JSON body the fdbg.cgi endpoint expects.
type dongleRequest struct {
	Data string `json:"data"`
}

// dongleResponse is the JSON body the fdbg.cgi endpoint returns.
// The exact shape of the response envelope varies by dongle firmware;
// adjust the field name / unwrapping below once verified against real
// hardware. What matters is that .data (or similar) contains the hex
// string of the raw Modbus RTU response frame, mirroring the request.
type dongleResponse struct {
	Data string `json:"data"`
}

// post sends a raw modbus RTU frame to the dongle and returns the raw
// response frame bytes. Respects the per-host rate limit (see throttle)
// before actually issuing the HTTP request.
//
// The dongle is a weak, cloud-first ESP32. Manual testing (Talend API
// Tester) shows that even a single, uncontended request can legitimately
// take up to ~11s to complete - this is normal for this hardware, not a
// sign of overload. requestTimeout is set generously above that observed
// ceiling so we don't cut off responses that are simply slow rather than
// lost. If a request still fails within that generous budget, a short
// pause before the single retry is enough to let a brief collision with
// other traffic clear, without adding excessive extra wait time on top of
// the already-generous per-request timeout.
const (
	maxRetries     = 2
	retryBaseDelay = 3 * time.Second
)

func (m *ModbusHttp) post(ctx context.Context, frame []byte) ([]byte, error) {
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			m.log.TRACE.Printf("retry %d/%d after %v (previous error: %v)", attempt+1, maxRetries, retryBaseDelay, lastErr)
			select {
			case <-time.After(retryBaseDelay):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		raw, err := m.doPost(ctx, frame)
		if err == nil {
			return raw, nil
		}
		lastErr = err
	}

	return nil, fmt.Errorf("modbushttp: giving up after %d attempts: %w", maxRetries, lastErr)
}

// doPost performs a single request/response round trip, without retries.
func (m *ModbusHttp) doPost(ctx context.Context, frame []byte) ([]byte, error) {
	if err := throttle(ctx, m.host, m.minInterval); err != nil {
		return nil, fmt.Errorf("modbushttp: rate limit wait: %w", err)
	}

	reqHex := hex.EncodeToString(frame)
	m.log.TRACE.Printf("send %s", reqHex)

	reqBody, err := json.Marshal(dongleRequest{Data: reqHex})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.uri, bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("modbushttp: unexpected status: %d", resp.StatusCode)
	}

	var dr dongleResponse
	if err := json.NewDecoder(resp.Body).Decode(&dr); err != nil {
		return nil, err
	}

	m.log.TRACE.Printf("recv %s", dr.Data)

	raw, err := hex.DecodeString(dr.Data)
	if err != nil {
		return nil, fmt.Errorf("modbushttp: invalid hex response: %w", err)
	}

	return raw, nil
}

// modbus exception codes, see Modbus Application Protocol spec §7
var modbusExceptions = map[byte]string{
	0x01: "illegal function",
	0x02: "illegal data address",
	0x03: "illegal data value",
	0x04: "server device failure",
	0x05: "acknowledge",
	0x06: "server device busy",
	0x0a: "gateway path unavailable",
	0x0b: "gateway target device failed to respond",
}

// checkException returns a descriptive error if the response frame's function
// code has the exception bit (0x80) set, e.g. because a register address is
// invalid or the device rejected the write (wrong mode, out of range, ...).
func checkException(raw []byte) error {
	if len(raw) < 3 {
		return fmt.Errorf("modbushttp: response too short")
	}
	if raw[1]&0x80 == 0 {
		return nil
	}
	name, ok := modbusExceptions[raw[2]]
	if !ok {
		name = fmt.Sprintf("unknown (0x%02x)", raw[2])
	}
	return fmt.Errorf("modbushttp: modbus exception: %s", name)
}

// parseReadResponse extracts the register value from a read-holding-registers
// RTU response frame: [id][fc][bytecount][data...][crc lo][crc hi]
func parseReadResponse(raw []byte) (uint16, error) {
	if err := checkException(raw); err != nil {
		return 0, err
	}
	if len(raw) < 5 {
		return 0, fmt.Errorf("modbushttp: response too short")
	}
	if raw[1] != fcReadHoldingRegisters {
		return 0, fmt.Errorf("modbushttp: unexpected function code 0x%02x", raw[1])
	}
	byteCount := int(raw[2])
	if byteCount < 2 || len(raw) < 3+byteCount {
		return 0, fmt.Errorf("modbushttp: malformed response")
	}
	return uint16(raw[3])<<8 | uint16(raw[4]), nil
}

var _ FloatGetter = (*ModbusHttp)(nil)

// FloatGetter implements the plugin read interface.
func (m *ModbusHttp) FloatGetter() (func() (float64, error), error) {
	return func() (float64, error) {
		ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
		defer cancel()

		f := m.frame(fcReadHoldingRegisters, m.register.Address, 1) // read 1 register
		raw, err := m.post(ctx, f)
		if err != nil {
			return 0, err
		}

		val, err := parseReadResponse(raw)
		if err != nil {
			return 0, err
		}

		return decodeValue(val, m.register.Scale, m.register.Decode), nil
	}, nil
}

var _ IntGetter = (*ModbusHttp)(nil)

// IntGetter wraps FloatGetter, mirroring the pattern in plugin/modbus.go.
func (m *ModbusHttp) IntGetter() (func() (int64, error), error) {
	g, err := m.FloatGetter()
	if err != nil {
		return nil, err
	}

	return func() (int64, error) {
		f, err := g()
		return int64(f), err
	}, nil
}

var _ FloatSetter = (*ModbusHttp)(nil)

// FloatSetter implements the plugin write interface.
// This is what makes variable charge/discharge power possible: the value
// is encoded into the RTU frame and CRC computed fresh on every call,
// instead of relying on pre-baked hex strings for a handful of fixed values.
func (m *ModbusHttp) FloatSetter(param string) (func(float64) error, error) {
	return func(v float64) error {
		ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
		defer cancel()

		payload := encodeValue(v, m.register.Scale, m.register.Decode)
		f := m.frame(fcWriteSingleRegister, m.register.Address, payload)

		raw, err := m.post(ctx, f)
		if err != nil {
			return err
		}
		return checkException(raw)
	}, nil
}

var _ IntSetter = (*ModbusHttp)(nil)

// IntSetter implements the plugin write interface for integer values, e.g.
// enum-like registers such as the battery run mode (2=self-use, 4=custom).
// Mirrors the pattern in plugin/modbus.go (IntSetter wraps FloatSetter).
func (m *ModbusHttp) IntSetter(param string) (func(int64) error, error) {
	set, err := m.FloatSetter(param)
	if err != nil {
		return nil, err
	}

	return func(v int64) error {
		return set(float64(v))
	}, nil
}
