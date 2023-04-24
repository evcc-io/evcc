package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"nhooyr.io/websocket"
)

func TestSocketProvider(t *testing.T) {
	// ctx, cancel := context.WithTimeout(r.Context(), time.Second*10)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, nil)
		require.NoError(t, err)
		defer c.Close(websocket.StatusNormalClosure, "")

		for {
			if err := c.Write(ctx, websocket.MessageText, []byte(`{"version":"0.3","data":{"uuid":"8f32e780-a937-11ec-a5ac-redacted","tuples":[[1682319567986,-71,1]]}}`)); err != nil {
				require.NoError(t, err)
			}

			select {
			case <-time.Tick(time.Second):
			case <-ctx.Done():
				return
			}
		}
	}))

	defer srv.Close()

	addr := "ws://" + srv.Listener.Addr().String()
	p, err := NewSocketProviderFromConfig(map[string]any{
		"uri": addr,
		"jq":  ".data.tuples[0][1]",
	})
	require.NoError(t, err)

	g := p.IntGetter()
	i, err := g()
	require.NoError(t, err)
	require.Equal(t, int64(-71), i)
}
