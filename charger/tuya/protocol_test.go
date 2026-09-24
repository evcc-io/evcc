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

func mustHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	require.NoError(t, err)
	return b
}

// vectors generated with tinytuya
func TestDecodeTinytuya(t *testing.T) {
	for _, tc := range []struct {
		version, session, frame string
		cmd                     uint32
	}{
		{"3.3", "", "000055aa000000070000000700000037332e330000000000000000000000009045be9a7874164e4e1b26bb0bcf7be6966b9c62350422878a31183c31bbd4a325f4e6390000aa55", cmdControl},
		{"3.4", "bc6a855faf3e089afe8721f5925ffef2", "000055aa000000070000000d0000005419e9a922443e8151612bc5d14ac50e868a0e5458ea415a35b5d4cc2b81ccefe28d9239fd8f0103e061b8634ed04836c520c2da7a264d654fa3796a46ce9e2e6b162832feafc38ac70854c642f72bbaae0000aa55", cmdControlNew},
		{"3.5", "86fce4ea964f68132ec71b85283981d4", "000066990000000000070000000d0000003f781727a7026fc3df0cb5d813cb552003da3c0ab78e2c7a370c8ca022639df413260168aa71244951c4d4d211b04b0dd1d1bcb627aaa922d3543a860e87c5ec00009966", cmdControlNew},
	} {
		t.Run(tc.version, func(t *testing.T) {
			c, err := newCodec(tc.version, testKey)
			require.NoError(t, err)

			if tc.session != "" {
				key, err := c.deriveSessionKey([]byte("LLLLLLLLLLLLLLLL"), []byte("RRRRRRRRRRRRRRRR"))
				require.NoError(t, err)
				assert.Equal(t, tc.session, hex.EncodeToString(key))
				c.sessionKey = key
			}

			frame, err := readFrame(bufio.NewReader(bytes.NewReader(append([]byte{0xde, 0xad}, mustHex(t, tc.frame)...))))
			require.NoError(t, err)

			msg, err := c.decode(frame)
			require.NoError(t, err)
			assert.Equal(t, uint32(7), msg.seq)
			assert.Equal(t, tc.cmd, msg.cmd)
			assert.Equal(t, `{"dps":{"140":true}}`, string(msg.payload))
		})
	}
}

func TestRoundtrip(t *testing.T) {
	for _, version := range []string{"3.3", "3.4", "3.5"} {
		t.Run(version, func(t *testing.T) {
			c, err := newCodec(version, testKey)
			require.NoError(t, err)

			if version != "3.3" {
				c.sessionKey = []byte("fedcba9876543210")
			}

			for _, cmd := range []uint32{cmdControlNew, cmdDpQueryNew} {
				frame, err := c.encode(42, cmd, []byte(`{"dps":{"150":10}}`))
				require.NoError(t, err)

				msg, err := c.decode(frame)
				require.NoError(t, err)
				assert.Equal(t, uint32(42), msg.seq)
				assert.Equal(t, cmd, msg.cmd)
				assert.Equal(t, `{"dps":{"150":10}}`, string(msg.payload))
			}
		})
	}
}

func TestDecodeRetcode(t *testing.T) {
	c, err := newCodec("3.5", testKey)
	require.NoError(t, err)

	frame, err := pack6699(testKey, 1, cmdHeartbeat, append([]byte{0, 0, 0, 0}, []byte(`{}`)...))
	require.NoError(t, err)

	msg, err := c.decode(frame)
	require.NoError(t, err)
	assert.Equal(t, `{}`, string(msg.payload))
}

func TestDecodeTampered(t *testing.T) {
	for _, version := range []string{"3.3", "3.4", "3.5"} {
		c, err := newCodec(version, testKey)
		require.NoError(t, err)

		frame, err := c.encode(1, cmdControlNew, []byte(`{}`))
		require.NoError(t, err)

		frame[20] ^= 0xff
		_, err = c.decode(frame)
		assert.Error(t, err, version)
	}
}

func TestReadFrameTooLarge(t *testing.T) {
	for _, frame := range []string{
		"000055aa00000001000000070fffffff",
		"000055aa0000000100000007ffffffff",
		"00006699000000000001000000070fffffff",
		"000066990000000000010000000700fffffffc",
	} {
		_, err := readFrame(bufio.NewReader(bytes.NewReader(mustHex(t, frame))))
		assert.ErrorContains(t, err, "frame too large", frame)
	}
}
