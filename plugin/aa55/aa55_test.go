package aa55

import (
	"encoding/hex"
	"testing"

	"github.com/evcc-io/evcc/util/modbus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Real captured frames (marcelblijleven/goodwe tests/sample/ + discussion #27411)
// ---------------------------------------------------------------------------
//
// All frames are verbatim UDP datagrams received from real inverters.
// In the per-register protocol each PDU fetches exactly one value; the
// response payload starts at offset 0.
//
// Register map summary:
//
//  Family  Reading   Register  Count  Decode    Expected (captures below)
//  DT      power     0x75AF    2      int32be   GW3000-DNS-30=1972W  GW17K-DT=12470W
//  DT      energy    0x75C1    2      uint32be  GW17K-DT=29984.4kWh  GW6000-DT=13350.2kWh
//  ES      pv        0x7506    2      int32be
//  ES      grid      0x750C    2      int32be
//  ES      battery   0x7512    2      int32be
//  ES      soc       0x750E    1      uint16be
//  ET      pv        0x8941    2      int32be   GW10K-ET=831W
//  ET      grid      0x8943    2      int32be   GW10K-ET=-3W  GW25K-ET=1511W  GW29K9-ET=-5403W
//  ET      battery   0x896E    2      int32be   GW10K-ET=-2512W (charging)
//  ET      energy    0x8977    2      uint32be  GW10K-ET=6085.3kWh  GW25K-ET=160.3kWh
//  ET      soc       0x908F    1      uint16be  GW10K-ET=68%  GW25K-ET=100%
//
// Note: these captures are full block-read responses used to verify the
// per-register values at offset 0 of what the inverter would return for
// a targeted single-register read. The payload bytes at the register's
// offset within the block are identical to what a per-register read returns.

const (
	// DT family (source byte 0x7F, block PDU READ 73 @ 0x7594)
	capGW3000DNS30 = `aa557f03921a020e0e301007cf005f053b000b00000000ffffffffffffffffffffffffffffffffffff08eeffffffff0056ffffffff1387ffffffff000007b40001000000000000000007a600000002ffffffff03e7ffff011bffffffff00140000a9f9000013ff0006ffffffffffffffffffffffffffffffffffffffffffffffffffff0e05ffffffffffff013e000000030cdaffff00393eb0`
	capGW17kDT     = `aa557f03921805140a23371518006912930094ffffffffffffffffffffffffffffffff102210130fff093f094f094500b000af00af138a138a138a000030b600010000000000000000000000000000000000000000ffff01c9ffffffff012500049344000020a500010000000000000000ffffffffffffffffffffffffffffffff0222184a0c4600000004000003a300f7000400000064b2f2`
	capGW20kAUDT   = `aa557f0392160a1513172a0f4100440dbc0047ffffffffffffffffffffffffffffffff0f2d0f4d0f6908d508bc08eb0048004a00471384138413850000135d000100000000000000000000000000cd0000000003e7ffff016cffffffff00c60000a8280000047300200000000000000000ffffffffffffffffffffffffffffffff0000174b0bad000000040000044b00000004000000696b04`
	capGW6000DT    = `aa557f039215081f0c03020c88001f0ca90020ffffffffffffffffffffffffffffffffffffffffffff08d008f90906001b001a001b1386138613860000072b0001000000000000ffffffffffffffffffffffff0000ffff019dffffffff003c0002097e0000210300140000ffff0000ffff0000ffff0000ffffffffffffffffffff0000177c0beeffffffff00cf016302f00000000000649f03`

	// ET family (source byte 0xF7, block PDU READ 125 @ 0x891C)
	capGW10kET  = `aa55f703fa1508160b0b0c0cfe00330000069f0cfe0035000006e100000000000000000000000000000000000002020959000f138700000150096f000d13870000011f096b000b1387000000ce00010000033ffffffffd000000000000000009560006138600010000006b096d000913880001000000bd096c00021387000100000000000000e000000050000000e9000001380000020a000401fe0000024b00001f640fb209eeff9efffff63000030000002000010000000000000000edb50000007d0000b8520000241e00620000024400000001588a007400006bbd003500005f65001d0005000000010000000000000000000107000800000209ee000055ae`
	capGW25kET  = `aa55f703fa170c030e07071cd3000e000004091cd30000000003d51d82000d000000001d82000000000000000002020905001d13830000024d0906001b1385000002290900002a138500000323000100000799000005e7000004d7000008d308f7001e138300000000003408fc0012138500000000000f08f6002013850000000001580000002c0000001000000153000001980000001a000701ce000001ae00001e350f1a0868000000000000000200000020000100000000000000000643000000930000056100000184001d00000094000a000000ac000200000391006e000002b800000004000000000000000000000000000002040180000200008f005ece`
	capGW29k9ET = `aa55f703fa1801110e310e1aad000f000001de1aad0000000002a7168d001200000186168d000000000000000202020909001d1387000002470919001b1387000002350920001d13850000024b0001000006c7ffffeae500000133000007b708fb00071386000000000015090b0007138800000000000509190006138500000000002500000287000002800000028b0000004200001ba0000100f1000000cd00001db40eda0000ffff0000000000000000002000010000000000000000b237000000090000af6100000497000c0000005700000001a39e01b600000000000000000000000000060000000000000000000000000000020400ce00000000030064b6`

	// ET battery info (source byte 0xF7, block PDU READ 24 @ 0x9088)
	capGW10kETBattery = `aa55f7033000ff01000001015e001900190000004400630005000001010000000000000000000000000000000000000000000000006447`
	capGW25kETBattery = `aa55f7033000ff0137000100e600000028000000640064000400000105000000000316000000000000000000000000000000000000dc7a`
)

// ---------------------------------------------------------------------------
// buildPDU
// ---------------------------------------------------------------------------

func TestBuildPDU_DTpower(t *testing.T) {
	got := buildPDU(0x7F, 0x75AF, 2)
	assert.Equal(t, []byte{0x7f, 0x03, 0x75, 0xaf, 0x00, 0x02}, got)
}

func TestBuildPDU_DefaultAddress(t *testing.T) {
	// When address is omitted from config, InverterAddr (0x7F) must be used.
	// This guards existing DT/DNS and ES/EM setups that rely on the default.
	got := buildPDU(InverterAddr, 0x75AF, 2)
	assert.Equal(t, byte(0x7F), got[0], "default address byte must be 0x7F")
}

func TestBuildPDU_ETgrid(t *testing.T) {
	got := buildPDU(0xF7, 0x8943, 2)
	assert.Equal(t, []byte{0xf7, 0x03, 0x89, 0x43, 0x00, 0x02}, got)
}

func TestBuildPDU_SoC(t *testing.T) {
	got := buildPDU(0xF7, 0x908F, 1)
	assert.Equal(t, []byte{0xf7, 0x03, 0x90, 0x8f, 0x00, 0x01}, got)
}

// ---------------------------------------------------------------------------
// write PDUs
// ---------------------------------------------------------------------------

func TestBuildWriteSinglePDU(t *testing.T) {
	// EMSPowerMode (47511 = 0xB997) := 1
	got := buildWriteSinglePDU(0xF7, 47511, 1)
	assert.Equal(t, []byte{0xf7, 0x06, 0xb9, 0x97, 0x00, 0x01}, got)
}

func TestBuildWriteMultiplePDU(t *testing.T) {
	// EMSPowerSet (47512 = 0xB998) := 10000 (0x2710), one register
	got := buildWriteMultiplePDU(0xF7, 47512, []byte{0x27, 0x10})
	assert.Equal(t, []byte{0xf7, 0x10, 0xb9, 0x98, 0x00, 0x01, 0x02, 0x27, 0x10}, got)
}

func TestValidateWriteResponse(t *testing.T) {
	// echoed function code → accepted
	require.NoError(t, validateWriteResponse([]byte{0xAA, 0x55, 0xF7, 0x06, 0xb9, 0x97, 0x00, 0x01}, 0x06))
	// high bit set → inverter rejected the write
	require.Error(t, validateWriteResponse([]byte{0xAA, 0x55, 0xF7, 0x86, 0x02}, 0x06))
	// wrong function code
	require.Error(t, validateWriteResponse([]byte{0xAA, 0x55, 0xF7, 0x03, 0x00}, 0x06))
	// bad magic
	require.Error(t, validateWriteResponse([]byte{0xFF, 0x55, 0xF7, 0x06}, 0x06))
}

// ---------------------------------------------------------------------------
// stripHeader
// ---------------------------------------------------------------------------

func TestStripHeader_DT(t *testing.T) {
	payload, err := stripHeader(mustHex(t, capGW3000DNS30))
	require.NoError(t, err)
	assert.Equal(t, 146, len(payload))
}

func TestStripHeader_ET(t *testing.T) {
	payload, err := stripHeader(mustHex(t, capGW10kET))
	require.NoError(t, err)
	assert.Equal(t, 250, len(payload))
}

func TestStripHeader_BadMagic(t *testing.T) {
	_, err := stripHeader([]byte{0xFF, 0x55, 0x7F, 0x03, 0x04, 0x01, 0x02, 0x03, 0x04, 0x00, 0x00})
	require.Error(t, err)
}

func TestStripHeader_Short(t *testing.T) {
	_, err := stripHeader([]byte{0xAA, 0x55, 0x7F, 0x03, 0x10, 0x01, 0x02})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "short")
}

