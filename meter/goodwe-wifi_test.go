package meter

// goodwe_wifi_test.go
//
// Unit tests for the goodwe-wifi meter implementation.
//
// # Protocol framing
//
// GoodWe inverters use a custom framing over UDP port 8899:
//
//	AA 55 7F 03 <byteCount> <payload…> <crc_lo> <crc_hi>
//
// sendCommand strips the 5-byte prefix (AA 55 7F 03 <byteCount>) and the
// trailing 2-byte CRC before returning, so all offsets in CurrentPower /
// TotalEnergy / Soc are into the bare payload.
//
// # Key design constraint
//
// NewGoodWeWifi(uri, usage) unconditionally appends ":8899" to the uri, so
// the mock listener must bind to port 8899 on loopback.  Because only one
// listener can own a port at a time the tests that need a live mock acquire
// port 8899 sequentially (Go's test runner executes top-level Test* functions
// sequentially by default, and each test calls t.Cleanup to close the
// listener before the next one starts).
//
// Tests that do NOT need a network round-trip (payload-parsing tests) call
// unexported helpers directly and bypass the network entirely.
//
// # Real captured frames
//
// All hex constants below are verbatim UDP payloads captured from a
// GW3000-DNS-30 inverter (S/N 53000DSC243W0186) as published in:
// https://github.com/evcc-io/evcc/discussions/27411
//
// Model-name query  (READ 8 regs @ 0x9CED = 40173):
//
//	Request : 7f 03 9c ed 00 08 f0 77
//	Response: aa 55 7f 03 10
//	          47 57 33 30 30 30 2d 44 4e 53 2d 33 30 00 00 00   ← "GW3000-DNS-30"
//	          05 57                                              ← CRC
//	Stripped payload (16 bytes): "GW3000-DNS-30\x00\x00\x00"
//	→ detectFamily sees "DNS" → family = "DT"
//
// Runtime data query (READ 73 regs @ 0x7594 = 30100):
//
//	Request : 7f 03 75 94 00 49 d5 c2
//	Response: aa 55 7f 03 92  (byteCount = 0x92 = 146)
//	Stripped payload layout (byte offsets are 0-based):
//	  [54..57] = 00 00 07 b4  → int32 BE  = 1972  → CurrentPower = 1972 W
//	  [90..93] = 00 00 a9 f9  → uint32 BE = 43513 → TotalEnergy  = 4351.3 kWh

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Captured frames (verbatim from discussion #27411)
// ---------------------------------------------------------------------------

// modelNameResponseHex is the complete UDP datagram returned by the inverter
// for the model-name query (READ 8 regs @ 0x9CED).
const modelNameResponseHex = "aa557f03104757333030302d444e532d33300000000557"

// runtimeDataResponseHex is the first complete runtime datagram
// (READ 73 regs @ 0x7594 = 30100), copied verbatim from discussion #27411.
// Stripped payload (146 bytes):
//   [54..57] = 00 00 07 b4  → int32 BE = 1972 W
//   [90..93] = 00 00 a9 f9  → uint32 BE = 43513 → 4351.3 kWh
const runtimeDataResponseHex = "aa557f03921a020e0e301007cf005f053b000b00000000" +
	"ffffffffffffffffffffffffffffffffffff08eeffffffff" +
	"0056ffffffff1387ffffffff000007b4000100000000000000" +
	"0007a600000002ffffffff03e7ffff011bffffffff00140000" +
	"a9f9000013ff0006ffffffffffffffffffffffffffffffffffff" +
	"ffffffffffffffff0e05ffffffffffff013e000000030cdaffff" +
	"00393eb0"

// runtimeDataResponse2Hex is the second capture (one second later, same values),
// copied verbatim from discussion #27411.
const runtimeDataResponse2Hex = "aa557f03921a020e0e301107cf005f053b000b00000000" +
	"ffffffffffffffffffffffffffffffffffff08eeffffffff" +
	"0056ffffffff1387ffffffff000007b4000100000000000000" +
	"0007a600000002ffffffff03e7ffff011bffffffff00140000" +
	"a9f9000013ff0006ffffffffffffffffffffffffffffffffffff" +
	"ffffffffffffffff0e05ffffffffffff013e000000030cdaffff" +
	"0039efa0"

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// mustDecodeHex decodes a compact hex string (spaces stripped) or panics.
func mustDecodeHex(s string) []byte {
	s = strings.ReplaceAll(s, " ", "")
	b, err := hex.DecodeString(s)
	if err != nil {
		panic("mustDecodeHex: " + err.Error())
	}
	return b
}

// buildModelFrame constructs a minimal AA55 model-name response for an
// arbitrary model string, padded/truncated to exactly 16 payload bytes.
func buildModelFrame(model string) []byte {
	payload := make([]byte, 16)
	copy(payload, model)
	frame := []byte{0xAA, 0x55, 0x7F, 0x03, 0x10}
	frame = append(frame, payload...)
	frame = append(frame, 0x00, 0x00) // CRC not validated by sendCommand
	return frame
}

// buildRuntimeFrame constructs a synthetic AA55 runtime response whose
// stripped payload is payloadLen bytes (minimum 94), with:
//   - powerW    placed at offset 54 (DT CurrentPower offset)
//   - energyVal placed at offset 90 (TotalEnergy offset, as the code reads it)
func buildRuntimeFrame(payloadLen int, powerW int32, energyVal uint32) []byte {
	if payloadLen < 94 {
		payloadLen = 94
	}
	payload := make([]byte, payloadLen)
	binary.BigEndian.PutUint32(payload[54:], uint32(powerW))
	binary.BigEndian.PutUint32(payload[90:], energyVal)
	frame := []byte{0xAA, 0x55, 0x7F, 0x03, byte(payloadLen)}
	frame = append(frame, payload...)
	frame = append(frame, 0x00, 0x00) // CRC placeholder
	return frame
}

// mockOnPort8899 starts a UDP listener on 127.0.0.1:8899, serving the given
// responses in order (cycling when exhausted).  It returns "127.0.0.1" — the
// host-only string that NewGoodWeWifi expects (it appends ":8899" itself).
// The listener is closed via t.Cleanup.
//
// If port 8899 is unavailable (e.g. another process owns it) the test is
// skipped rather than failed, because this is an environment constraint.
func mockOnPort8899(t *testing.T, responses [][]byte) string {
	t.Helper()

	conn, err := net.ListenPacket("udp4", "127.0.0.1:8899")
	if err != nil {
		t.Skipf("cannot bind 127.0.0.1:8899 (%v) – skipping integration test", err)
	}
	t.Cleanup(func() { conn.Close() })

	go func() {
		buf := make([]byte, 1024)
		i := 0
		for {
			_, addr, err := conn.ReadFrom(buf)
			if err != nil {
				return // listener closed during t.Cleanup — normal shutdown
			}
			resp := responses[i%len(responses)]
			i++
			_, _ = conn.WriteTo(resp, addr)
		}
	}()

	return "127.0.0.1"
}

// ---------------------------------------------------------------------------
// Payload-parsing helpers (mirror the code's arithmetic, no network needed)
// ---------------------------------------------------------------------------

// stripAA55Header mimics what sendCommand returns: the bare payload after the
// 5-byte AA55 prefix, with the trailing 2-byte CRC excluded.
func stripAA55Header(frame []byte) ([]byte, error) {
	if len(frame) < 6 || frame[0] != 0xAA || frame[1] != 0x55 ||
		frame[2] != 0x7F || frame[3] != 0x03 {
		return nil, fmt.Errorf("invalid response header")
	}
	byteCount := int(frame[4])
	if len(frame) < 5+byteCount+2 {
		return nil, fmt.Errorf("short response")
	}
	return frame[5 : 5+byteCount], nil
}

func parseDTPower(payload []byte) (float64, error) {
	const offset = 54
	if len(payload) < offset+4 {
		return 0, fmt.Errorf("short runtime data")
	}
	return float64(int32(binary.BigEndian.Uint32(payload[offset : offset+4]))), nil
}

func parseTotalEnergy(payload []byte) (float64, error) {
	const offset = 90
	if len(payload) < offset+4 {
		return 0, fmt.Errorf("short runtime data")
	}
	return float64(binary.BigEndian.Uint32(payload[offset:offset+4])) / 10.0, nil
}

func parseHybridPower(payload []byte, usage string) (float64, error) {
	offsets := map[string]int{"pv": 12, "grid": 24, "battery": 36}
	offset, ok := offsets[usage]
	if !ok {
		return 0, fmt.Errorf("unknown usage: %s", usage)
	}
	if len(payload) < offset+4 {
		return 0, fmt.Errorf("short runtime data")
	}
	return float64(int32(binary.BigEndian.Uint32(payload[offset : offset+4]))), nil
}

// ---------------------------------------------------------------------------
// Payload-parsing unit tests  (no network, no port-8899 requirement)
// ---------------------------------------------------------------------------

func TestStripAA55Header_Valid(t *testing.T) {
	payload := []byte{0x01, 0x02, 0x03, 0x04}
	frame := append([]byte{0xAA, 0x55, 0x7F, 0x03, 0x04}, payload...)
	frame = append(frame, 0x00, 0x00)
	got, err := stripAA55Header(frame)
	require.NoError(t, err)
	assert.Equal(t, payload, got)
}

func TestStripAA55Header_BadMagic(t *testing.T) {
	frame := []byte{0xFF, 0x55, 0x7F, 0x03, 0x04, 0x01, 0x02, 0x03, 0x04, 0x00, 0x00}
	_, err := stripAA55Header(frame)
	assert.Error(t, err)
}

func TestStripAA55Header_TruncatedPayload(t *testing.T) {
	// header says 10 bytes follow but only 3 are present
	frame := []byte{0xAA, 0x55, 0x7F, 0x03, 0x0A, 0x01, 0x02, 0x03, 0x00, 0x00}
	_, err := stripAA55Header(frame)
	assert.Error(t, err)
}

// TestParseDTPower_RealCapture verifies power parsing against the real capture.
// Stripped payload offset 54 = 00 00 07 b4 → 1972 W.
func TestParseDTPower_RealCapture(t *testing.T) {
	frame := mustDecodeHex(runtimeDataResponseHex)
	payload, err := stripAA55Header(frame)
	require.NoError(t, err)

	power, err := parseDTPower(payload)
	require.NoError(t, err)
	assert.InDelta(t, 1972.0, power, 0.5)
}

func TestParseDTPower_SecondCapture(t *testing.T) {
	frame := mustDecodeHex(runtimeDataResponse2Hex)
	payload, err := stripAA55Header(frame)
	require.NoError(t, err)

	power, err := parseDTPower(payload)
	require.NoError(t, err)
	assert.InDelta(t, 1972.0, power, 0.5)
}

func TestParseDTPower_Zero(t *testing.T) {
	payload := make([]byte, 100)
	power, err := parseDTPower(payload)
	require.NoError(t, err)
	assert.Equal(t, 0.0, power)
}

// TestParseDTPower_Negative verifies signed int32 decoding for negative power.
func TestParseDTPower_Negative(t *testing.T) {
	payload := make([]byte, 100)
	var neg int32 = -500
	binary.BigEndian.PutUint32(payload[54:], uint32(neg))
	power, err := parseDTPower(payload)
	require.NoError(t, err)
	assert.InDelta(t, -500.0, power, 0.5)
}

func TestParseDTPower_Short(t *testing.T) {
	_, err := parseDTPower(make([]byte, 20))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "short")
}

