package modbus

import (
	"encoding/binary"
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLength(t *testing.T) {
	tc := []struct {
		value string
		want  uint16
	}{
		{"bool8", 1},
		{"int16", 1},
		{"float32", 2},
		{"uint64s", 4},
	}

	for _, tc := range tc {
		res, err := Register{Encoding: tc.value}.Length()
		require.NoError(t, err, tc)
		require.Equal(t, tc.want, res, tc)
	}
}

func TestEncoding(t *testing.T) {
	v32 := math.Float32bits(float32(0x12345678))

	var b32, b32s [4]byte
	binary.BigEndian.PutUint32(b32[:], v32)
	binary.BigEndian.PutUint32(b32s[:], v32>>16|v32&0xFFFF<<16)

	var b64 [8]byte
	binary.BigEndian.PutUint64(b64[:], math.Float64bits(float64(0x12345678)))

	tc := []struct {
		r   Register
		in  float64
		out []byte
	}{
		{Register{Encoding: "int16"}, 0x1234, []byte{0x12, 0x34}},
		{Register{Encoding: "uint16"}, 0x1234, []byte{0x12, 0x34}},
		{Register{Encoding: "int32"}, 0x12345678, []byte{0x12, 0x34, 0x56, 0x78}},
		{Register{Encoding: "uint32"}, 0x12345678, []byte{0x12, 0x34, 0x56, 0x78}},
		{Register{Encoding: "int32s"}, 0x12345678, []byte{0x56, 0x78, 0x12, 0x34}},
		{Register{Encoding: "uint32s"}, 0x12345678, []byte{0x56, 0x78, 0x12, 0x34}},
		{Register{Encoding: "float32"}, 0x12345678, b32[:]},
		{Register{Encoding: "ieee754"}, 0x12345678, b32[:]},
		{Register{Encoding: "float32s"}, 0x12345678, b32s[:]},
		{Register{Encoding: "ieee754s"}, 0x12345678, b32s[:]},
		{Register{Encoding: "float64"}, 0x12345678, b64[:]},
	}

	for _, tc := range tc {
		fun, err := tc.r.EncodeFunc()
		require.NoError(t, err, tc)

		res, err := fun(tc.in)
		require.NoError(t, err, tc)
		require.Equal(t, tc.out, res, tc)
	}
}