// ---------------------------------------------------------------------------
// modbusCRC16
// ---------------------------------------------------------------------------

func TestModbusCRC16_DTPdu(t *testing.T) {
	pdu := buildPDU(0x7F, 0x75AF, 2)
	crc := modbusCRC16(pdu)
	assert.Len(t, crc, 2)
	assert.Equal(t, crc, modbusCRC16(pdu), "CRC must be deterministic")
}

func TestModbusCRC16_ETPdu(t *testing.T) {
	pdu := buildPDU(0xF7, 0x8943, 2)
	crc := modbusCRC16(pdu)
	assert.Len(t, crc, 2)
	assert.Equal(t, crc, modbusCRC16(pdu))
}

func TestModbusCRC16_KnownValue(t *testing.T) {
	// The original block-read DT PDU 7f 03 75 94 00 49 → CRC d5 c2.
	// Known-good value verified against real hardware.
	pdu := buildPDU(0x7F, 0x7594, 0x49)
	assert.Equal(t, []byte{0xd5, 0xc2}, modbusCRC16(pdu))
}

// ---------------------------------------------------------------------------
// Real-capture register value tests
//
// These verify that extracting bytes at the register's offset within a
// block-read capture gives the same value a per-register read would return
// at offset 0. This is the core correctness guarantee for the register map.
// ---------------------------------------------------------------------------

