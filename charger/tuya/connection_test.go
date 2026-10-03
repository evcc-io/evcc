package tuya

import (
	"bufio"
	"context"
	"crypto/hmac"
	"encoding/json"
	"maps"
	"net"
	"testing"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// status report pushed by the device
const cmdStatus = 0x08

// deviceFrame encodes a device response including return code
func deviceFrame(dev *codec, cmd uint32, payload []byte) ([]byte, error) {
	retcode := []byte{0, 0, 0, 0}

	if dev.version == "3.5" {
		return dev.encode(1, cmd, append(retcode, payload...))
	}

	enc, err := ecbEncrypt(dev.key(), pkcs7Pad(payload))
	if err != nil {
		return nil, err
	}

	var mac []byte
	if dev.version == "3.4" {
		mac = dev.key()
	}

	return pack55AA(mac, 1, cmd, append(retcode, enc...)), nil
}

// fakeDevice emulates a device, closing connections that use a different protocol version
func fakeDevice(t *testing.T, l net.Listener, version string, control chan<- string) {
	for {
		conn, err := l.Accept()
		if err != nil {
			return
		}

		fakeSession(t, conn, version, control)
	}
}

func fakeSession(t *testing.T, conn net.Conn, version string, control chan<- string) {
	defer conn.Close()

	r := bufio.NewReader(conn)
	dev, _ := newCodec(version, testKey)

	recv := func() (message, error) {
		frame, err := readFrame(r)
		if err != nil {
			return message{}, err
		}
		return dev.decode(frame)
	}

	respond := func(cmd uint32, payload []byte) error {
		frame, err := deviceFrame(dev, cmd, payload)
		if err == nil {
			_, err = conn.Write(frame)
		}
		return err
	}

	state := map[string]any{"101": 200, "150": 10}
	status := func() []byte {
		b, _ := json.Marshal(map[string]any{"dps": state})
		return b
	}

	first, err := recv()
	if err != nil {
		// different protocol version
		return
	}

	if version == "3.3" {
		if !assert.Equal(t, uint32(cmdDpQuery), first.cmd) || !assert.NoError(t, respond(cmdDpQuery, status())) {
			return
		}
	} else {
		if !assert.Equal(t, uint32(cmdSessKeyNegStart), first.cmd) {
			return
		}

		remoteNonce := []byte("RRRRRRRRRRRRRRRR")
		if !assert.NoError(t, respond(cmdSessKeyNegResp, append(remoteNonce, hmacSum(testKey, first.payload)...))) {
			return
		}

		finish, err := recv()
		if !assert.NoError(t, err) || !assert.True(t, hmac.Equal(hmacSum(testKey, remoteNonce), finish.payload)) {
			return
		}

		if dev.sessionKey, err = dev.deriveSessionKey(first.payload, remoteNonce); !assert.NoError(t, err) {
			return
		}
	}

	for {
		msg, err := recv()
		if err != nil {
			return
		}

		switch msg.cmd {
		case cmdDpQuery, cmdDpQueryNew:
			assert.NoError(t, respond(msg.cmd, status()))
		case cmdControl, cmdControlNew:
			var req struct {
				Dps  map[string]any `json:"dps"`
				Data struct {
					Dps map[string]any `json:"dps"`
				} `json:"data"`
			}
			if assert.NoError(t, json.Unmarshal(msg.payload, &req)) {
				maps.Copy(state, req.Dps)
				maps.Copy(state, req.Data.Dps)
			}
			control <- string(msg.payload)
			assert.NoError(t, respond(cmdStatus, status()))
		}
	}
}

func TestConnection(t *testing.T) {
	for _, tc := range []struct {
		version, control string
	}{
		{"3.5", `"data":{"dps":{"150":0}}`},
		{"3.4", `"data":{"dps":{"150":0}}`},
		{"3.3", `"dps":{"150":0}`},
	} {
		t.Run(tc.version, func(t *testing.T) {
			l, err := net.Listen("tcp", "127.0.0.1:0")
			require.NoError(t, err)
			defer l.Close()

			control := make(chan string, 1)
			go fakeDevice(t, l, tc.version, control)

			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			conn, err := NewConnection(ctx, util.NewLogger("test"), l.Addr().String(), "bf01", string(testKey))
			require.NoError(t, err)

			dps, err := conn.DpsContext(ctx)
			require.NoError(t, err)
			assert.Equal(t, float64(200), dps["101"])
			assert.Equal(t, float64(10), dps["150"])

			require.NoError(t, conn.Set(map[string]any{"150": int64(0)}))
			assert.Contains(t, <-control, tc.control)

			assert.Eventually(t, func() bool {
				dps, err := conn.Dps()
				return err == nil && dps["150"] == float64(0)
			}, time.Second, 10*time.Millisecond, "written value cached")
		})
	}
}
