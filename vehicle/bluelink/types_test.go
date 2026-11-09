package bluelink

import (
	"encoding/json"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatusCCS(t *testing.T) {
	for _, tc := range []struct {
		green  string
		status api.ChargeStatus
	}{
		{`{"ChargingInformation":{"ConnectorFastening":{"State":0}}}`, api.StatusA},
		{`{"ChargingInformation":{"ConnectorFastening":{"State":1}}}`, api.StatusB},
		{`{"ChargingInformation":{"ConnectorFastening":{"State":1},"Charging":{"RemainTime":70}}}`, api.StatusC},
	} {
		var res StatusLatestResponseCCS
		require.NoError(t, json.Unmarshal([]byte(`{"resMsg":{"state":{"Vehicle":{"Green":`+tc.green+`}}}}`), &res))

		status, err := res.Status()
		require.NoError(t, err)
		assert.Equal(t, tc.status, status, tc.green)
	}
}
