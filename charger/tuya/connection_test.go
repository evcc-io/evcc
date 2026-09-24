package tuya

import (
	"bufio"
	"context"
	"crypto/hmac"
	"net"
	"testing"

	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeDevice emulates a 3.5 device: session key negotiation, status query and control
func fakeDevice(t *testing.T, l net.Listener, control chan<- string) {
	conn, err := l.Accept()
	if err != nil {
		return
	}
	defer conn.Close()

	r := bufio.NewReader(conn)
	dev, _ := newCodec("3.5", testKey)

	recv := func() (message, error) {
		frame, err := readFrame(r)
		if err != nil {
			return message{}, err
		}
		return dev.decode(frame)
	}

	respond := func(cmd uint32, payload []byte) error {
		frame, err := pack6699(dev.key(), 1, cmd, append([]byte{0, 0, 0, 0}, payload...))
		if err == nil {
			_, err = conn.Write(frame)
		}
		return err
	}

	start, err := recv()
	if !assert.NoError(t, err) || !assert.Equal(t, uint32(cmdSessKeyNegStart), start.cmd) {
		return
	}

	remoteNonce := []byte("RRRRRRRRRRRRRRRR")
	if !assert.NoError(t, respond(cmdSessKeyNegResp, append(remoteNonce, hmacSum(testKey, start.payload)...))) {
		return
	}

	finish, err := recv()
	if !assert.NoError(t, err) || !assert.True(t, hmac.Equal(hmacSum(testKey, remoteNonce), finish.payload)) {
		return
	}

	if dev.sessionKey, err = dev.deriveSessionKey(start.payload, remoteNonce); !assert.NoError(t, err) {
		return
	}

	for {
		msg, err := recv()
		if err != nil {
			return
		}

		switch msg.cmd {
		case cmdDpQueryNew:
			assert.NoError(t, respond(cmdDpQueryNew, []byte(`{"dps":{"109":"IDLEINS","150":10}}`)))
		case cmdControlNew:
			control <- string(msg.payload)
		}
	}
}

func TestConnection(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer l.Close()

	control := make(chan string, 1)
	go fakeDevice(t, l, control)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	conn, err := NewConnection(ctx, util.NewLogger("test"), l.Addr().String(), "bf01", string(testKey), "3.5")
	require.NoError(t, err)

	dps, err := conn.DpsContext(ctx)
	require.NoError(t, err)
	assert.Equal(t, "IDLEINS", dps["109"])
	assert.Equal(t, float64(10), dps["150"])

	require.NoError(t, conn.Set(map[string]any{"150": int64(0)}))
	assert.Contains(t, <-control, `"data":{"dps":{"150":0}}`)

	dps, err = conn.Dps()
	require.NoError(t, err)
	assert.Equal(t, float64(0), dps["150"], "written value cached")
}