// TestParseTotalEnergy_RealCapture verifies TotalEnergy parsing against the
// real capture.  Stripped payload offset 90 = 00 00 a9 f9 → 43513 → 4351.3 kWh.
func TestParseTotalEnergy_RealCapture(t *testing.T) {
	frame := mustDecodeHex(runtimeDataResponseHex)
	payload, err := stripAA55Header(frame)
	require.NoError(t, err)

	energy, err := parseTotalEnergy(payload)
	require.NoError(t, err)
	assert.InDelta(t, 4351.3, energy, 0.001)
}

// TestParseTotalEnergy_KnownValue places a known value at offset 90 and
// verifies the ÷10 scaling with a synthetic payload.
func TestParseTotalEnergy_KnownValue(t *testing.T) {
	payload := make([]byte, 100)
	binary.BigEndian.PutUint32(payload[90:], 43513) // 4351.3 kWh × 10
	energy, err := parseTotalEnergy(payload)
	require.NoError(t, err)
	assert.InDelta(t, 4351.3, energy, 0.001)
}

func TestParseTotalEnergy_Short(t *testing.T) {
	_, err := parseTotalEnergy(make([]byte, 50))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "short")
}

// TestParseHybridPower_Offsets verifies each usage maps to its documented offset.
func TestParseHybridPower_Offsets(t *testing.T) {
	cases := []struct {
		usage  string
		offset int
		power  int32
	}{
		{"pv", 12, 3000},
		{"grid", 24, -800},  // negative = exporting to grid
		{"battery", 36, 500},
	}
	for _, tc := range cases {
		t.Run(tc.usage, func(t *testing.T) {
			payload := make([]byte, 100)
			binary.BigEndian.PutUint32(payload[tc.offset:], uint32(tc.power))
			power, err := parseHybridPower(payload, tc.usage)
			require.NoError(t, err)
			assert.InDelta(t, float64(tc.power), power, 0.5)
		})
	}
}

