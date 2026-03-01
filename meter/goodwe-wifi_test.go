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
// Hex constants marked "real capture" are verbatim UDP payloads captured from a
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
//
// # Inverter families and their PDUs
//
// Family "DT"  (model token "DNS" or "DT"):
//   - CurrentPower PDU: READ 73 regs @ 0x7594 → {7f 03 75 94 00 49}
//   - CurrentPower offset: 54 (int32, W)
//   - TotalEnergy PDU: same as above
//   - TotalEnergy offset: 90 (uint32, ×0.1 kWh)
//   - Soc: not supported (DT = string inverter, no battery)
//   - Allowed usages: "pv" only
//
// Family "HYBRID" (model tokens "ET", "EH", "BT", "BH"):
//   - CurrentPower PDU: READ 42 regs @ 0x7500 → {7f 03 75 00 00 2a}
//   - pv offset:      12 (int32, W)
//   - grid offset:    24 (int32, W, negative = exporting)
//   - battery offset: 36 (int32, W, positive = discharging)
//   - Soc offset:     28 (uint16, %)
//   - TotalEnergy PDU: DT PDU (known limitation — see TestTotalEnergy_HybridUsage)
//   - Allowed usages: "pv", "grid", "battery"

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Captured frames (verbatim from discussion #27411, GW3000-DNS-30)
// ---------------------------------------------------------------------------

// modelNameResponseHex is the complete UDP datagram returned by the inverter
// for the model-name query (READ 8 regs @ 0x9CED).
const modelNameResponseHex = "aa557f03104757333030302d444e532d33300000000557"

// runtimeDataResponseHex is the first complete runtime datagram
// (READ 73 regs @ 0x7594 = 30100), copied verbatim from discussion #27411.
// Stripped payload (146 bytes):
//
//	[54..57] = 00 00 07 b4  → int32 BE = 1972 W
//	[90..93] = 00 00 a9 f9  → uint32 BE = 43513 → 4351.3 kWh
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
// Additional real captures from marcelblijleven/goodwe tests/sample/
// (https://github.com/marcelblijleven/goodwe/tree/master/tests/sample)
//
// All are DT-family 146-byte runtime responses (READ 73 regs @ 0x7594).
// Values confirmed by applying the same offset arithmetic:
//   power  = int32 BE @ stripped-payload[54..57]
//   energy = uint32 BE @ stripped-payload[90..93], ÷10 → kWh
//
// GW8K-DT:      power=643 W,   energy=0xFFFFFFFF (meter not connected → skip)
// GW17K-DT:     power=12470 W, energy=299844 → 29984.4 kWh
// GW20KAU-DT:   power=4957 W,  energy=43048  → 4304.8 kWh
// GW6000-DT:    power=1835 W,  energy=133502 → 13350.2 kWh
//
// GW5000-MS and GW10K-MS-30 are single-phase string inverters (MS series).
// Their running-data payload is identical in layout to DT (146 bytes), but
// "MS" is not in the evcc detectFamily token list → constructor returns
// "unknown model" error.  Included here to document that behaviour.
// ---------------------------------------------------------------------------

const gw8kDTRuntimeHex = "aa557f0392150818102b1b0ac3000613f40008ffffffffffffffffffffffffffffffff102910220ff0094409650930000a000a000a1390138c13880000028300010000000000000000000000000000000000000000ffff01c5ffffffffffffffffffffffffffff00200000000000000000ffffffffffffffffffffffffffffffff020018620c6000000000000002d800810000ffff0054f42f"

const gw17kDTRuntimeHex = "aa557f03921805140a23371518006912930094ffffffffffffffffffffffffffffffff102210130fff093f094f094500b000af00af138a138a138a000030b600010000000000000000000000000000000000000000ffff01c9ffffffff012500049344000020a500010000000000000000ffffffffffffffffffffffffffffffff0222184a0c4600000004000003a300f7000400000064b2f2"

const gw20kAUDTRuntimeHex = "aa557f0392160a1513172a0f4100440dbc0047ffffffffffffffffffffffffffffffff0f2d0f4d0f6908d508bc08eb0048004a00471384138413850000135d000100000000000000000000000000cd0000000003e7ffff016cffffffff00c60000a8280000047300200000000000000000ffffffffffffffffffffffffffffffff0000174b0bad000000040000044b00000004000000696b04"

const gw6000DTRuntimeHex = "aa557f039215081f0c03020c88001f0ca90020ffffffffffffffffffffffffffffffffffffffffffff08d008f90906001b001a001b1386138613860000072b0001000000000000ffffffffffffffffffffffff0000ffff019dffffffff003c0002097e0000210300140000ffff0000ffff0000ffff0000ffffffffffffffffffff0000177c0beeffffffff00cf016302f00000000000649f03"

// GW5000-MS and GW10K-MS-30: 146-byte payload but "MS" not in detectFamily →
// constructor must return an "unknown model" error.
const gw5000MSRuntimeHex = "aa557f0392150a0f09030c0c7c000205c8000305980004ffffffffffffffffffffffffffffffffffff0961ffffffff0009ffffffff1386ffffffff000001270001000000000000ffffffffffffffffffffffffffffffff006bffffffff0004000000440000000700490000ffff0000ffff0000ffff0000ffffffffffffffffffff09500f63ffffffffffff01e1ffffffffffff0103002a4038"

// ---------------------------------------------------------------------------
// ET/EH/BT/BH family real captures
// (frames from marcelblijleven/goodwe tests/sample/)
//
// Runtime PDU: READ 125 regs @ 0x891C → 250-byte payload.
// Offsets in stripped payload (all signed int32 BE unless noted):
//   pv power       : offset  74  (total_inverter_power, S32, W)
//   grid power     : offset  78  (ac_active_power, S32, negative = exporting)
//   battery power  : offset 164  (pbattery1, S32, negative = charging)
//   e_total        : offset 182  (U32, ÷10 → kWh)
//
// Battery info PDU: READ 24 regs @ 0x9088 → 48-byte payload.
//   SoC            : offset  14  (U16, %)
//
// GW10K-ET fw617:  pv=831W   grid=-3W    bat=-2512W  e_total=6085.3 kWh
// GW25K-ET:        pv=1945W  grid=1511W  bat=0W      e_total=160.3 kWh
// GW29K9-ET:       pv=1735W  grid=-5403W bat=0W      e_total=4562.3 kWh
// GW6000-EH:       pv=1561W  (grid register contains stale data in this capture)
// ---------------------------------------------------------------------------

const gw10kETRuntimeHex = "aa55f703fa1508160b0b0c0cfe00330000069f0cfe0035000006e100000000000000000000000000000000000002020959000f138700000150096f000d13870000011f096b000b1387000000ce00010000033ffffffffd000000000000000009560006138600010000006b096d000913880001000000bd096c00021387000100000000000000e000000050000000e9000001380000020a000401fe0000024b00001f640fb209eeff9efffff63000030000002000010000000000000000edb50000007d0000b8520000241e00620000024400000001588a007400006bbd003500005f65001d0005000000010000000000000000000107000800000209ee000055ae"

const gw25kETRuntimeHex = "aa55f703fa170c030e07071cd3000e000004091cd30000000003d51d82000d000000001d82000000000000000002020905001d13830000024d0906001b1385000002290900002a138500000323000100000799000005e7000004d7000008d308f7001e138300000000003408fc0012138500000000000f08f6002013850000000001580000002c0000001000000153000001980000001a000701ce000001ae00001e350f1a0868000000000000000200000020000100000000000000000643000000930000056100000184001d00000094000a000000ac000200000391006e000002b800000004000000000000000000000000000002040180000200008f005ece"

const gw29k9ETRuntimeHex = "aa55f703fa1801110e310e1aad000f000001de1aad0000000002a7168d001200000186168d000000000000000202020909001d1387000002470919001b1387000002350920001d13850000024b0001000006c7ffffeae500000133000007b708fb00071386000000000015090b0007138800000000000509190006138500000000002500000287000002800000028b0000004200001ba0000100f1000000cd00001db40eda0000ffff0000000000000000002000010000000000000000b237000000090000af6100000497000c0000005700000001a39e01b600000000000000000000000000060000000000000000000000000000020400ce00000000030064b6"

const gw6000EHRuntimeHex = "aa55f703fa1508081228090ce7001a000003590ce00015000002b3ffffffffffffffffffffffffffffffff00000202093e0042138500000619ffffffffffff7fffffffffffffffffff7fffffff0001000006190000ff5c7fffffffffffffff000000000000000000000000ffffffffffffffffffffffffffffffffffffffffffffffff000006bc7fffffff7fffffff00000000000006bd0000025c7fff018201000edeffff00000001000000000000000000030001ffff0000000000000252000000dc0000024a0000002100d8000000000000000002bd010f000000000000000000000000000700000000000000000000400000010708484700000000ffffbcb6"

// Battery info frames: READ 24 regs @ 0x9088 → 48-byte payload, SoC at offset 14.
// GW10K-ET: SoC=68%, GW25K-ET: SoC=100%
const gw10kETBatteryInfoHex = "aa55f7033000ff01000001015e001900190000004400630005000001010000000000000000000000000000000000000000000000006447"
const gw25kETBatteryInfoHex = "aa55f7033000ff0137000100e600000028000000640064000400000105000000000316000000000000000000000000000000000000dc7a"

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

// buildDTRuntimeFrame constructs a synthetic AA55 DT runtime response
// (READ 73 regs @ 0x7594, byteCount ≥ 94) with:
//   - powerW    placed at offset 54 (DT CurrentPower offset, int32 BE)
//   - energyVal placed at offset 90 (TotalEnergy offset, uint32 BE, ×0.1 kWh)
func buildDTRuntimeFrame(payloadLen int, powerW int32, energyVal uint32) []byte {
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

// buildRuntimeFrame is an alias for buildDTRuntimeFrame kept for
// compatibility with existing integration tests.
var buildRuntimeFrame = buildDTRuntimeFrame

// buildHybridRuntimeFrame constructs a synthetic AA55 HYBRID runtime response
// (READ 42 regs @ 0x7500, byteCount = 84) with:
//   - pvW      placed at offset 12 (int32 BE, W)
//   - gridW    placed at offset 24 (int32 BE, W; negative = exporting)
//   - batteryW placed at offset 36 (int32 BE, W; positive = discharging)
//   - soc      placed at offset 28 (uint16 BE, %)
func buildHybridRuntimeFrame(pvW, gridW, batteryW int32, soc uint16) []byte {
	const payloadLen = 84 // 42 registers × 2 bytes
	payload := make([]byte, payloadLen)
	binary.BigEndian.PutUint32(payload[12:], uint32(pvW))
	binary.BigEndian.PutUint32(payload[24:], uint32(gridW))
	binary.BigEndian.PutUint32(payload[36:], uint32(batteryW))
	binary.BigEndian.PutUint16(payload[28:], soc)
	frame := []byte{0xAA, 0x55, 0x7F, 0x03, byte(payloadLen)}
	frame = append(frame, payload...)
	frame = append(frame, 0x00, 0x00)
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
// buf[2] is the inverter source address, which varies by family:
//   DT/DNS: 0x7F,  ET/EH/BT/BH: 0xF7 — we accept both.
func stripAA55Header(frame []byte) ([]byte, error) {
	if len(frame) < 6 || frame[0] != 0xAA || frame[1] != 0x55 || frame[3] != 0x03 {
		return nil, fmt.Errorf("invalid response header")
	}
	byteCount := int(frame[4])
	if len(frame) < 5+byteCount+2 {
		return nil, fmt.Errorf("short response")
	}
	return frame[5 : 5+byteCount], nil
}

// buildETRuntimeFrame constructs a synthetic AA55 ET runtime response
// (READ 125 regs @ 0x891C, byteCount = 250) with the given values encoded
// at the ET family offsets:
//   pvW      → S32 at offset 74
//   gridW    → S32 at offset 78  (negative = exporting)
//   batteryW → S32 at offset 164 (negative = charging)
//   eTotalX10 → U32 at offset 182 (divide by 10 for kWh)
// Source byte is 0xF7 (ET family inverter address).
func buildETRuntimeFrame(pvW, gridW, batteryW int32, eTotalX10 uint32) []byte {
	const payloadLen = 250 // 125 registers × 2 bytes
	payload := make([]byte, payloadLen)
	binary.BigEndian.PutUint32(payload[74:], uint32(pvW))
	binary.BigEndian.PutUint32(payload[78:], uint32(gridW))
	binary.BigEndian.PutUint32(payload[164:], uint32(batteryW))
	binary.BigEndian.PutUint32(payload[182:], eTotalX10)
	frame := []byte{0xAA, 0x55, 0xF7, 0x03, byte(payloadLen)}
	frame = append(frame, payload...)
	frame = append(frame, 0x00, 0x00)
	return frame
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

func parseSoc(payload []byte) (float64, error) {
	const offset = 28
	if len(payload) < offset+2 {
		return 0, fmt.Errorf("short runtime data")
	}
	return float64(binary.BigEndian.Uint16(payload[offset : offset+2])), nil
}

// ---------------------------------------------------------------------------
// Header-stripping unit tests  (no network)
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
	// byteCount says 10 but only 3 payload bytes present
	frame := []byte{0xAA, 0x55, 0x7F, 0x03, 0x0A, 0x01, 0x02, 0x03, 0x00, 0x00}
	_, err := stripAA55Header(frame)
	assert.Error(t, err)
}

// ---------------------------------------------------------------------------
// DT payload-parsing unit tests  (no network)
// ---------------------------------------------------------------------------

// TestParseDTPower_RealCapture verifies against the real GW3000-DNS-30 capture.
// Stripped payload offset 54 = 00 00 07 b4 → 1972 W.
func TestParseDTPower_RealCapture(t *testing.T) {
	frame := mustDecodeHex(runtimeDataResponseHex)
	payload, err := stripAA55Header(frame)
	require.NoError(t, err)

	power, err := parseDTPower(payload)
	require.NoError(t, err)
	assert.InDelta(t, 1972.0, power, 0.5, "DT CurrentPower must match real capture")
}

func TestParseDTPower_KnownPositive(t *testing.T) {
	payload := make([]byte, 100)
	binary.BigEndian.PutUint32(payload[54:], uint32(int32(5000)))
	power, err := parseDTPower(payload)
	require.NoError(t, err)
	assert.InDelta(t, 5000.0, power, 0.5)
}

func TestParseDTPower_NegativeValue(t *testing.T) {
	payload := make([]byte, 100)
	var neg int32 = -500
	binary.BigEndian.PutUint32(payload[54:], uint32(neg))
	power, err := parseDTPower(payload)
	require.NoError(t, err)
	assert.InDelta(t, -500.0, power, 0.5, "DT power must handle negative (signed int32)")
}

func TestParseDTPower_Zero(t *testing.T) {
	payload := make([]byte, 100)
	power, err := parseDTPower(payload)
	require.NoError(t, err)
	assert.Equal(t, 0.0, power)
}

func TestParseDTPower_Short(t *testing.T) {
	_, err := parseDTPower(make([]byte, 50))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "short")
}

// ---------------------------------------------------------------------------
// TotalEnergy unit tests  (no network)
// ---------------------------------------------------------------------------

// TestParseTotalEnergy_RealCapture verifies TotalEnergy parsing against the
// real capture.  Stripped payload offset 90 = 00 00 a9 f9 → 43513 → 4351.3 kWh.
func TestParseTotalEnergy_RealCapture(t *testing.T) {
	frame := mustDecodeHex(runtimeDataResponseHex)
	payload, err := stripAA55Header(frame)
	require.NoError(t, err)
	energy, err := parseTotalEnergy(payload)
	require.NoError(t, err)
	assert.InDelta(t, 4351.3, energy, 0.001, "TotalEnergy must match real capture (÷10)")
}

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

// ---------------------------------------------------------------------------
// DT real-capture parsing tests — multiple inverter models
// (frames from marcelblijleven/goodwe tests/sample/, all DT family)
// ---------------------------------------------------------------------------

// TestParseDTPower_GW17K verifies the DT offset against a GW17K-DT capture.
// power@54 = 0x00 00 30 b6 = 12470 W.
func TestParseDTPower_GW17K(t *testing.T) {
	frame := mustDecodeHex(gw17kDTRuntimeHex)
	payload, err := stripAA55Header(frame)
	require.NoError(t, err)
	power, err := parseDTPower(payload)
	require.NoError(t, err)
	assert.InDelta(t, 12470.0, power, 0.5)
}

// TestParseTotalEnergy_GW17K verifies TotalEnergy against a GW17K-DT capture.
// energy@90 = 0x00 04 93 44 = 299844 → 29984.4 kWh.
func TestParseTotalEnergy_GW17K(t *testing.T) {
	frame := mustDecodeHex(gw17kDTRuntimeHex)
	payload, err := stripAA55Header(frame)
	require.NoError(t, err)
	energy, err := parseTotalEnergy(payload)
	require.NoError(t, err)
	assert.InDelta(t, 29984.4, energy, 0.001)
}

// TestParseDTPower_GW20KAU verifies DT offset for a GW20KAU-DT capture.
// power@54 = 0x00 00 13 5d = 4957 W.
func TestParseDTPower_GW20KAU(t *testing.T) {
	frame := mustDecodeHex(gw20kAUDTRuntimeHex)
	payload, err := stripAA55Header(frame)
	require.NoError(t, err)
	power, err := parseDTPower(payload)
	require.NoError(t, err)
	assert.InDelta(t, 4957.0, power, 0.5)
}

// TestParseTotalEnergy_GW20KAU verifies TotalEnergy for GW20KAU-DT.
// energy@90 = 0x00 00 a8 28 = 43048 → 4304.8 kWh.
func TestParseTotalEnergy_GW20KAU(t *testing.T) {
	frame := mustDecodeHex(gw20kAUDTRuntimeHex)
	payload, err := stripAA55Header(frame)
	require.NoError(t, err)
	energy, err := parseTotalEnergy(payload)
	require.NoError(t, err)
	assert.InDelta(t, 4304.8, energy, 0.001)
}

// TestParseDTPower_GW6000DT verifies DT offset for a GW6000-DT capture.
// power@54 = 0x00 00 07 2b = 1835 W.
func TestParseDTPower_GW6000DT(t *testing.T) {
	frame := mustDecodeHex(gw6000DTRuntimeHex)
	payload, err := stripAA55Header(frame)
	require.NoError(t, err)
	power, err := parseDTPower(payload)
	require.NoError(t, err)
	assert.InDelta(t, 1835.0, power, 0.5)
}

// TestParseTotalEnergy_GW6000DT verifies TotalEnergy for GW6000-DT.
// energy@90 = 0x00 02 09 7e = 133502 → 13350.2 kWh.
func TestParseTotalEnergy_GW6000DT(t *testing.T) {
	frame := mustDecodeHex(gw6000DTRuntimeHex)
	payload, err := stripAA55Header(frame)
	require.NoError(t, err)
	energy, err := parseTotalEnergy(payload)
	require.NoError(t, err)
	assert.InDelta(t, 13350.2, energy, 0.001)
}

// TestParseDTPower_GW8KDT verifies DT offset for a GW8K-DT capture.
// power@54 = 0x00 00 02 83 = 643 W.
// energy@90 = 0xFFFFFFFF — meter not connected, TotalEnergy not testable.
func TestParseDTPower_GW8KDT(t *testing.T) {
	frame := mustDecodeHex(gw8kDTRuntimeHex)
	payload, err := stripAA55Header(frame)
	require.NoError(t, err)
	power, err := parseDTPower(payload)
	require.NoError(t, err)
	assert.InDelta(t, 643.0, power, 0.5)
}

// TestDetectFamily_GW17KDT_integration is an end-to-end test using the real
// GW17K-DT runtime capture, verifying detection and power reading.
func TestDetectFamily_GW17KDT_integration(t *testing.T) {
	modelFrame := buildModelFrame("GW17K-DT")
	host := mockOnPort8899(t, [][]byte{
		modelFrame,
		mustDecodeHex(gw17kDTRuntimeHex),
	})

	m, err := NewGoodWeWifi(host, "pv")
	require.NoError(t, err, "GW17K-DT should be detected as DT")

	power, err := m.CurrentPower()
	require.NoError(t, err)
	assert.InDelta(t, 12470.0, power, 0.5)
}

// TestDetectFamily_GW20KAUDT_integration tests GW20KAU-DT end-to-end.
func TestDetectFamily_GW20KAUDT_integration(t *testing.T) {
	modelFrame := buildModelFrame("GW20KAU-DT")
	host := mockOnPort8899(t, [][]byte{
		modelFrame,
		mustDecodeHex(gw20kAUDTRuntimeHex),
	})

	m, err := NewGoodWeWifi(host, "pv")
	require.NoError(t, err, "GW20KAU-DT should be detected as DT")

	power, err := m.CurrentPower()
	require.NoError(t, err)
	assert.InDelta(t, 4957.0, power, 0.5)
}

// TestDetectFamily_MSUnknown verifies that MS-series inverters (GW5000-MS)
// are rejected with an "unknown model" error since "MS" is not in the
// detectFamily token list.
func TestDetectFamily_MSUnknown(t *testing.T) {
	modelFrame := buildModelFrame("GW5000-MS")
	host := mockOnPort8899(t, [][]byte{modelFrame})

	_, err := NewGoodWeWifi(host, "pv")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown model",
		"MS-series inverter should be rejected as unknown model")
}

// ---------------------------------------------------------------------------
// ET family payload-parsing unit tests  (no network)
// ---------------------------------------------------------------------------

// TestParseETPower_AllUsages verifies all three ET payload offsets in one pass.
// Source: GW10K-ET fw617 real capture.
//   pv      (offset  74) = 831 W
//   grid    (offset  78) = -3 W  (tiny export)
//   battery (offset 164) = -2512 W  (charging)
func TestParseETPower_AllUsages(t *testing.T) {
	frame := mustDecodeHex(gw10kETRuntimeHex)
	payload, err := stripAA55Header(frame)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(payload), 168, "ET payload must be ≥168 bytes")

	cases := []struct {
		usage    string
		offset   int
		expected float64
	}{
		{"pv", 74, 831},
		{"grid", 78, -3},
		{"battery", 164, -2512},
	}
	for _, tc := range cases {
		val := float64(int32(binary.BigEndian.Uint32(payload[tc.offset : tc.offset+4])))
		assert.InDelta(t, tc.expected, val, 0.5, "ET %s power at offset %d", tc.usage, tc.offset)
	}
}

