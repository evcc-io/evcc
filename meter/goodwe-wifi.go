package meter

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

// goodWeWifi supports three protocol families discovered at connect time:
//
//   - "DT"     – DT/DNS string inverters (e.g. GW3000-DNS-30, GW8K-DT)
//     runtime PDU: READ 73 regs @ 0x7594
//     power offset 54 (S32), energy offset 90 (U32 ÷10 → kWh)
//     usage: pv only
//
//   - "HYBRID" – ES/EM single-phase hybrids (e.g. GW5048-EM, GW5048D-ES)
//     runtime PDU: READ 42 regs @ 0x7500
//     pv@12, grid@24, battery@36 (all S32); SoC@28 (U16)
//     usage: pv / grid / battery
//
//   - "ET"     – ET/EH/BT/BH three-phase and HV-battery hybrids
//     (e.g. GW10K-ET, GW25K-ET, GW6000-EH, GW5K-BT)
//     runtime PDU: READ 125 regs @ 0x891C
//     pv@74, grid@78, battery@164 (all S32); energy@182 (U32 ÷10 → kWh)
//     SoC PDU: READ 24 regs @ 0x9088, SoC@14 (U16)
//     usage: pv / grid / battery
type goodWeWifi struct {
	log    *util.Logger
	conn   *net.UDPConn
	family string // "DT", "HYBRID", or "ET"
	usage  string
	mu     sync.Mutex
}

func init() {
	registry.Add("goodwe-wifi", NewGoodWeWifiFromConfig)
}

// NewGoodWeWifiFromConfig is the standard entry point used by the evcc template system.
func NewGoodWeWifiFromConfig(other map[string]interface{}) (api.Meter, error) {
	var cc struct {
		URI   string `mapstructure:"uri"`
		Usage string `mapstructure:"usage"`
	}
	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewGoodWeWifi(cc.URI, cc.Usage)
}

// NewGoodWeWifi contains the real connection and protocol logic.
func NewGoodWeWifi(uri, usage string) (api.Meter, error) {
	log := util.NewLogger("goodwe-wifi").Redact(uri)

	addr, err := net.ResolveUDPAddr("udp4", uri+":8899")
	if err != nil {
		return nil, err
	}

	conn, err := net.DialUDP("udp4", nil, addr)
	if err != nil {
		return nil, err
	}

	g := &goodWeWifi{
		log:   log,
		conn:  conn,
		usage: usage,
	}

	if err := g.detectFamily(); err != nil {
		return nil, fmt.Errorf("family detection failed: %w", err)
	}

	// DT series only supports pv usage (no grid meter, no battery).
	if g.family == "DT" && (g.usage == "battery" || g.usage == "grid") {
		return nil, fmt.Errorf("usage '%s' is not supported on DT/DNS series (only 'pv' is valid)", g.usage)
	}

	return g, nil
}

func (g *goodWeWifi) Close() error {
	return g.conn.Close()
}

// modbusCRC computes the Modbus CRC-16 of data (little-endian byte order).
func modbusCRC(data []byte) []byte {
	crc := uint16(0xFFFF)
	for _, b := range data {
		crc ^= uint16(b)
		for i := 0; i < 8; i++ {
			if crc&0x0001 != 0 {
				crc = (crc >> 1) ^ 0xA001
			} else {
				crc >>= 1
			}
		}
	}
	return []byte{byte(crc & 0xFF), byte(crc >> 8)}
}