func TestParseHybridPower_UnknownUsage(t *testing.T) {
	_, err := parseHybridPower(make([]byte, 100), "solar")
	assert.Error(t, err)
}

// ---------------------------------------------------------------------------
// Integration tests  (require 127.0.0.1:8899 to be available)
// ---------------------------------------------------------------------------

// TestDetectFamily_DNS30_isDT drives the full NewGoodWeWifi constructor with
// the real model-name capture and verifies detection completes without error.
// A successful CurrentPower call with the real runtime capture then confirms
// the DT branch was taken (1972 W from offset 54).
func TestDetectFamily_DNS30_isDT(t *testing.T) {
	modelResp := mustDecodeHex(modelNameResponseHex)
	runtimeResp := mustDecodeHex(runtimeDataResponseHex)

	host := mockOnPort8899(t, [][]byte{modelResp, runtimeResp})

	m, err := NewGoodWeWifi(host, "pv")
	require.NoError(t, err, "GW3000-DNS-30 should be detected as DT without error")

	power, err := m.CurrentPower()
	require.NoError(t, err)
	assert.InDelta(t, 1972.0, power, 0.5)
}

// TestDetectFamily_ET_isHybrid verifies that a GW10K-ET model is accepted and
// that CurrentPower with usage="pv" reads offset 12 from the HYBRID response.
func TestDetectFamily_ET_isHybrid(t *testing.T) {
	modelFrame := buildModelFrame("GW10K-ET")
	runtimeFrame := buildRuntimeFrame(100, 0, 0)
	// Place 2000 W at HYBRID pv offset (12)
	binary.BigEndian.PutUint32(runtimeFrame[5+12:], uint32(int32(2000)))

	host := mockOnPort8899(t, [][]byte{modelFrame, runtimeFrame})

	m, err := NewGoodWeWifi(host, "pv")
	require.NoError(t, err, "GW10K-ET should be detected as HYBRID without error")

	power, err := m.CurrentPower()
	require.NoError(t, err)
	assert.InDelta(t, 2000.0, power, 0.5)
}

