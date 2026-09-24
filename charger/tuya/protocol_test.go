package tuya

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testKey = []byte("0123456789abcdef")

// vector generated with tinytuya
func TestDecodeTinytuya(t *testing.T) {
	c, err := newCodec(testKey)
	require.NoError(t, err)

	key, err := c.deriveSessionKey([]byte("LLLLLLLLLLLLLLLL"), []byte("RRRRRRRRRRRRRRRR"))
	require.NoError(t, err)
	assert.Equal(t, "86fce4ea964f68132ec71b85283981d4", hex.EncodeToString(key))
	c.sessionKey = key

	vector, err := hex.DecodeString("000066990000000000070000000d0000003f781727a7026fc3df0cb5d813cb552003da3c0ab78e2c7a370c8ca022639df413260168aa71244951c4d4d211b04b0dd1d1bcb627aaa922d3543a860e87c5ec00009966")
	require.NoError(t, err)

	frame, err := readFrame(bufio.NewReader(bytes.NewReader(append([]byte{0xde, 0xad}, vector...))))
	require.NoError(t, err)

	msg, err := c.decode(frame)
	require.NoError(t, err)
	assert.Equal(t, uint32(7), msg.seq)
	assert.Equal(t, uint32(cmdControlNew), msg.cmd)
	assert.Equal(t, `{"dps":{"140":true}}`, string(msg.payload))
}

func TestRoundtrip(t *testing.T) {
	c, err := newCodec(testKey)
	require.NoError(t, err)
	c.sessionKey = []byte("fedcba9876543210")

	for _, cmd := range []uint32{cmdControlNew, cmdDpQueryNew} {
		frame, err := c.encode(42, cmd, []byte(`{"dps":{"150":10}}`))
		require.NoError(t, err)

		msg, err := c.decode(frame)
		require.NoError(t, err)
		assert.Equal(t, uint32(42), msg.seq)
		assert.Equal(t, cmd, msg.cmd)
		assert.Equal(t, `{"dps":{"150":10}}`, string(msg.payload))
	}
}

func TestDecodeRetcode(t *testing.T) {
	c, err := newCodec(testKey)
	require.NoError(t, err)

	frame, err := c.encode(1, cmdHeartbeat, append([]byte{0, 0, 0, 0}, []byte(`{}`)...))
	require.NoError(t, err)

	msg, err := c.decode(frame)
	require.NoError(t, err)
	assert.Equal(t, `{}`, string(msg.payload))
}

func TestDecodeTampered(t *testing.T) {
	c, err := newCodec(testKey)
	require.NoError(t, err)

	frame, err := c.encode(1, cmdControlNew, []byte(`{}`))
	require.NoError(t, err)

	frame[20] ^= 0xff
	_, err = c.decode(frame)
	assert.Error(t, err)
}