// sendCommand sends a 6-byte Modbus PDU (with CRC appended) and returns the
// stripped payload from the AA55-framed response.
func (g *goodWeWifi) sendCommand(pdu []byte) ([]byte, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if len(pdu) != 6 {
		return nil, errors.New("invalid PDU length")
	}

	packet := append(pdu, modbusCRC(pdu)...)

	if _, err := g.conn.Write(packet); err != nil {
		return nil, err
	}

	if err := g.conn.SetReadDeadline(time.Now().Add(4 * time.Second)); err != nil {
		return nil, fmt.Errorf("set read deadline: %w", err)
	}

	buf := make([]byte, 512)
	n, err := g.conn.Read(buf)
	if err != nil {
		return nil, err
	}

	if n < 6 || buf[0] != 0xaa || buf[1] != 0x55 || buf[3] != 0x03 {
		// buf[2] is the inverter's source address, which varies by family:
		//   DT/DNS family:   0x7F
		//   ET/EH/BT/BH:     0xF7
		// Only the AA55 magic bytes and the function code (buf[3]==0x03) are
		// stable across all families, so we don't validate buf[2].
		return nil, errors.New("invalid response header")
	}

	byteCount := int(buf[4])
	if n < 5+byteCount+2 {
		return nil, errors.New("short response")
	}

	return buf[5 : 5+byteCount], nil
}

// detectFamily queries the model string (READ 8 regs @ 0x9CED) and sets
// g.family to "DT", "HYBRID", or "ET".
func (g *goodWeWifi) detectFamily() error {
	pdu := []byte{0x7f, 0x03, 0x9c, 0xed, 0x00, 0x08}
	data, err := g.sendCommand(pdu)
	if err != nil {
		return err
	}
	if len(data) < 16 {
		return errors.New("short model data")
	}

	model := strings.TrimSpace(string(data[0:16]))
	switch {
	case strings.Contains(model, "DNS") || strings.Contains(model, "DT"):
		g.family = "DT"
	case strings.Contains(model, "ES") || strings.Contains(model, "EM"):
		// ES/EM: single-phase hybrids (e.g. GW5048D-ES, GW5048-EM).
		// Runtime PDU: READ 42 regs @ 0x7500 → 84-byte payload.
		g.family = "HYBRID"
	case strings.Contains(model, "ET") || strings.Contains(model, "EH") ||
		strings.Contains(model, "BT") || strings.Contains(model, "BH"):
		// ET/EH/BT/BH: three-phase and HV-battery hybrids.
		// Runtime PDU: READ 125 regs @ 0x891C → 250-byte payload.
		g.family = "ET"
	default:
		return fmt.Errorf("unknown model: %s", model)
	}
	g.log.DEBUG.Printf("detected GoodWe family: %s (model: %s)", g.family, model)
	return nil
}

// CurrentPower implements api.Meter.
//
// DT family   – READ 73 @ 0x7594, total inverter power at offset 54 (S32)
// HYBRID (ES) – READ 42 @ 0x7500, pv/grid/battery at offsets 12/24/36 (S32)
// ET family   – READ 125 @ 0x891C, pv/grid/battery at offsets 74/78/164 (S32)
func (g *goodWeWifi) CurrentPower() (float64, error) {
	switch g.family {
	case "DT":
		// READ 73 regs @ 0x7594 → 146-byte payload
		data, err := g.sendCommand([]byte{0x7f, 0x03, 0x75, 0x94, 0x00, 0x49})
		if err != nil {
			return 0, err
		}
		const offset = 54
		if len(data) < offset+4 {
			return 0, errors.New("short runtime data")
		}
		return float64(int32(binary.BigEndian.Uint32(data[offset : offset+4]))), nil

	case "HYBRID":
		// READ 42 regs @ 0x7500 → 84-byte payload (ES/EM single-phase hybrids)
		data, err := g.sendCommand([]byte{0x7f, 0x03, 0x75, 0x00, 0x00, 0x2a})
		if err != nil {
			return 0, err
		}
		var offset int
		switch g.usage {
		case "pv":
			offset = 12
		case "grid":
			offset = 24 // negative = exporting
		case "battery":
			offset = 36 // negative = charging
		default:
			return 0, fmt.Errorf("unknown usage: %s", g.usage)
		}
		if len(data) < offset+4 {
			return 0, errors.New("short runtime data")
		}
		return float64(int32(binary.BigEndian.Uint32(data[offset : offset+4]))), nil

	case "ET":
		// READ 125 regs @ 0x891C → 250-byte payload (ET/EH/BT/BH hybrids)
		data, err := g.sendCommand([]byte{0x7f, 0x03, 0x89, 0x1c, 0x00, 0x7d})
		if err != nil {
			return 0, err
		}
		var offset int
		switch g.usage {
		case "pv":
			offset = 74 // total_inverter_power (S32)
		case "grid":
			offset = 78 // ac_active_power (S32, negative = exporting)
		case "battery":
			offset = 164 // pbattery1 (S32, negative = charging)
		default:
			return 0, fmt.Errorf("unknown usage: %s", g.usage)
		}
		if len(data) < offset+4 {
			return 0, errors.New("short runtime data")
		}
		return float64(int32(binary.BigEndian.Uint32(data[offset : offset+4]))), nil

	default:
		return 0, fmt.Errorf("unsupported family: %s", g.family)
	}
}