// TestDTRejectsBatteryUsage verifies that construction with usage="battery" on
// a DT inverter returns an error containing "battery".
func TestDTRejectsBatteryUsage(t *testing.T) {
	host := mockOnPort8899(t, [][]byte{mustDecodeHex(modelNameResponseHex)})

	_, err := NewGoodWeWifi(host, "battery")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "battery")
}

// TestDTRejectsGridUsage mirrors the above for usage="grid".
func TestDTRejectsGridUsage(t *testing.T) {
	host := mockOnPort8899(t, [][]byte{mustDecodeHex(modelNameResponseHex)})

	_, err := NewGoodWeWifi(host, "grid")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "grid")
}

// TestUnknownModel verifies that a model string containing none of the
// recognised family tokens (DNS/DT/ET/EH/BT/BH) causes construction to fail
// with an error that mentions "unknown model".
func TestUnknownModel(t *testing.T) {
	host := mockOnPort8899(t, [][]byte{buildModelFrame("XYZZY-99")})

	_, err := NewGoodWeWifi(host, "pv")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown model")
}

// TestCurrentPower_RealCapture is an end-to-end integration test using both
// real captured frames.
func TestCurrentPower_RealCapture(t *testing.T) {
	host := mockOnPort8899(t, [][]byte{
		mustDecodeHex(modelNameResponseHex),
		mustDecodeHex(runtimeDataResponseHex),
	})

	m, err := NewGoodWeWifi(host, "pv")
	require.NoError(t, err)

	power, err := m.CurrentPower()
	require.NoError(t, err)
	assert.InDelta(t, 1972.0, power, 0.5)
}

