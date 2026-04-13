package danfoss

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// ComLynx message type codes.
const (
	typePingRequest           byte = 0x15
	typePingReply             byte = 0x95
	typeGetNodeInfoRequest    byte = 0x13
	typeGetNodeInfoReply      byte = 0x93
	typeCanRequest            byte = 0x01
	typeCanReply              byte = 0x81
	typeApplicationErrorMask  byte = 0x20 // bit 5
	typeTransmissionErrorMask byte = 0x40 // bit 6
	typeCode                  byte = 0x1f // bits 0-4 carry the message code
)

// TLX hardware module used by the parameter read command. ULX inverters use
// 0x04 instead; we only target TLX here.
const tlxDestinationModule byte = 0x08

// Address is a 2-byte ComLynx node address. Network and subnet fit into the
// first byte (4 bits each); node occupies the second byte.
type Address struct {
	Network byte // 0..15
	Subnet  byte // 0..15
	Node    byte // 0..255
}

// NewAddress constructs an Address. It panics on out-of-range inputs because
// those only come from hard-coded constants or config validation.
func NewAddress(network, subnet, node byte) Address {
	if network > 0x0f || subnet > 0x0f {
		panic("danfoss: address network/subnet must fit in 4 bits")
	}
	return Address{Network: network, Subnet: subnet, Node: node}
}

// Broadcast is the ComLynx broadcast address. Any ComLynx node replies to a
// request targeted at it.
var Broadcast = Address{Network: 0x0f, Subnet: 0x0f, Node: 0xff}

// DefaultSource is the source address evcc advertises on the bus. Matches the
// hard-coded value used by AMajland/Danfoss-TLX, known to be accepted by TLX
// inverters.
var DefaultSource = Address{Network: 0x00, Subnet: 0x00, Node: 0x02}

func (a Address) bytes() [2]byte {
	return [2]byte{a.Network<<4 | a.Subnet, a.Node}
}

func addressFromBytes(first, second byte) Address {
	return Address{Network: first >> 4, Subnet: first & 0x0f, Node: second}
}

func (a Address) String() string {
	return fmt.Sprintf("%x-%x-%02x", a.Network, a.Subnet, a.Node)
}

// Equal reports whether two addresses are identical.
func (a Address) Equal(b Address) bool {
	return a.Network == b.Network && a.Subnet == b.Subnet && a.Node == b.Node
}

// encodeMessage builds the body of an HDLC frame (before FCS and flags):
//
//	FF 03 src(2) dst(2) len type data...
//
// The length byte counts the data payload only, not the type byte.
func encodeMessage(src, dst Address, msgType byte, data []byte) []byte {
	if len(data) > 0xff {
		panic("danfoss: message data exceeds 255 bytes")
	}
	srcB := src.bytes()
	dstB := dst.bytes()
	body := make([]byte, 0, 8+len(data))
	body = append(body,
		broadcast, unnumberedInfo,
		srcB[0], srcB[1],
		dstB[0], dstB[1],
		byte(len(data)), msgType,
	)
	body = append(body, data...)
	return body
}

// decodedMessage carries the parsed contents of a ComLynx reply frame.
type decodedMessage struct {
	Source      Address
	Destination Address
	Type        byte
	Data        []byte
}

// decodeMessage parses a validated frame body (as returned by decodeFrame)
// into a decodedMessage. It returns an error when the transmission or
// application error bit is set on the message type.
func decodeMessage(body []byte) (decodedMessage, error) {
	var msg decodedMessage
	if len(body) < 8 {
		return msg, fmt.Errorf("message body too short: %d bytes", len(body))
	}
	if body[0] != broadcast || body[1] != unnumberedInfo {
		return msg, errors.New("unexpected HDLC header")
	}

	msg.Source = addressFromBytes(body[2], body[3])
	msg.Destination = addressFromBytes(body[4], body[5])
	dataLen := int(body[6])
	msg.Type = body[7]

	if got := len(body) - 8; got != dataLen {
		return msg, fmt.Errorf("declared data length %d does not match payload %d", dataLen, got)
	}
	msg.Data = body[8:]

	if msg.Type&typeApplicationErrorMask != 0 {
		return msg, fmt.Errorf("application error: type 0x%02x", msg.Type)
	}
	if msg.Type&typeTransmissionErrorMask != 0 {
		return msg, fmt.Errorf("transmission error: type 0x%02x", msg.Type)
	}
	return msg, nil
}

// encodeCanRequest builds a 10-byte Embedded CAN Kingdom read-parameter
// payload for the given TLX parameter ID.
func encodeCanRequest(paramID uint16) []byte {
	return []byte{
		0xc8, tlxDestinationModule, 0xd0,
		byte(paramID >> 8), byte(paramID & 0xff),
		0x80, 0x00, 0x00, 0x00, 0x00,
	}
}

// decodeCanReply extracts the 32-bit value from a CanReply payload. The
// value occupies the last four bytes, little-endian. AMajland/Danfoss-TLX
// treats all TLX readings this way; per-parameter scaling happens one layer
// up in params.go.
func decodeCanReply(paramID uint16, data []byte) (int32, error) {
	if len(data) != 10 {
		return 0, fmt.Errorf("can reply data length %d, want 10", len(data))
	}
	// The reply echoes the parameter id at data[3..4].
	if gotHi, gotLo := data[3], data[4]; gotHi != byte(paramID>>8) || gotLo != byte(paramID&0xff) {
		return 0, fmt.Errorf("reply for unexpected parameter %02x%02x (want %04x)", gotHi, gotLo, paramID)
	}
	return int32(binary.LittleEndian.Uint32(data[6:10])), nil
}