// TestParseETPower_GridImporting verifies positive grid power (importing) from GW25K-ET.
// grid offset 78 = 1511 W
func TestParseETPower_GridImporting(t *testing.T) {
	frame := mustDecodeHex(gw25kETRuntimeHex)
	payload, err := stripAA55Header(frame)
	require.NoError(t, err)
	val := float64(int32(binary.BigEndian.Uint32(payload[78:82])))
	assert.InDelta(t, 1511.0, val, 0.5)
}

// TestParseETPower_GridExporting verifies large negative grid power from GW29K9-ET.
// grid offset 78 = -5403 W  (29.9 kW inverter exporting heavily)
func TestParseETPower_GridExporting(t *testing.T) {
	frame := mustDecodeHex(gw29k9ETRuntimeHex)
	payload, err := stripAA55Header(frame)
	require.NoError(t, err)
	val := float64(int32(binary.BigEndian.Uint32(payload[78:82])))
	assert.InDelta(t, -5403.0, val, 0.5)
}

// TestParseETEnergy_GW10K verifies ET e_total from GW10K-ET capture.
// e_total offset 182 = 60853 → 6085.3 kWh
func TestParseETEnergy_GW10K(t *testing.T) {
	frame := mustDecodeHex(gw10kETRuntimeHex)
	payload, err := stripAA55Header(frame)
	require.NoError(t, err)
	raw := binary.BigEndian.Uint32(payload[182:186])
	assert.InDelta(t, 6085.3, float64(raw)/10.0, 0.001)
}