// TestTotalEnergy_KnownValue is an end-to-end test with a synthetic frame that
// places 43513 at offset 90, exercising the full stack including ÷10 scaling.
func TestTotalEnergy_KnownValue(t *testing.T) {
	host := mockOnPort8899(t, [][]byte{
		mustDecodeHex(modelNameResponseHex),
		buildRuntimeFrame(100, 0, 43513), // 4351.3 kWh
	})

	m, err := NewGoodWeWifi(host, "pv")
	require.NoError(t, err)

	me, ok := m.(api.MeterEnergy)
	require.True(t, ok, "goodWeWifi must implement api.MeterEnergy")

	energy, err := me.TotalEnergy()
	require.NoError(t, err)
	assert.InDelta(t, 4351.3, energy, 0.001)
}

// TestSentPDU_DT verifies the exact bytes sent to a DT inverter for a runtime
// data request.  From the real capture:
//
//	Sending: READ 73 regs from 30100  →  7f 03 75 94 00 49 d5 c2
func TestSentPDU_DT(t *testing.T) {
	conn, err := net.ListenPacket("udp4", "127.0.0.1:8899")
	if err != nil {
		t.Skipf("cannot bind 127.0.0.1:8899: %v", err)
	}
	t.Cleanup(func() { conn.Close() })

	responses := [][]byte{
		mustDecodeHex(modelNameResponseHex),
		mustDecodeHex(runtimeDataResponseHex),
	}
	received := make(chan []byte, 10)

	go func() {
		buf := make([]byte, 1024)
		i := 0
		for {
			n, addr, err := conn.ReadFrom(buf)
			if err != nil {
				return
			}
			pkt := make([]byte, n)
			copy(pkt, buf[:n])
			received <- pkt
			_, _ = conn.WriteTo(responses[i%len(responses)], addr)
			i++
		}
	}()

	m, err := NewGoodWeWifi("127.0.0.1", "pv")
	require.NoError(t, err)

	// Drain the identification PDU sent during construction.
	select {
	case <-received:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for identification PDU")
	}

	_, _ = m.CurrentPower()

	var runtimePDU []byte
	select {
	case runtimePDU = <-received:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for runtime PDU")
	}

	// 6 header bytes + 2 CRC = 8 total
	require.Len(t, runtimePDU, 8, "DT PDU must be exactly 8 bytes (6 + 2 CRC)")
	assert.Equal(t,
		[]byte{0x7F, 0x03, 0x75, 0x94, 0x00, 0x49},
		runtimePDU[:6],
		"DT runtime PDU header mismatch (READ 73 regs @ 0x7594)")
	// CRC d5 c2 from the real capture
	assert.Equal(t,
		[]byte{0xD5, 0xC2},
		runtimePDU[6:8],
		"DT runtime PDU CRC mismatch")
}

// TestTimeout verifies that the constructor returns an error within a
// reasonable time when the inverter never responds (sendCommand has a
// 4-second SetReadDeadline).
func TestTimeout(t *testing.T) {
	conn, err := net.ListenPacket("udp4", "127.0.0.1:8899")
	if err != nil {
		t.Skipf("cannot bind 127.0.0.1:8899: %v", err)
	}
	t.Cleanup(func() { conn.Close() })

	go func() {
		buf := make([]byte, 1024)
		for {
			if _, _, err := conn.ReadFrom(buf); err != nil {
				return
			}
			// intentionally no reply
		}
	}()

	start := time.Now()
	_, err = NewGoodWeWifi("127.0.0.1", "pv")
	elapsed := time.Since(start)

	assert.Error(t, err, "constructor must return an error when inverter is silent")
	assert.Less(t, elapsed, 10*time.Second,
		"should time out in ~4 s (SetReadDeadline), not hang indefinitely")
}
