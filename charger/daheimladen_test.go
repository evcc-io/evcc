package charger

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func utf16BE(s string) []byte {
	b := make([]byte, 0, 2*len(s))
	for _, r := range s {
		b = append(b, 0x00, byte(r))
	}
	return b
}

func TestDecodeStationID(t *testing.T) {
	id := "6a2d0e24143e79d937"

	tc := []struct {
		name string
		raw  []byte
		want string
	}{
		{"utf16be original firmware", utf16BE(id), id},
		// replacement comm module writes the id as raw ASCII (issue #32907)
		{"raw ascii replacement module", []byte(id), id},
		{"heidelbridge utf16be", utf16BE("HeidelBridge"), "HeidelBridge"},
		{"empty register map", make([]byte, 32), ""},
		{"non printable", []byte{0x80, 0x81, 0x82}, "\x80\x81\x82"},
	}

	for _, tc := range tc {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, decodeStationID(tc.raw))
		})
	}
}

func TestIsPrintableASCII(t *testing.T) {
	assert.True(t, isPrintableASCII("6a2d0e24143e79d937"))
	assert.False(t, isPrintableASCII(""))
	assert.False(t, isPrintableASCII("㙡㉤"))
}