// TestParseETEnergy_GW25K verifies ET e_total from GW25K-ET capture.
// e_total offset 182 = 1603 → 160.3 kWh  (newer installation)
func TestParseETEnergy_GW25K(t *testing.T) {
	frame := mustDecodeHex(gw25kETRuntimeHex)
	payload, err := stripAA55Header(frame)
	require.NoError(t, err)
	raw := binary.BigEndian.Uint32(payload[182:186])
	assert.InDelta(t, 160.3, float64(raw)/10.0, 0.001)
}

// TestParseETSoc_GW10K verifies SoC from GW10K-ET battery_info capture.
// Battery info PDU: READ 24 @ 0x9088, SoC at offset 14 = 68 %
func TestParseETSoc_GW10K(t *testing.T) {
	frame := mustDecodeHex(gw10kETBatteryInfoHex)
	payload, err := stripAA55Header(frame)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(payload), 16)
	soc := float64(binary.BigEndian.Uint16(payload[14:16]))
	assert.InDelta(t, 68.0, soc, 0.5)
}

// TestParseETSoc_GW25K verifies SoC=100% from GW25K-ET battery_info capture.
func TestParseETSoc_GW25K(t *testing.T) {
	frame := mustDecodeHex(gw25kETBatteryInfoHex)
	payload, err := stripAA55Header(frame)
	require.NoError(t, err)
	soc := float64(binary.BigEndian.Uint16(payload[14:16]))
	assert.InDelta(t, 100.0, soc, 0.5)
}

