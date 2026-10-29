package charger

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func daheimladenUTF16BE(s string) []byte {
	b := make([]byte, 0, 2*len(s))
	for _, r := range s {
		b = append(b, 0x00, byte(r))
	}
	return b
}

func TestDecodeStationID(t *testing.T) {
	id := "a6d2e04241e3979d73"

	tests := []struct {
		name string
		raw  []byte
		want string
	}{
		{"utf16be original firmware", daheimladenUTF16BE(id), id},
		// replacement comm module writes the id as ASCII, low byte first per register
		{"byte-swapped ascii replacement module", []byte("6a2d0e24143e79d937"), id},
		{"byte-swapped ascii odd length", []byte{0x62, 0x61, 0x00, 0x63, 0x00, 0x00}, "abc"},
		{"heidelbridge utf16be", daheimladenUTF16BE("HeidelBridge"), "HeidelBridge"},
		{"empty register map", make([]byte, 32), ""},
		{"non printable", []byte{0x80, 0x81, 0x82, 0x83}, "\x81\x80\x83\x82"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, decodeStationID(tc.raw))
		})
	}
}

func TestIsPrintableASCII(t *testing.T) {
	assert.True(t, isPrintableASCII("a6d2e04241e3979d73"))
	assert.False(t, isPrintableASCII(""))
	assert.False(t, isPrintableASCII("㙡㉤"))
}
