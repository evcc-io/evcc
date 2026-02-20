package meter

import (
	"encoding/binary"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

type GoodWeDTWifi struct {
	log   *util.Logger
	conn  *net.UDPConn
	addr  *net.UDPAddr
	model string
}

func init() {
	registry.Add("goodwe-dt-wifi", NewGoodWeDTWifi)
}

func NewGoodWeDTWifi(other map[string]interface{}) (api.Meter, error) {
	var cc struct {
		URI string `mapstructure:"uri"`
	}
	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	log := util.NewLogger("goodwe-dt-wifi").Redact(cc.URI)

	addr, err := net.ResolveUDPAddr("udp4", cc.URI+":8899")
	if err != nil {
		return nil, err
	}

	conn, err := net.DialUDP("udp4", nil, addr)
	if err != nil {
		return nil, err
	}
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	g := &GoodWeDTWifi{
		log:  log,
		conn: conn,
		addr: addr,
	}

	if err := g.detectFamily(); err != nil {
		return nil, fmt.Errorf("family detection failed: %w", err)
	}

	return g, nil
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

// sendCommand sends raw Modbus RTU PDU + CRC, returns register payload
func (g *GoodWeDTWifi) sendCommand(pdu []byte) ([]byte, error) {
	if len(pdu) != 6 {
		return nil, fmt.Errorf("invalid PDU length")
	}
	packet := append(pdu, modbusCRC(pdu)...)

	_, err := g.conn.Write(packet)
	if err != nil {
		return nil, err
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
func (g *GoodWeDTWifi) detectFamily() error {
	pdu := []byte{0x7f, 0x03, 0x9c, 0xed, 0x00, 0x08}
	data, err := g.sendCommand(pdu)
	if err != nil {
		return err
	}
	if len(data) < 16 {
		return fmt.Errorf("short model data")
	}

	model := strings.TrimSpace(string(data[0:16]))
	if strings.Contains(model, "DNS") || strings.Contains(model, "DT") {
		g.model = "DT"
	} else {
		return fmt.Errorf("unknown model: %s", model)
	}
	return nil
}

// CurrentPower implements api.Meter
func (g *GoodWeDTWifi) CurrentPower() (float64, error) {
	pdu := []byte{0x7f, 0x03, 0x75, 0x94, 0x00, 0x49}
	data, err := g.sendCommand(pdu)
	if err != nil {
		return 0, err
	}
	if len(data) < 146 {
		return 0, fmt.Errorf("short runtime data")
	}
	// total_inverter_power (register 30127 → offset 54)
	return float64(int32(binary.BigEndian.Uint32(data[54:58]))), nil
}

// TotalEnergy implements api.MeterEnergy (lifetime in kWh)
func (g *GoodWeDTWifi) TotalEnergy() (float64, error) {
	pdu := []byte{0x7f, 0x03, 0x75, 0x94, 0x00, 0x49}
	data, err := g.sendCommand(pdu)
	if err != nil {
		return 0, err
	}
	if len(data) < 146 {
		return 0, fmt.Errorf("short runtime data")
	}
	// e_total (register 30145 → offset 90, 0.1 kWh)
	return float64(binary.BigEndian.Uint32(data[90:94])) / 10.0, nil
}