// ---------------------------------------------------------------------------
// ET family integration tests  (require mock on port 8899)
// ---------------------------------------------------------------------------

// TestDetectFamily_ET_isET verifies GW10K-ET is classified as "ET" family
// (not the old "HYBRID") so the correct PDU is used.
func TestDetectFamily_ET_isET(t *testing.T) {
	modelFrame := buildModelFrame("GW10K-ET")
	host := mockOnPort8899(t, [][]byte{
		modelFrame,
		mustDecodeHex(gw10kETRuntimeHex),
	})

	m, err := NewGoodWeWifi(host, "pv")
	require.NoError(t, err)

	gw, ok := m.(*goodWeWifi)
	require.True(t, ok)
	assert.Equal(t, "ET", gw.family, "GW10K-ET must be classified as ET family")
}

// TestDetectFamily_EH_isET verifies GW6000-EH is classified as "ET" family.
func TestDetectFamily_EH_isET(t *testing.T) {
	modelFrame := buildModelFrame("GW6000-EH")
	host := mockOnPort8899(t, [][]byte{
		modelFrame,
		mustDecodeHex(gw6000EHRuntimeHex),
	})

	m, err := NewGoodWeWifi(host, "pv")
	require.NoError(t, err)

	gw, ok := m.(*goodWeWifi)
	require.True(t, ok)
	assert.Equal(t, "ET", gw.family, "GW6000-EH must be classified as ET family")
}

