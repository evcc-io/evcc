package plugin

import (
	"encoding/hex"
	"net"
	"testing"

	"github.com/evcc-io/evcc/plugin/aa55"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Wire-protocol primitives (BuildPDU, StripHeader, DecodeAt, ModbusCRC16,
// Cache, and real-capture register tests) are unit-tested in plugin/aa55.
// This file only covers the plugin adapter end-to-end via FloatGetter.

// Real captured frames used to build single-register mock responses.
const (
	capGW17kDT        = `aa557f03921805140a23371518006912930094ffffffffffffffffffffffffffffffff102210130fff093f094f094500b000af00af138a138a138a000030b600010000000000000000000000000000000000000000ffff01c9ffffffff012500049344000020a500010000000000000000ffffffffffffffffffffffffffffffff0222184a0c4600000004000003a300f7000400000064b2f2`
	capGW10kET        = `aa55f703fa1508160b0b0c0cfe00330000069f0cfe0035000006e100000000000000000000000000000000000002020959000f138700000150096f000d13870000011f096b000b1387000000ce00010000033ffffffffd000000000000000009560006138600010000006b096d000913880001000000bd096c00021387000100000000000000e000000050000000e9000001380000020a000401fe0000024b00001f640fb209eeff9efffff63000030000002000010000000000000000edb50000007d0000b8520000241e00620000024400000001588a007400006bbd003500005f65001d0005000000010000000000000000000107000800000209ee000055ae`
	capGW10kETBattery = `aa55f7033000ff01000001015e001900190000004400630005000001010000000000000000000000000000000000000000000000006447`
)

// TestFloatGetter_DT_Power verifies the full query/decode pipeline using the
// GW17K-DT real capture sliced to just the power register bytes: 12470 W.
func TestFloatGetter_DT_Power(t *testing.T) {
	response := singleRegResponse(t, capGW17kDT, 54, 4, 0x7f)
	p := &AA55UDP{
		log:    util.NewLogger("test"),
		conn:   mockConn(t, response),
		pdu:    aa55.BuildPDU(0x7F, 0x75AF, 2),
		decode: "int32be",
		scale:  1.0,
	}
	getter, err := p.FloatGetter()
	require.NoError(t, err)
	val, err := getter()
	require.NoError(t, err)
	assert.InDelta(t, 12470.0, val, 0.5)
}

// TestFloatGetter_DT_Energy verifies scale 0.1: 299844 × 0.1 = 29984.4 kWh.
func TestFloatGetter_DT_Energy(t *testing.T) {
	response := singleRegResponse(t, capGW17kDT, 90, 4, 0x7f)
	p := &AA55UDP{
		log:    util.NewLogger("test"),
		conn:   mockConn(t, response),
		pdu:    aa55.BuildPDU(0x7F, 0x75C1, 2),
		decode: "uint32be",
		scale:  0.1,
	}
	getter, err := p.FloatGetter()
	require.NoError(t, err)
	val, err := getter()
	require.NoError(t, err)
	assert.InDelta(t, 29984.4, val, 0.001)
}

// TestFloatGetter_ET_PV verifies ET pv power: GW10K-ET = 831 W.
func TestFloatGetter_ET_PV(t *testing.T) {
	response := singleRegResponse(t, capGW10kET, 74, 4, 0xf7)
	p := &AA55UDP{
		log:    util.NewLogger("test"),
		conn:   mockConn(t, response),
		pdu:    aa55.BuildPDU(0xF7, 0x8941, 2),
		decode: "int32be",
		scale:  1.0,
	}
	getter, err := p.FloatGetter()
	require.NoError(t, err)
	val, err := getter()
	require.NoError(t, err)
	assert.InDelta(t, 831.0, val, 0.5)
}

// TestFloatGetter_ET_Battery verifies charging battery: GW10K-ET = -2512 W.
func TestFloatGetter_ET_Battery(t *testing.T) {
	response := singleRegResponse(t, capGW10kET, 164, 4, 0xf7)
	p := &AA55UDP{
		log:    util.NewLogger("test"),
		conn:   mockConn(t, response),
		pdu:    aa55.BuildPDU(0xF7, 0x896E, 2),
		decode: "int32be",
		scale:  1.0,
	}
	getter, err := p.FloatGetter()
	require.NoError(t, err)
	val, err := getter()
	require.NoError(t, err)
	assert.InDelta(t, -2512.0, val, 0.5)
}

// TestFloatGetter_ET_SoC verifies SoC: GW10K-ET = 68%.
func TestFloatGetter_ET_SoC(t *testing.T) {
	response := singleRegResponse(t, capGW10kETBattery, 14, 2, 0xf7)
	p := &AA55UDP{
		log:    util.NewLogger("test"),
		conn:   mockConn(t, response),
		pdu:    aa55.BuildPDU(0xF7, 0x908F, 1),
		decode: "uint16be",
		scale:  1.0,
	}
	getter, err := p.FloatGetter()
	require.NoError(t, err)
	val, err := getter()
	require.NoError(t, err)
	assert.InDelta(t, 68.0, val, 0.5)
}

// TestBuildReadConfig_RegisterMode verifies default register-read config.
func TestBuildReadConfig_RegisterMode(t *testing.T) {
	cfg, err := buildReadConfig(int(aa55.InverterAddr), "", 0x75AF, 2, 0)
	require.NoError(t, err)
	assert.Equal(t, []byte{0x7f, 0x03, 0x75, 0xaf, 0x00, 0x02}, cfg.pdu)
	assert.Equal(t, 0, cfg.offset)
	assert.False(t, cfg.useCache)
}

// TestBuildReadConfig_BlockMode verifies block-read config from raw PDU.
func TestBuildReadConfig_BlockMode(t *testing.T) {
	cfg, err := buildReadConfig(int(aa55.InverterAddr), "f703891c007d", 0, 2, 54)
	require.NoError(t, err)
	assert.Equal(t, []byte{0xf7, 0x03, 0x89, 0x1c, 0x00, 0x7d}, cfg.pdu)
	assert.Equal(t, 54, cfg.offset)
	assert.True(t, cfg.useCache)
}

// TestBuildReadConfig_RejectsMixedConfig rejects pdu+register combinations.
func TestBuildReadConfig_RejectsMixedConfig(t *testing.T) {
	_, err := buildReadConfig(int(aa55.InverterAddr), "f703891c007d", 0x75AF, 2, 0)
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// singleRegResponse builds the AA55 response frame that an inverter would
// return for a single-register read, by slicing the value bytes out of a
// real block-read capture at the given offset.
// src is the inverter source byte (0x7f for DT, 0xf7 for ET).
func singleRegResponse(t *testing.T, capHex string, offset, valueBytes int, src byte) []byte {
	t.Helper()
	cap, err := hex.DecodeString(capHex)
	require.NoError(t, err)
	blockPayload, err := aa55.StripHeader(cap)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(blockPayload), offset+valueBytes)

	value := blockPayload[offset : offset+valueBytes]
	frame := []byte{0xAA, 0x55, src, 0x03, byte(valueBytes)}
	frame = append(frame, value...)
	frame = append(frame, 0x00, 0x00) // CRC not validated by aa55.StripHeader
	return frame
}

// mockConn starts a UDP server that responds with response to every packet,
// and returns a *net.UDPConn already dialled at that server.
func mockConn(t *testing.T, response []byte) *net.UDPConn {
	t.Helper()
	srv, err := net.ListenPacket("udp4", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { srv.Close() })

	go func() {
		buf := make([]byte, 512)
		for {
			_, addr, err := srv.ReadFrom(buf)
			if err != nil {
				return
			}
			_, _ = srv.WriteTo(response, addr)
		}
	}()

	addr, err := net.ResolveUDPAddr("udp4", srv.LocalAddr().String())
	require.NoError(t, err)
	conn, err := net.DialUDP("udp4", nil, addr)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	return conn
}
