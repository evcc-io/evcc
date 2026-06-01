// Package evsemaster implements the binary UDP protocol used by EVSE Master
// compatible charging stations (tested on Sync EV and generic EVSE Master devices).
// Protocol reverse-engineered from https://github.com/johnwoo-nl/emproto
package evsemaster

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net"
	"time"
)

const (
	// Port is the default EVSE Master UDP port
	Port = 28376

	packetHeader = uint16(0x0601)
	packetTail   = uint16(0x0f02)
	headerSize   = 25 // bytes: hdr(2)+len(2)+keytype(1)+serial(8)+passwd(6)+cmd(2)+csum(2)+tail(2)

	// Commands sent by App → EVSE
	CmdRequestLogin = uint16(0x8002) // Send password to request login
	CmdLoginConfirm = uint16(0x8001) // Confirm successful login
	CmdHeadingResp  = uint16(0x8003) // Respond to EVSE keepalive
	CmdStatusAck    = uint16(0x8004) // Acknowledge SingleACStatus
	CmdChargingAck  = uint16(0x0006) // Acknowledge charging session status
	CmdChargeStart  = uint16(0x8007) // Start charging
	CmdChargeStop   = uint16(0x8008) // Stop charging
	CmdSetCurrent   = uint16(0x8107) // Set maximum output current
	CmdHeading      = uint16(0x0003) // Trigger EVSE to start pushing status

	// Commands received from EVSE → App
	CmdLoginBroadcast  = uint16(0x0001) // EVSE discovery broadcast (also carries device info)
	CmdLoginResp       = uint16(0x0002) // Password accepted (in response to RequestLogin)
	CmdHeadingFromEVSE = uint16(0x0003) // EVSE keepalive ping (same wire code as CmdHeading)
	CmdACStatus        = uint16(0x0004) // Real-time charger status
	CmdChargeStatus    = uint16(0x0005) // Charging session status
	CmdPasswordError   = uint16(0x0155) // Wrong password
	CmdSetCurrentResp  = uint16(0x0107) // Confirmation of current change
)

// Packet is a decoded EVSE Master UDP datagram.
// The wire format is:
//
//	hdr(2) | total_len(2) | keytype(1) | serial(8) | passwd(6) | cmd(2) | payload(N) | checksum(2) | tail(2)
type Packet struct {
	Serial   string // 16-char lowercase hex string representing 8 serial bytes
	Password string // up to 6 ASCII characters
	Command  uint16
	Payload  []byte
}

// ReceivedPacket wraps a decoded packet together with its UDP source address.
type ReceivedPacket struct {
	*Packet
	From *net.UDPAddr
}

// Pack serialises the packet to a UDP payload.
func (p *Packet) Pack() ([]byte, error) {
	serialBytes, err := hex.DecodeString(p.Serial)
	if err != nil || len(serialBytes) != 8 {
		return nil, fmt.Errorf("invalid serial %q: must be 16-char hex", p.Serial)
	}

	size := headerSize + len(p.Payload)
	buf := make([]byte, size)

	binary.BigEndian.PutUint16(buf[0:], packetHeader)
	binary.BigEndian.PutUint16(buf[2:], uint16(size))
	buf[4] = 0x00 // key_type
	copy(buf[5:13], serialBytes)

	pw := []byte(p.Password)
	if len(pw) > 6 {
		pw = pw[:6]
	}
	copy(buf[13:19], pw)

	binary.BigEndian.PutUint16(buf[19:], p.Command)
	copy(buf[21:], p.Payload)

	var checksum uint32
	for _, b := range buf[:size-4] {
		checksum += uint32(b)
	}
	binary.BigEndian.PutUint16(buf[size-4:], uint16(checksum%0xFFFF))
	binary.BigEndian.PutUint16(buf[size-2:], packetTail)

	return buf, nil
}