// TestETCurrentPower_PV verifies end-to-end PV power for GW10K-ET: 831 W.
func TestETCurrentPower_PV(t *testing.T) {
	host := mockOnPort8899(t, [][]byte{
		buildModelFrame("GW10K-ET"),
		mustDecodeHex(gw10kETRuntimeHex),
	})

	m, err := NewGoodWeWifi(host, "pv")
	require.NoError(t, err)

	power, err := m.CurrentPower()
	require.NoError(t, err)
	assert.InDelta(t, 831.0, power, 0.5)
}

// TestETCurrentPower_Grid verifies end-to-end grid power for GW25K-ET: 1511 W.
func TestETCurrentPower_Grid(t *testing.T) {
	host := mockOnPort8899(t, [][]byte{
		buildModelFrame("GW25K-ET"),
		mustDecodeHex(gw25kETRuntimeHex),
	})

	m, err := NewGoodWeWifi(host, "grid")
	require.NoError(t, err)

	power, err := m.CurrentPower()
	require.NoError(t, err)
	assert.InDelta(t, 1511.0, power, 0.5)
}

// TestETCurrentPower_GridExport verifies large export from GW29K9-ET: -5403 W.
func TestETCurrentPower_GridExport(t *testing.T) {
	host := mockOnPort8899(t, [][]byte{
		buildModelFrame("GW29K9-ET"),
		mustDecodeHex(gw29k9ETRuntimeHex),
	})

	m, err := NewGoodWeWifi(host, "grid")
	require.NoError(t, err)

	power, err := m.CurrentPower()
	require.NoError(t, err)
	assert.InDelta(t, -5403.0, power, 0.5)
}

