package solarman

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCRC(t *testing.T) {
	data := []byte{0x01, 0x03, 0x00, 0x03, 0x00, 0x05}
	lo, hi := CRC(data)

	require.Equal(t, lo, byte(0x75))
	require.Equal(t, hi, byte(0xc9))
}