// TotalEnergy implements api.MeterEnergy.
//
// DT family: READ 73 @ 0x7594, e_total at offset 90 (U32 ÷10 → kWh)
// ET family: READ 125 @ 0x891C, e_total at offset 182 (U32 ÷10 → kWh)
// HYBRID (ES/EM): uses the DT PDU as a best-effort fallback — the ES/EM
// WiFi protocol does not expose total energy on the 0x7500 registers.
func (g *goodWeWifi) TotalEnergy() (float64, error) {
	switch g.family {
	case "DT", "HYBRID":
		// DT PDU also happens to work for ES/EM as a best-effort energy source.
		data, err := g.sendCommand([]byte{0x7f, 0x03, 0x75, 0x94, 0x00, 0x49})
		if err != nil {
			return 0, err
		}
		if len(data) < 94 {
			return 0, errors.New("short runtime data")
		}
		return float64(binary.BigEndian.Uint32(data[90:94])) / 10.0, nil

	case "ET":
		// ET runtime data (READ 125 @ 0x891C) carries e_total at offset 182.
		data, err := g.sendCommand([]byte{0x7f, 0x03, 0x89, 0x1c, 0x00, 0x7d})
		if err != nil {
			return 0, err
		}
		if len(data) < 186 {
			return 0, errors.New("short runtime data")
		}
		return float64(binary.BigEndian.Uint32(data[182:186])) / 10.0, nil

	default:
		return 0, fmt.Errorf("unsupported family: %s", g.family)
	}
}

// Soc implements api.Battery.
//
// HYBRID (ES/EM): READ 42 @ 0x7500, SoC at offset 28 (U16, %)
// ET family:      READ 24 @ 0x9088, SoC at offset 14 (U16, %)
// DT family:      not supported (no battery)
func (g *goodWeWifi) Soc() (float64, error) {
	switch g.family {
	case "DT":
		return 0, errors.New("battery SoC not supported on DT/DNS series")

	case "HYBRID":
		data, err := g.sendCommand([]byte{0x7f, 0x03, 0x75, 0x00, 0x00, 0x2a})
		if err != nil {
			return 0, err
		}
		if len(data) < 30 {
			return 0, errors.New("short battery data")
		}
		return float64(binary.BigEndian.Uint16(data[28:30])), nil

	case "ET":
		// Battery info: READ 24 regs @ 0x9088 → 48-byte payload.
		// SoC is at offset 14 (register 37007, U16, %).
		data, err := g.sendCommand([]byte{0x7f, 0x03, 0x90, 0x88, 0x00, 0x18})
		if err != nil {
			return 0, err
		}
		if len(data) < 16 {
			return 0, errors.New("short battery data")
		}
		return float64(binary.BigEndian.Uint16(data[14:16])), nil

	default:
		return 0, fmt.Errorf("unsupported family: %s", g.family)
	}
}

// Capacity implements api.BatteryCapacity.
// Nominal capacity is not exposed via the WiFi protocol on any family.
func (g *goodWeWifi) Capacity() (float64, error) {
	if g.family == "DT" {
		return 0, errors.New("battery capacity not supported on DT/DNS series")
	}
	return 0, nil
}