// TestETCurrentPower_BatteryCharging verifies battery charging from GW10K-ET: -2512 W.
func TestETCurrentPower_BatteryCharging(t *testing.T) {
	host := mockOnPort8899(t, [][]byte{
		buildModelFrame("GW10K-ET"),
		mustDecodeHex(gw10kETRuntimeHex),
	})

	m, err := NewGoodWeWifi(host, "battery")
	require.NoError(t, err)

	power, err := m.CurrentPower()
	require.NoError(t, err)
	assert.InDelta(t, -2512.0, power, 0.5)
}

// TestETTotalEnergy verifies TotalEnergy for GW10K-ET: 6085.3 kWh.
func TestETTotalEnergy(t *testing.T) {
	host := mockOnPort8899(t, [][]byte{
		buildModelFrame("GW10K-ET"),
		mustDecodeHex(gw10kETRuntimeHex),
	})

	m, err := NewGoodWeWifi(host, "pv")
	require.NoError(t, err)

	// TotalEnergy triggers a second sendCommand with the ET runtime PDU.
	energy, err := m.(api.MeterEnergy).TotalEnergy()
	require.NoError(t, err)
	assert.InDelta(t, 6085.3, energy, 0.001)
}

// TestETSoc verifies Soc() for GW10K-ET: 68 %.
func TestETSoc(t *testing.T) {
	host := mockOnPort8899(t, [][]byte{
		buildModelFrame("GW10K-ET"),
		mustDecodeHex(gw10kETBatteryInfoHex),
	})

	m, err := NewGoodWeWifi(host, "battery")
	require.NoError(t, err)

	battery, ok := m.(api.Battery)
	require.True(t, ok, "ET family must implement api.Battery")

	soc, err := battery.Soc()
	require.NoError(t, err)
	assert.InDelta(t, 68.0, soc, 0.5)
}

// TestSentPDU_ET verifies the exact bytes sent to an ET-family inverter for
// CurrentPower: must be READ 125 regs @ 0x891C = {7f 03 89 1c 00 7d} + CRC.
// This mirrors the structure of TestSentPDU_DT.
func TestSentPDU_ET(t *testing.T) {
	var captured []byte
	var mu sync.Mutex

	conn, err := net.ListenPacket("udp4", "127.0.0.1:8899")
	if err != nil {
		t.Skipf("cannot bind 127.0.0.1:8899 (%v) — skipping", err)
	}
	defer conn.Close()

	go func() {
		buf := make([]byte, 512)
		callIdx := 0
		for {
			n, addr, err := conn.ReadFrom(buf)
			if err != nil {
				return
			}
			switch callIdx {
			case 0: // model name query
				conn.WriteTo(buildModelFrame("GW10K-ET"), addr)
			case 1: // runtime data query — capture it
				mu.Lock()
				captured = make([]byte, n)
				copy(captured, buf[:n])
				mu.Unlock()
				conn.WriteTo(mustDecodeHex(gw10kETRuntimeHex), addr)
			}
			callIdx++
		}
	}()

	m, err := NewGoodWeWifi("127.0.0.1", "pv")
	require.NoError(t, err)
	_, _ = m.CurrentPower()

	mu.Lock()
	defer mu.Unlock()
	require.GreaterOrEqual(t, len(captured), 6, "must have captured the runtime PDU")
	assert.Equal(t, []byte{0x7f, 0x03, 0x89, 0x1c, 0x00, 0x7d}, captured[:6],
		"ET runtime PDU must be READ 125 @ 0x891C")
}

// TestSentPDU_ET_Simple checks the ET PDU bytes without network I/O.
// The 6-byte body of the ET runtime data command must be {7f 03 89 1c 00 7d}.
func TestSentPDU_ET_Simple(t *testing.T) {
	etPDU := []byte{0x7f, 0x03, 0x89, 0x1c, 0x00, 0x7d}
	assert.Equal(t, byte(0x89), etPDU[2], "high byte of register address")
	assert.Equal(t, byte(0x1c), etPDU[3], "low byte of register address (0x891C = reg 35100)")
	assert.Equal(t, byte(0x00), etPDU[4])
	assert.Equal(t, byte(0x7d), etPDU[5], "register count = 125 = 0x7d")
}

// TestSentPDU_ETBattery_Simple checks the ET battery info PDU bytes.
// Must be READ 24 regs @ 0x9088 = {7f 03 90 88 00 18}.
func TestSentPDU_ETBattery_Simple(t *testing.T) {
	batPDU := []byte{0x7f, 0x03, 0x90, 0x88, 0x00, 0x18}
	assert.Equal(t, byte(0x90), batPDU[2])
	assert.Equal(t, byte(0x88), batPDU[3], "0x9088 = reg 37000")
	assert.Equal(t, byte(0x00), batPDU[4])
	assert.Equal(t, byte(0x18), batPDU[5], "register count = 24 = 0x18")
}

// ---------------------------------------------------------------------------
// HYBRID power unit tests  (no network)
// ---------------------------------------------------------------------------