func TestDT_Power_GW3000DNS30(t *testing.T) {
	assertBlockOffset(t, capGW3000DNS30, 54, "int32", 1.0, 1972.0)
}

func TestDT_Power_GW17K(t *testing.T) {
	assertBlockOffset(t, capGW17kDT, 54, "int32", 1.0, 12470.0)
}

func TestDT_Power_GW20KAU(t *testing.T) {
	assertBlockOffset(t, capGW20kAUDT, 54, "int32", 1.0, 4957.0)
}

func TestDT_Energy_GW17K(t *testing.T) {
	assertBlockOffset(t, capGW17kDT, 90, "uint32", 0.1, 29984.4)
}

func TestDT_Energy_GW6000(t *testing.T) {
	assertBlockOffset(t, capGW6000DT, 90, "uint32", 0.1, 13350.2)
}

func TestDT_Energy_GW20KAU(t *testing.T) {
	assertBlockOffset(t, capGW20kAUDT, 90, "uint32", 0.1, 4304.8)
}

func TestET_PV_GW10K(t *testing.T) {
	assertBlockOffset(t, capGW10kET, 74, "int32", 1.0, 831.0)
}

func TestET_Grid_GW10K_TinyExport(t *testing.T) {
	assertBlockOffset(t, capGW10kET, 78, "int32", 1.0, -3.0)
}

func TestET_Grid_GW25K_Importing(t *testing.T) {
	assertBlockOffset(t, capGW25kET, 78, "int32", 1.0, 1511.0)
}

func TestET_Grid_GW29K9_Exporting(t *testing.T) {
	assertBlockOffset(t, capGW29k9ET, 78, "int32", 1.0, -5403.0)
}

func TestET_Battery_GW10K_Charging(t *testing.T) {
	assertBlockOffset(t, capGW10kET, 164, "int32", 1.0, -2512.0)
}

func TestET_Energy_GW10K(t *testing.T) {
	assertBlockOffset(t, capGW10kET, 182, "uint32", 0.1, 6085.3)
}

func TestET_Energy_GW25K(t *testing.T) {
	assertBlockOffset(t, capGW25kET, 182, "uint32", 0.1, 160.3)
}

func TestET_SoC_GW10K(t *testing.T) {
	assertBlockOffset(t, capGW10kETBattery, 14, "uint16", 1.0, 68.0)
}

func TestET_SoC_GW25K(t *testing.T) {
	assertBlockOffset(t, capGW25kETBattery, 14, "uint16", 1.0, 100.0)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func mustHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	require.NoError(t, err)
	return b
}

func assertBlockOffset(t *testing.T, capHex string, offset int, decode string, scale, expected float64) {
	t.Helper()
	payload, err := stripHeader(mustHex(t, capHex))
	require.NoError(t, err)
	reg := modbus.Register{Type: "holding", Decode: decode}
	length, err := reg.Length()
	require.NoError(t, err)
	fn, err := reg.DecodeFunc()
	require.NoError(t, err)
	v := fn(payload[offset : offset+int(length)*2])
	assert.InDelta(t, expected, v*scale, 0.05)
}
