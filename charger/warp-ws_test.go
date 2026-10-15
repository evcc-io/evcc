package charger

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/coder/websocket"
	"github.com/evcc-io/evcc/charger/warp"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWarpWSEnergyManagerTopics(t *testing.T) {
	const messages = `{"topic":"meters/0/value_ids","payload":[74,209]}
{"topic":"meters/0/values","payload":[2,100]}
{"topic":"power_manager/state","payload":{"config_error_flags":0}}
`

	for _, tc := range []struct {
		name      string
		role      wsRole
		wantPower float64
		wantPM    bool
	}{
		{"main", wsRoleMain, 2, false},
		{"energy manager", wsRolePM, 0, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				conn, err := websocket.Accept(w, r, nil)
				if err != nil {
					return
				}
				defer conn.CloseNow()

				_ = conn.Write(r.Context(), websocket.MessageText, []byte(messages))
				conn.Close(websocket.StatusNormalClosure, "")
			}))
			defer srv.Close()

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			conn, _, err := websocket.Dial(ctx, "ws"+srv.URL[len("http"):], nil)
			require.NoError(t, err)

			w := &WarpWS{
				Connection: &warp.Connection{},
				pm:         &warp.Connection{},
				log:        util.NewLogger("foo"),
				features:   []string{warp.FeatureMeters},
				meterMap:   map[int]int{},
			}

			_ = w.handleConnection(ctx, tc.role, conn)

			assert.Equal(t, tc.wantPower, w.meter.Power)
			assert.Equal(t, tc.wantPM, w.pmState != nil)
		})
	}
}