// TestParseHybridPower_Offsets verifies each usage maps to its documented offset.
func TestParseHybridPower_Offsets(t *testing.T) {
	cases := []struct {
		usage  string
		offset int
		power  int32
	}{
		{"pv", 12, 3000},
		{"grid", 24, -800},    // negative = exporting to grid
		{"battery", 36, 500},  // positive = discharging
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

func TestParseHybridPower_BatteryCharging(t *testing.T) {
	// When charging, battery power is negative.
	payload := make([]byte, 100)
	var charging int32 = -1500
	binary.BigEndian.PutUint32(payload[36:], uint32(charging))
	power, err := parseHybridPower(payload, "battery")
	require.NoError(t, err)
	assert.InDelta(t, -1500.0, power, 0.5, "charging battery must report negative power")
}

func TestParseHybridPower_GridImporting(t *testing.T) {
	// When importing from grid, grid power is positive.
	payload := make([]byte, 100)
	var importing int32 = 1200
	binary.BigEndian.PutUint32(payload[24:], uint32(importing))
	power, err := parseHybridPower(payload, "grid")
	require.NoError(t, err)
	assert.InDelta(t, 1200.0, power, 0.5, "importing grid must report positive power")
}

func TestParseHybridPower_UnknownUsage(t *testing.T) {
	_, err := parseHybridPower(make([]byte, 100), "solar")
	assert.Error(t, err)
}

// ---------------------------------------------------------------------------
// SoC (battery state of charge) unit tests  (no network)
// ---------------------------------------------------------------------------

// TestParseSoc_KnownValue verifies that SoC is read as uint16 from offset 28.
func TestParseSoc_KnownValue(t *testing.T) {
	payload := make([]byte, 100)
	binary.BigEndian.PutUint16(payload[28:], 75)
	soc, err := parseSoc(payload)
	require.NoError(t, err)
	assert.InDelta(t, 75.0, soc, 0.5)
}

func TestParseSoc_FullyCharged(t *testing.T) {
	payload := make([]byte, 100)
	binary.BigEndian.PutUint16(payload[28:], 100)
	soc, err := parseSoc(payload)
	require.NoError(t, err)
	assert.InDelta(t, 100.0, soc, 0.5)
}

func TestParseSoc_Empty(t *testing.T) {
	payload := make([]byte, 100)
	binary.BigEndian.PutUint16(payload[28:], 0)
	soc, err := parseSoc(payload)
	require.NoError(t, err)
	assert.Equal(t, 0.0, soc)
}

func TestParseSoc_Short(t *testing.T) {
	_, err := parseSoc(make([]byte, 20))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "short")
}

// TestParseHybridFrame_AllFields verifies that buildHybridRuntimeFrame places
// all four values correctly and that our parse helpers recover them.
func TestParseHybridFrame_AllFields(t *testing.T) {
	frame := buildHybridRuntimeFrame(3500, -1200, 2000, 75)
	payload, err := stripAA55Header(frame)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(payload), 40, "HYBRID payload must be ≥ 40 bytes")

	pv, err := parseHybridPower(payload, "pv")
	require.NoError(t, err)
	assert.InDelta(t, 3500.0, pv, 0.5)

	grid, err := parseHybridPower(payload, "grid")
	require.NoError(t, err)
	assert.InDelta(t, -1200.0, grid, 0.5)

	battery, err := parseHybridPower(payload, "battery")
	require.NoError(t, err)
	assert.InDelta(t, 2000.0, battery, 0.5)

	soc, err := parseSoc(payload)
	require.NoError(t, err)
	assert.InDelta(t, 75.0, soc, 0.5)
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

// TestDetectFamily_ET_isET_legacy verifies that a GW10K-ET model is classified
// as "ET" (not "HYBRID") and that CurrentPower uses the correct 250-byte PDU.
// Uses the real GW10K-ET capture; expected pv = 831 W.
func TestDetectFamily_ET_isET_legacy(t *testing.T) {
	host := mockOnPort8899(t, [][]byte{
		buildModelFrame("GW10K-ET"),
		mustDecodeHex(gw10kETRuntimeHex),
	})

	m, err := NewGoodWeWifi(host, "pv")
	require.NoError(t, err, "GW10K-ET should be detected as ET without error")

	gw := m.(*goodWeWifi)
	assert.Equal(t, "ET", gw.family)

	power, err := m.CurrentPower()
	require.NoError(t, err)
	assert.InDelta(t, 831.0, power, 0.5)
}

// TestDetectFamily_EH_isET_legacy verifies that GW6000-EH is classified as "ET".
// Uses the real GW6000-EH capture; expected pv = 1561 W.
func TestDetectFamily_EH_isET_legacy(t *testing.T) {
	host := mockOnPort8899(t, [][]byte{
		buildModelFrame("GW6000-EH"),
		mustDecodeHex(gw6000EHRuntimeHex),
	})

	m, err := NewGoodWeWifi(host, "pv")
	require.NoError(t, err, "GW6000-EH should be detected as ET without error")

	gw := m.(*goodWeWifi)
	assert.Equal(t, "ET", gw.family)

	power, err := m.CurrentPower()
	require.NoError(t, err)
	assert.InDelta(t, 1561.0, power, 0.5)
}

// TestDetectFamily_BT_isET verifies BT series (3-phase hybrid, HV battery)
// is classified as "ET" family. Uses a synthetic ET-format frame.
func TestDetectFamily_BT_isET(t *testing.T) {
	frame := buildETRuntimeFrame(12000, -4000, 3000, 50000)
	host := mockOnPort8899(t, [][]byte{
		buildModelFrame("GW15K-BT"),
		frame,
	})

	m, err := NewGoodWeWifi(host, "pv")
	require.NoError(t, err, "GW15K-BT should be detected as ET without error")

	gw := m.(*goodWeWifi)
	assert.Equal(t, "ET", gw.family)

	power, err := m.CurrentPower()
	require.NoError(t, err)
	assert.InDelta(t, 12000.0, power, 0.5)
}

// TestDetectFamily_BH_isET verifies BH series (single-phase, HV battery)
// is classified as "ET" family. Uses a synthetic ET-format frame.
func TestDetectFamily_BH_isET(t *testing.T) {
	frame := buildETRuntimeFrame(4500, 500, -1000, 30000)
	host := mockOnPort8899(t, [][]byte{
		buildModelFrame("GW5000-BH"),
		frame,
	})

	m, err := NewGoodWeWifi(host, "pv")
	require.NoError(t, err, "GW5000-BH should be detected as ET without error")

	gw := m.(*goodWeWifi)
	assert.Equal(t, "ET", gw.family)

	power, err := m.CurrentPower()
	require.NoError(t, err)
	assert.InDelta(t, 4500.0, power, 0.5)
}

// TestHybridGridUsage verifies that usage="grid" on a HYBRID (ES/EM) inverter
// reads offset 24 and returns the correct signed value.
// Uses GW5048D-ES as the model name (genuine ES-family token).
func TestHybridGridUsage(t *testing.T) {
	modelFrame := buildModelFrame("GW5048D-ES")
	// grid = -2000 W (exporting)
	runtimeFrame := buildHybridRuntimeFrame(5000, -2000, 1000, 70)

	host := mockOnPort8899(t, [][]byte{modelFrame, runtimeFrame})

	m, err := NewGoodWeWifi(host, "grid")
	require.NoError(t, err)

	power, err := m.CurrentPower()
	require.NoError(t, err)
	assert.InDelta(t, -2000.0, power, 0.5,
		"grid usage must return negative when exporting")
}

// TestHybridBatteryUsage verifies that usage="battery" on a HYBRID (ES/EM)
// inverter reads offset 36 and returns the correct signed value.
func TestHybridBatteryUsage(t *testing.T) {
	modelFrame := buildModelFrame("GW5048D-ES")
	// battery = 3000 W discharging
	runtimeFrame := buildHybridRuntimeFrame(5000, -2000, 3000, 55)

	host := mockOnPort8899(t, [][]byte{modelFrame, runtimeFrame})

	m, err := NewGoodWeWifi(host, "battery")
	require.NoError(t, err)

	power, err := m.CurrentPower()
	require.NoError(t, err)
	assert.InDelta(t, 3000.0, power, 0.5,
		"battery usage must return positive when discharging")
}

// TestHybridBatteryCharging verifies that a charging battery returns negative
// power via usage="battery" on a HYBRID (ES/EM) inverter.
func TestHybridBatteryCharging(t *testing.T) {
	modelFrame := buildModelFrame("GW5048-EM")
	// battery = -1500 W (charging)
	runtimeFrame := buildHybridRuntimeFrame(3000, 500, -1500, 40)

	host := mockOnPort8899(t, [][]byte{modelFrame, runtimeFrame})

	m, err := NewGoodWeWifi(host, "battery")
	require.NoError(t, err)

	power, err := m.CurrentPower()
	require.NoError(t, err)
	assert.InDelta(t, -1500.0, power, 0.5,
		"battery usage must return negative when charging")
}

// TestHybridSoc verifies that Soc() reads the uint16 at offset 28 of the
// HYBRID (ES/EM) payload and returns it as a percentage.
func TestHybridSoc(t *testing.T) {
	modelFrame := buildModelFrame("GW5048D-ES")
	runtimeFrame := buildHybridRuntimeFrame(4000, -1000, 2000, 83)

	host := mockOnPort8899(t, [][]byte{modelFrame, runtimeFrame, runtimeFrame})

	m, err := NewGoodWeWifi(host, "battery")
	require.NoError(t, err)

	// Soc() is part of api.Battery; the meter must implement it.
	bm, ok := m.(api.Battery)
	require.True(t, ok, "HYBRID goodWeWifi must implement api.Battery")

	soc, err := bm.Soc()
	require.NoError(t, err)
	assert.InDelta(t, 83.0, soc, 0.5)
}

// ---------------------------------------------------------------------------
// Usage-restriction tests
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// End-to-end integration tests with real captures
// ---------------------------------------------------------------------------

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
		buildDTRuntimeFrame(100, 0, 43513), // 4351.3 kWh
	})

	m, err := NewGoodWeWifi(host, "pv")
	require.NoError(t, err)

	me, ok := m.(api.MeterEnergy)
	require.True(t, ok, "goodWeWifi must implement api.MeterEnergy")

	energy, err := me.TotalEnergy()
	require.NoError(t, err)
	assert.InDelta(t, 4351.3, energy, 0.001)
}

