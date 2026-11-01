package charger

import (
	"encoding/json"
	"testing"

	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestWarpWSStateRequestsLoadpointUpdate(t *testing.T) {
	lp := loadpoint.NewMockAPI(gomock.NewController(t))
	lp.EXPECT().Notify().Times(3)

	w := &WarpWS{log: util.NewLogger("warp")}
	w.LoadpointControl(lp)

	for _, state := range []string{
		`{"iec61851_state":1}`, // plugged
		`{"iec61851_state":1}`, // unchanged
		`{"iec61851_state":2}`, // charging
		`{"iec61851_state":3}`, // unknown
		`{"iec61851_state":4}`, // unknown, unchanged status
	} {
		require.NoError(t, w.handleEvent("evse/state", json.RawMessage(state)))
	}
}
