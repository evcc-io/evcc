package meter

import (
	"context"
	"io"
	"time"

	"github.com/coder/websocket"
	"github.com/evcc-io/evcc/util/request"
)

// wsReadTimeout bounds a single message read. The P1 stream is continuous
// (roughly one telegram per second), so a read blocking this long means the
// connection is dead and a reconnect is triggered.
const wsReadTimeout = 30 * time.Second

// wsDialer returns a transport dialer that connects to the Homey Energy Dongle
// WebSocket endpoint and exposes the P1 stream as an io.ReadCloser.
func wsDialer(ctx context.Context, uri string) func() (io.ReadCloser, error) {
	return func() (io.ReadCloser, error) {
		dialCtx, cancel := context.WithTimeout(ctx, request.Timeout)
		defer cancel()

		conn, _, err := websocket.Dial(dialCtx, uri, nil)
		if err != nil {
			return nil, err
		}
		return &wsReader{ctx: ctx, conn: conn}, nil
	}
}

// wsReader adapts a coder/websocket connection to an io.ReadCloser by handing
// out one buffered message at a time, so the DSMR stream parser sees a
// continuous byte stream. The per-message read timeout detects a dead peer;
// canceling ctx aborts a pending read.
type wsReader struct {
	ctx  context.Context
	conn *websocket.Conn
	buf  []byte
}

func (r *wsReader) Read(p []byte) (int, error) {
	for len(r.buf) == 0 {
		ctx, cancel := context.WithTimeout(r.ctx, wsReadTimeout)
		msgType, data, err := r.conn.Read(ctx)
		cancel()
		if err != nil {
			return 0, err
		}

		// The P1 telegram may be streamed as text or binary frames, depending on
		// the dongle firmware. Skip control or metadata frames.
		if msgType != websocket.MessageText && msgType != websocket.MessageBinary {
			continue
		}

		r.buf = data
	}

	n := copy(p, r.buf)
	r.buf = r.buf[n:]
	return n, nil
}

func (r *wsReader) Close() error {
	return r.conn.CloseNow()
}