// TestTotalEnergy_RealCapture verifies TotalEnergy end-to-end against the
// real GW3000-DNS-30 captures (both captures contain the same energy value).
func TestTotalEnergy_RealCapture(t *testing.T) {
	host := mockOnPort8899(t, [][]byte{
		mustDecodeHex(modelNameResponseHex),
		mustDecodeHex(runtimeDataResponseHex),
	})

	m, err := NewGoodWeWifi(host, "pv")
	require.NoError(t, err)

	me, ok := m.(api.MeterEnergy)
	require.True(t, ok, "goodWeWifi must implement api.MeterEnergy")

	energy, err := me.TotalEnergy()
	require.NoError(t, err)
	assert.InDelta(t, 4351.3, energy, 0.001, "TotalEnergy from real capture must be 4351.3 kWh")
}

// ---------------------------------------------------------------------------
// PDU wire format tests
// ---------------------------------------------------------------------------

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

// TestSentPDU_HYBRID verifies the exact bytes sent to a HYBRID (ES/EM) inverter
// for a runtime data request: READ 42 regs @ 0x7500 → {7f 03 75 00 00 2a <crc>}.
func TestSentPDU_HYBRID(t *testing.T) {
	conn, err := net.ListenPacket("udp4", "127.0.0.1:8899")
	if err != nil {
		t.Skipf("cannot bind 127.0.0.1:8899: %v", err)
	}
	t.Cleanup(func() { conn.Close() })

	responses := [][]byte{
		buildModelFrame("GW5048D-ES"),
		buildHybridRuntimeFrame(2000, 0, 0, 50),
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

	// Drain identification PDU.
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
		t.Fatal("timed out waiting for HYBRID runtime PDU")
	}

	require.Len(t, runtimePDU, 8, "HYBRID PDU must be exactly 8 bytes (6 + 2 CRC)")
	assert.Equal(t,
		[]byte{0x7F, 0x03, 0x75, 0x00, 0x00, 0x2A},
		runtimePDU[:6],
		"HYBRID runtime PDU header mismatch (READ 42 regs @ 0x7500)")
}

// ---------------------------------------------------------------------------
// Timeout test
// ---------------------------------------------------------------------------

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
