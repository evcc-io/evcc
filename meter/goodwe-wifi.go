package meter

import (
	"encoding/binary"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

type goodWeWifi struct {
	log    *util.Logger
	conn   *net.UDPConn
	family string // "DT" or "HYBRID"
	usage  string
	mu     sync.Mutex
}

func init() {
	registry.Add("goodwe-wifi", NewGoodWeWifiFromConfig)
}

// NewGoodWeWifiFromConfig is the standard entry point used by all modern meters
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

// NewGoodWeWifi contains the real connection and protocol logic
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

	// DT series only supports pv usage
	if g.family == "DT" && (g.usage == "battery" || g.usage == "grid") {
		return nil, fmt.Errorf("usage '%s' is not supported on DT/DNS series (only 'pv' is valid)", g.usage)
	}

	return g, nil
}

func (g *goodWeWifi) Close() error {
	return g.conn.Close()
}

// modbusCRC16 – exact match to Marcel Blijleven's library
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

// sendCommand – locked + fresh deadline
func (g *goodWeWifi) sendCommand(pdu []byte) ([]byte, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if len(pdu) != 6 {
		return nil, fmt.Errorf("invalid PDU length")
	}

	packet := append(pdu, modbusCRC(pdu)...)

	_, err := g.conn.Write(packet)
	if err != nil {
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

	if n < 6 || buf[0] != 0xaa || buf[1] != 0x55 || buf[2] != 0x7f || buf[3] != 0x03 {
		return nil, fmt.Errorf("invalid response header")
	}

	byteCount := int(buf[4])
	if n < 5+byteCount+2 {
		return nil, fmt.Errorf("short response")
	}

	return buf[5 : 5+byteCount], nil
}

// detectFamily
func (g *goodWeWifi) detectFamily() error {
	pdu := []byte{0x7f, 0x03, 0x9c, 0xed, 0x00, 0x08}
	data, err := g.sendCommand(pdu)
	if err != nil {
		return err
	}
	if len(data) < 16 {
		return fmt.Errorf("short model data")
	}

	model := strings.TrimSpace(string(data[0:16]))
	switch {
	case strings.Contains(model, "DNS") || strings.Contains(model, "DT"):
		g.family = "DT"
	case strings.Contains(model, "ET") || strings.Contains(model, "EH") ||
		strings.Contains(model, "BT") || strings.Contains(model, "BH"):
		g.family = "HYBRID"
	default:
		return fmt.Errorf("unknown model: %s", model)
	}
	g.log.DEBUG.Printf("detected GoodWe family: %s", g.family)
	return nil
}

// CurrentPower implements api.Meter
func (g *goodWeWifi) CurrentPower() (float64, error) {
	var pdu []byte
	var offset int

	if g.family == "DT" {
		pdu = []byte{0x7f, 0x03, 0x75, 0x94, 0x00, 0x49}
		offset = 54 // total inverter power
	} else {
		// HYBRID – different registers depending on usage
		switch g.usage {
		case "pv":
			pdu = []byte{0x7f, 0x03, 0x75, 0x00, 0x00, 0x2a}
			offset = 12
		case "grid":
			pdu = []byte{0x7f, 0x03, 0x75, 0x00, 0x00, 0x2a}
			offset = 24 // grid power (negative = export)
		case "battery":
			pdu = []byte{0x7f, 0x03, 0x75, 0x00, 0x00, 0x2a}
			offset = 36 // battery power
		default:
			return 0, fmt.Errorf("unknown usage: %s", g.usage)
		}
	}

	data, err := g.sendCommand(pdu)
	if err != nil {
		return 0, err
	}
	if len(data) < offset+4 {
		return 0, fmt.Errorf("short runtime data")
	}

	return float64(int32(binary.BigEndian.Uint32(data[offset : offset+4]))), nil
}

// TotalEnergy implements api.MeterEnergy
func (g *goodWeWifi) TotalEnergy() (float64, error) {
	pdu := []byte{0x7f, 0x03, 0x75, 0x94, 0x00, 0x49}
	data, err := g.sendCommand(pdu)
	if err != nil {
		return 0, err
	}
	if len(data) < 94 {
		return 0, fmt.Errorf("short runtime data")
	}
	return float64(binary.BigEndian.Uint32(data[90:94])) / 10.0, nil
}

// Soc implements api.Battery (only available on HYBRID)
func (g *goodWeWifi) Soc() (float64, error) {
	if g.family != "HYBRID" {
		return 0, fmt.Errorf("battery not supported on DT family")
	}

	pdu := []byte{0x7f, 0x03, 0x75, 0x00, 0x00, 0x2a}
	data, err := g.sendCommand(pdu)
	if err != nil {
		return 0, err
	}
	if len(data) < 30 {
		return 0, fmt.Errorf("short battery data")
	}

	soc := float64(binary.BigEndian.Uint16(data[28:30])) // typical SoC location
	return soc, nil
}

// Capacity implements api.BatteryCapacity (only available on HYBRID)
func (g *goodWeWifi) Capacity() (float64, error) {
	if g.family != "HYBRID" {
		return 0, fmt.Errorf("battery not supported on DT family")
	}

	// Many hybrids do not expose nominal capacity via the WiFi protocol.
	// Return 0 for now (user can override in some UIs or we can extend later).
	return 0, nil
}