// Unpack deserialises a packet from raw UDP bytes.
func Unpack(buf []byte) (*Packet, error) {
	if len(buf) < headerSize {
		return nil, fmt.Errorf("packet too short (%d bytes)", len(buf))
	}
	if binary.BigEndian.Uint16(buf[0:]) != packetHeader {
		return nil, fmt.Errorf("invalid packet header")
	}

	totalLen := int(binary.BigEndian.Uint16(buf[2:]))
	if totalLen > len(buf) || totalLen < headerSize {
		return nil, fmt.Errorf("invalid length field: %d", totalLen)
	}

	// Verify checksum
	var sum uint32
	for _, b := range buf[:totalLen-4] {
		sum += uint32(b)
	}
	if uint16(sum%0xFFFF) != binary.BigEndian.Uint16(buf[totalLen-4:]) {
		return nil, fmt.Errorf("checksum mismatch")
	}
	if binary.BigEndian.Uint16(buf[totalLen-2:]) != packetTail {
		return nil, fmt.Errorf("invalid packet tail")
	}

	p := &Packet{
		Serial:  hex.EncodeToString(buf[5:13]),
		Command: binary.BigEndian.Uint16(buf[19:]),
	}

	// Trim null bytes from password field
	var pw [6]byte
	copy(pw[:], buf[13:19])
	end := 6
	for i, b := range pw {
		if b == 0 {
			end = i
			break
		}
	}
	p.Password = string(pw[:end])

	payloadLen := totalLen - headerSize
	if payloadLen > 0 {
		p.Payload = make([]byte, payloadLen)
		copy(p.Payload, buf[21:21+payloadLen])
	}

	return p, nil
}

// ACStatus holds real-time charger state from command 0x0004 (SingleACStatus).
// GunState: 0=unknown, 1=disconnected, 2=connected_unlocked, 3=negotiating, 4=connected_locked
// OutputState: 0=idle, 1=charging
type ACStatus struct {
	GunState    int
	OutputState int
	Power       float64 // W
	TotalEnergy float64 // kWh (lifetime total)
	L1Voltage   float64 // V
	L1Current   float64 // A
	L2Voltage   float64 // V
	L2Current   float64 // A
	L3Voltage   float64 // V
	L3Current   float64 // A
}

// ParseACStatus decodes the payload of command 0x0004.
func ParseACStatus(payload []byte) (*ACStatus, error) {
	if len(payload) < 25 {
		return nil, fmt.Errorf("ACStatus payload too short: %d bytes", len(payload))
	}
	s := &ACStatus{
		L1Voltage:   float64(binary.BigEndian.Uint16(payload[1:])) * 0.1,
		L1Current:   float64(binary.BigEndian.Uint16(payload[3:])) * 0.01,
		Power:       float64(binary.BigEndian.Uint32(payload[5:])),
		TotalEnergy: float64(binary.BigEndian.Uint32(payload[9:])) * 0.01,
		GunState:    int(payload[18]),
		OutputState: int(payload[19]),
	}

	if len(payload) >= 33 {
		s.L2Voltage = float64(binary.BigEndian.Uint16(payload[25:])) * 0.1
		s.L2Current = float64(binary.BigEndian.Uint16(payload[27:])) * 0.01
		s.L3Voltage = float64(binary.BigEndian.Uint16(payload[29:])) * 0.1
		s.L3Current = float64(binary.BigEndian.Uint16(payload[31:])) * 0.01
	}

	return s, nil
}

// PackChargeStart builds the 47-byte payload for command 0x8007 (ChargeStart).
// maxAmps must be in the range 6–32.
func PackChargeStart(maxAmps int) ([]byte, error) {
	if maxAmps < 6 || maxAmps > 32 {
		return nil, fmt.Errorf("maxAmps must be 6-32, got %d", maxAmps)
	}
	buf := make([]byte, 47)
	buf[0] = 1 // line_id

	// user_id (16 bytes, null-padded) – same default as the mobile app
	copy(buf[1:17], []byte("emmgr"))

	// charge_id (16 bytes, null-padded) – use current Unix timestamp as unique ID
	ts := fmt.Sprintf("%d", time.Now().Unix())
	if len(ts) > 16 {
		ts = ts[:16]
	}
	copy(buf[17:33], []byte(ts))

	buf[33] = 0 // not a reservation (immediate start)
	binary.BigEndian.PutUint32(buf[34:], uint32(time.Now().Unix()))
	buf[38] = 1                                  // start_type
	buf[39] = 1                                  // charge_type
	binary.BigEndian.PutUint16(buf[40:], 0xFFFF) // max_duration = unlimited
	binary.BigEndian.PutUint16(buf[42:], 0xFFFF) // max_energy = unlimited
	binary.BigEndian.PutUint16(buf[44:], 0xFFFF) // param3
	buf[46] = byte(maxAmps)

	return buf, nil
}

// PackSetCurrent builds the 2-byte payload for command 0x8107 (SetAndGetOutputElectricity).
// amps must be in the range 6–32.
func PackSetCurrent(amps int) []byte {
	return []byte{0x01, byte(amps)} // action=SET, value
}
