// Ported from shuette42/ecoflow-energy-ha (MIT), see the package documentation.

package ecoflow

import (
	"encoding/hex"
	"testing"

	"github.com/evcc-io/evcc/charger/ecoflow/ecoflowtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protowire"
)

// testSerial is the masked wallbox serial used by all fixtures. The real
// serial of the capture device has the same length, so byte layouts match.
const testSerial = "C376000000000001"

func mustHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	require.NoError(t, err)
	return b
}

func TestDescriptorBytes(t *testing.T) {
	assert.Equal(t, "08d701121043333736303030303030303030303031", hex.EncodeToString(descriptor(testSerial)))
	assert.Len(t, descriptor(testSerial), 21)
}

// 2/81 is a real app capture (ecoflow-energy-ha, serial masked as X). Start and
// stop are the frames that worked on the wallbox on 2026-10-23, serial masked.
func TestBuildMatchesCapturedAppFrames(t *testing.T) {
	tests := []struct {
		name string
		got  []byte
		want string
	}{
		{
			// ecoflow-energy-ha tests/fixtures/powerpulse/c376_standalone_current_ctrl_20260914.json
			name: "2/81 charge current 100 seq 205 (captured)",
			got:  BuildChargeCurrent("XXXXXXXXXXXXXXXX", 100, 205),
			want: "0a3a0a042864380210201802200128013803400248515004580170cd01800103880101ba0103696f73" +
				"ca011058585858585858585858585858585858",
		},
		{
			name: "241/100 start seq 243",
			got:  BuildStartStop(testSerial, 2, 243),
			want: "0a3d0a190a1508d7011210433337363030303030303030303030311002" +
				"1020186020012801380340f10148645019580170f301800103880101ba0103696f73",
		},
		{
			name: "241/100 stop seq 237",
			got:  BuildStartStop(testSerial, 1, 237),
			want: "0a3d0a190a1508d7011210433337363030303030303030303030311001" +
				"1020186020012801380340f10148645019580170ed01800103880101ba0103696f73",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, hex.EncodeToString(tc.got))
		})
	}
}

// Current writes as the wallbox executed them: 2/81 to the wallbox topic with
// the wallbox serial in the envelope, 13 A and 16 A while charging.
func TestBuildChargeCurrentMatchesCapturedFrames(t *testing.T) {
	for name, raw := range map[string]uint64{"current-13": 130, "current-16": 160} {
		t.Run(name, func(t *testing.T) {
			want := ecoflowtest.Write(name, 77)
			got := BuildChargeCurrent(ecoflowtest.Serial, raw, 77)
			assert.Equal(t, hex.EncodeToString(want.Payload), hex.EncodeToString(got))
		})
	}
}

// Phase writes as the wallbox executed them: 1p after a stop (stage 4b-2) and
// auto (stage 4b-5) on 2026-10-23, 3p after a stop on 2026-10-24. The charger
// itself never writes auto.
func TestBuildPhaseSetMatchesCapturedFrames(t *testing.T) {
	for name, mode := range map[string]uint64{"phase-1p": PhaseOne, "phase-3p": PhaseThree, "phase-auto": PhaseAuto} {
		t.Run(name, func(t *testing.T) {
			want := ecoflowtest.Write(name, 77)
			got := BuildPhaseSet(ecoflowtest.Serial, ecoflowtest.Gateway, mode, 77)
			assert.Equal(t, hex.EncodeToString(want.Payload), hex.EncodeToString(got))
		})
	}
}

func TestParseBuiltFrame(t *testing.T) {
	headers, err := ParseMessage(BuildPhaseSet(testSerial, "HJ35000000000001", 2, 88))
	require.NoError(t, err)
	require.Len(t, headers, 1)

	fn, id, ok := headers[0].Command()
	assert.True(t, ok)
	assert.Equal(t, uint64(241), fn)
	assert.Equal(t, uint64(102), id)

	seq, _ := headers[0].Varint(fSeq)
	assert.Equal(t, uint64(88), seq)
	length, _ := headers[0].Varint(fDataLen)
	assert.Equal(t, uint64(27), length)
}

// A payload may carry several concatenated headers. Taking only the first
// silently drops frames, so the parser must return all of them.
func TestParseRepeatedHeaders(t *testing.T) {
	msg := append(BuildStartStop(testSerial, 1, 10), BuildStartStop(testSerial, 2, 11)...)

	headers, err := ParseMessage(msg)
	require.NoError(t, err)
	require.Len(t, headers, 2)

	for i, want := range []uint64{10, 11} {
		seq, _ := headers[i].Varint(fSeq)
		assert.Equal(t, want, seq)
	}
}

func TestPdataUnmasking(t *testing.T) {
	plain := mustHex(t, "0a1508d7011210433337363030303030303030303030311001")

	tests := []struct {
		name    string
		encType uint64
		src     uint64
		seq     uint64
		masked  bool
	}{
		{name: "enc_type 1 from device is masked", encType: 1, src: 2, seq: 0x5A, masked: true},
		{name: "enc_type 0 is plaintext", encType: 0, src: 2, seq: 0x5A, masked: false},
		{name: "app-sent frame is never masked", encType: 1, src: srcApp, seq: 0x5A, masked: false},
		{name: "seq multiple of 256 gives key zero", encType: 1, src: 2, seq: 512, masked: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			body := plain
			if tc.masked {
				body = xorMask(plain, byte(tc.seq))
			}

			h := Header{fields: []field{
				{Num: fPdata, Wire: protowire.BytesType, B: body},
				{Num: fSrc, Wire: protowire.VarintType, V: tc.src},
				{Num: fEncType, Wire: protowire.VarintType, V: tc.encType},
				{Num: fSeq, Wire: protowire.VarintType, V: tc.seq},
			}}

			got := h.Pdata()
			require.Len(t, got, 1)
			assert.Equal(t, plain, got[0])
		})
	}
}

// Guards the "dev12 trap": reading enc_type from field 11 (need_ack) masked
// plaintext bodies, because every app SET carries need_ack = 1.
func TestNeedAckIsNotEncType(t *testing.T) {
	plain := mustHex(t, "0a1508d7011210433337363030303030303030303030311001")

	h := Header{fields: []field{
		{Num: fPdata, Wire: protowire.BytesType, B: plain},
		{Num: fSrc, Wire: protowire.VarintType, V: 2},
		{Num: fNeedAck, Wire: protowire.VarintType, V: 1},
		{Num: fSeq, Wire: protowire.VarintType, V: 0x5A},
	}}

	assert.Equal(t, plain, h.Pdata()[0], "need_ack was treated as enc_type")
}

// The get-all request carries no command and no pdata; the wallbox answers it
// with a bundled get_reply (2/33, 2/34, 241/44).
func TestBuildGetAll(t *testing.T) {
	headers, err := ParseMessage(BuildGetAll(1234))
	require.NoError(t, err)
	require.Len(t, headers, 1)
	h := headers[0]

	_, _, ok := h.Command()
	assert.False(t, ok, "get-all must not carry cmd_func/cmd_id")
	assert.Empty(t, h.Bytes(fPdata), "get-all must not carry pdata")

	for num, want := range map[int]uint64{fSrc: 32, fDest: 32, fSeq: 1234} {
		got, ok := h.Varint(num)
		assert.True(t, ok)
		assert.Equal(t, want, got, "field %d", num)
	}
	assert.Equal(t, [][]byte{[]byte("app")}, h.Bytes(fFrom))
}
