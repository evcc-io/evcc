package charger

import (
	"testing"

	"github.com/evcc-io/evcc/charger/zaptec"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/require"
)

func TestZaptecDetectVersion(t *testing.T) {
	for _, tc := range []struct {
		name string
		res  zaptec.StateResponse
		want int
	}{
		{"missing capabilities", zaptec.StateResponse{{StateId: zaptec.ChargerOperationMode, ValueAsString: "1"}}, zaptec.ZaptecGo1_Pro},
		{"empty capabilities", zaptec.StateResponse{{StateId: zaptec.Capabilities}}, zaptec.ZaptecGo1_Pro},
		{"go2", zaptec.StateResponse{{StateId: zaptec.Capabilities, ValueAsString: `{"productVariant":"Go2"}`}}, zaptec.ZaptecGo2},
		{"pro", zaptec.StateResponse{{StateId: zaptec.Capabilities, ValueAsString: `{"productVariant":"Smart"}`}}, zaptec.ZaptecGo1_Pro},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := &Zaptec{
				statusG: util.ResettableCached(func() (zaptec.StateResponse, error) { return tc.res, nil }, 0),
			}

			v, err := c.detectVersion()
			require.NoError(t, err)
			require.Equal(t, tc.want, v)
		})
	}
}
