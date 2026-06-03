package aa55

import (
	"encoding/hex"
	"net"
	"testing"

	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Wire-protocol primitives (buildPDU, stripHeader, DecodeAt, ModbusCRC16,
// Cache, real-capture register tests) are covered in aa55_test.go.
// This file covers the AA55UDP plugin adapter end-to-end via FloatGetter.

// TestFloatGetter_DT_Power verifies the full query/decode pipeline using the
// GW17K-DT real capture sliced to just the power register bytes: 12470 W.
func TestFloatGetter_DT_Power(t *testing.T) {
	response := singleRegResponse(t, capGW17kDT, 54, 4, 0x7f)
	p := &AA55UDP{
		log:    util.NewLogger("test"),
		conn:   mockConn(t, response),
		pdu:    buildPDU(0x7F, 0x75AF, 2),
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
		pdu:    buildPDU(0x7F, 0x75C1, 2),
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
		pdu:    buildPDU(0xF7, 0x8941, 2),
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
		pdu:    buildPDU(0xF7, 0x896E, 2),
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
		pdu:    buildPDU(0xF7, 0x908F, 1),
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
	cfg, err := buildReadConfig(int(InverterAddr), 0x75AF, 2, nil)
	require.NoError(t, err)
	assert.Equal(t, []byte{0x7f, 0x03, 0x75, 0xaf, 0x00, 0x02}, cfg.pdu)
	assert.Equal(t, 0, cfg.offset)
	assert.False(t, cfg.useCache)
}

// TestBuildReadConfig_BlockMode verifies the block PDU is built from the block
// register/count and the target register's offset is computed within it.
// ET grid (0x8943) within block READ 125 @ 0x891C → offset (35139-35100)*2 = 78.
func TestBuildReadConfig_BlockMode(t *testing.T) {
	cfg, err := buildReadConfig(0xF7, 0x8943, 2, &Block{Register: 0x891C, Count: 125})
	require.NoError(t, err)
	assert.Equal(t, []byte{0xf7, 0x03, 0x89, 0x1c, 0x00, 0x7d}, cfg.pdu)
	assert.Equal(t, 78, cfg.offset)
	assert.True(t, cfg.useCache)
}

// TestBuildReadConfig_RejectsRegisterOutsideBlock rejects a target register
// that does not fit entirely within the configured block.
func TestBuildReadConfig_RejectsRegisterOutsideBlock(t *testing.T) {
	// before block start
	_, err := buildReadConfig(0xF7, 0x8900, 2, &Block{Register: 0x891C, Count: 125})
	require.Error(t, err)

	// past block end (0x891C+125 = 0x8999; 0x8998+2 overruns)
	_, err = buildReadConfig(0xF7, 0x8998, 2, &Block{Register: 0x891C, Count: 125})
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
	blockPayload, err := stripHeader(cap)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(blockPayload), offset+valueBytes)

	value := blockPayload[offset : offset+valueBytes]
	frame := []byte{0xAA, 0x55, src, 0x03, byte(valueBytes)}
	frame = append(frame, value...)
	frame = append(frame, 0x00, 0x00) // CRC not validated by stripHeader
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
