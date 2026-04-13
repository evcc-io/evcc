package danfoss

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFCS16KnownVector verifies the FCS-16 implementation.
//
// The PPP FCS-16 (RFC 1662) uses init=0xFFFF and final XOR=0xFFFF. Because
// of the final XOR, the "good FCS" self-check residue (0xF0B8 of the raw
// accumulator) appears as 0xF0B8 ^ 0xFFFF = 0x0F47 when our fcs16 function
// is applied to a (data || fcs) buffer.
//
// The TestGoldenFrame test validates the implementation against a real
// captured TLX inverter frame and is the authoritative correctness check.
func TestFCS16KnownVector(t *testing.T) {
	// Self-check: feeding (data || fcs) back through fcs16 produces a fixed
	// residue. Because our fcs16 applies the final 0xFFFF XOR, the residue
	// is fcsGoodResidue (0xF0B8) XOR 0xFFFF = 0x0F47.
	data := []byte{0x01, 0x02, 0x03}
	v := fcs16(data)
	payload := append(data, byte(v&0xff), byte(v>>8))
	residue := fcs16(payload)
	const wantResidue = uint16(fcsGoodResidue ^ 0xffff)
	assert.Equal(t, wantResidue, residue, "good FCS residue (with final XOR applied)")

	// Verify that encoding and decoding a frame round-trips without FCS errors.
	body := encodeMessage(DefaultSource, Broadcast, typePingRequest, nil)
	frame := encodeFrame(body)
	decoded, err := decodeFrame(frame)
	assert.NoError(t, err)
	assert.Equal(t, body, decoded)
}

// TestByteStuffUnstuff verifies round-trip symmetry of the byte stuffing.
func TestByteStuffUnstuff(t *testing.T) {
	cases := []struct {
		name  string
		input []byte
	}{
		{"no special bytes", []byte{0x01, 0x02, 0x03}},
		{"flag sequence", []byte{0x7e, 0x01}},
		{"control escape", []byte{0x7d, 0x01}},
		{"both", []byte{0x7e, 0x7d, 0x01, 0x7e}},
		{"all zeros", make([]byte, 8)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stuffed := byteStuff(nil, tc.input)
			got, err := byteUnstuff(stuffed)
			require.NoError(t, err)
			assert.Equal(t, tc.input, got)
		})
	}
}

// TestEncodeDecodeFrame verifies that a round-trip encode→decode preserves the
// message body and that the FCS check passes.
func TestEncodeDecodeFrame(t *testing.T) {
	src := DefaultSource
	dst := Address{Network: 0xc, Subnet: 0x6, Node: 0xb1}
	data := encodeCanRequest(paramGridPowerTotal)
	body := encodeMessage(src, dst, typeCanRequest, data)

	frame := encodeFrame(body)
	// Frame must begin and end with flag sequence.
	assert.Equal(t, flagSequence, frame[0])
	assert.Equal(t, flagSequence, frame[len(frame)-1])

	// Decode must recover the original body.
	decoded, err := decodeFrame(frame)
	require.NoError(t, err)
	assert.Equal(t, body, decoded)
}

// TestDecodeMessage verifies that a decoded body produces the expected message
// fields.
func TestDecodeMessage(t *testing.T) {
	src := DefaultSource
	dst := Address{Network: 0xc, Subnet: 0x6, Node: 0xb1}
	data := encodeCanRequest(paramGridPowerTotal)
	body := encodeMessage(src, dst, typeCanRequest, data)

	msg, err := decodeMessage(body)
	require.NoError(t, err)
	assert.Equal(t, src, msg.Source)
	assert.Equal(t, dst, msg.Destination)
	assert.Equal(t, typeCanRequest, msg.Type)
	assert.Equal(t, data, msg.Data)
}

// TestDecodeCanReply exercises the 32-bit little-endian value extraction from
// a realistic CanReply payload.
//
// Payload construction based on AMajland/Danfoss-TLX observed capture:
//
//	7EFF03 C6B1 0002 0A81 C80D800A02 46 3C000000 xxxx 7E
//	                           ^^^^  ^^
//	                     param high  low (0x0246 = GridPowerTotal)
//	                                    ^^^^^^^^
//	                                value 60 W (little-endian uint32)
func TestDecodeCanReply(t *testing.T) {
	payload := []byte{
		0xc8, 0x0d, 0x80,
		byte(paramGridPowerTotal >> 8), byte(paramGridPowerTotal & 0xff),
		0x06,                   // data type nibble in lower bits (UNSIGNED_16)
		0x3c, 0x00, 0x00, 0x00, // 60 W, little-endian
	}
	v, err := decodeCanReply(paramGridPowerTotal, payload)
	require.NoError(t, err)
	assert.EqualValues(t, 60, v)
}

// TestDecodeCanReplyParamMismatch ensures an error is returned when the reply
// echoes a different parameter than requested.
func TestDecodeCanReplyParamMismatch(t *testing.T) {
	payload := []byte{
		0xc8, 0x0d, 0x80,
		0x01, 0x02, // param 0x0102 (TotalEnergy), not what we requested
		0x07,
		0x00, 0x00, 0x00, 0x00,
	}
	_, err := decodeCanReply(paramGridPowerTotal, payload)
	assert.ErrorContains(t, err, "unexpected parameter")
}

// TestGoldenFrame is a byte-level regression test using a real frame sequence
// captured from an AMajland ESP32 talking to a TLX inverter (OpMode = 60,
// "ONGRID"). This guards against any inadvertent change to the codec.
//
// Raw capture (AMajland debug output, reversed from hex dump):
//
//	TX: 7EFF030002FFFF0A01C808D00A028000000090117E  (... wait, let me use the real example)
//	RX: 7EFF03C6B100020A81C80D800A02453C00000090117E
func TestGoldenFrame(t *testing.T) {
	// Pre-computed frame from AMajland comment in DanfossTLX-RS485.cpp:
	// "7EFF03C6B100020A81C80D800A02453C00000090117E"
	frame := []byte{
		0x7e, 0xff, 0x03, 0xc6, 0xb1, 0x00, 0x02, 0x0a, 0x81,
		0xc8, 0x0d, 0x80, 0x0a, 0x02, 0x45, 0x3c, 0x00, 0x00, 0x00,
		0x90, 0x11, 0x7e,
	}

	body, err := decodeFrame(frame)
	require.NoError(t, err)

	msg, err := decodeMessage(body)
	require.NoError(t, err)
	assert.Equal(t, typeCanReply, msg.Type)
	// Source = inverter at network=C subnet=6 node=B1
	assert.Equal(t, Address{Network: 0xc, Subnet: 0x6, Node: 0xb1}, msg.Source)
	// Destination = our evcc source address
	assert.Equal(t, Address{Network: 0x0, Subnet: 0x0, Node: 0x02}, msg.Destination)

	// The parameter echoed is 0x0A02 (OpMode). Value in data[3..4].
	assert.Equal(t, byte(0x0a), msg.Data[3])
	assert.Equal(t, byte(0x02), msg.Data[4])

	// Value = 60 decimal = ONGRID
	v, err := decodeCanReply(paramOperatingMode, msg.Data)
	require.NoError(t, err)
	assert.EqualValues(t, 60, v)
}
