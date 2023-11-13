package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"nhooyr.io/websocket"
)

func TestSocketProvider(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, nil)
		require.NoError(t, err)
		defer c.Close(websocket.StatusNormalClosure, "")

		uuids := []string{"foo", "bar"}
		for i := 0; ; i++ {
			json := fmt.Sprintf(`{"data":{"uuid":"%s","tuples":[[1682319567986,%d]]}}`, uuids[i%2], i%2)

			if err := c.Write(ctx, websocket.MessageText, []byte(json)); err != nil {
				require.NoError(t, err)
			}

			select {
			case <-time.Tick(time.Millisecond):
			case <-ctx.Done():
				return
			}
		}
	}))

	defer srv.Close()

	addr := "ws://" + srv.Listener.Addr().String()
	p, err := NewSocketProviderFromConfig(map[string]any{
		"uri": addr,
		"jq":  `.data | select(.uuid=="bar") .tuples[0][1]`,
	})
	require.NoError(t, err)

	<-p.(*Socket).val.Done()

	g := p.(IntProvider).IntGetter()
	i, err := g()
	require.NoError(t, err)
	require.Equal(t, int64(1), i)
}
