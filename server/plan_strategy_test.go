package server

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlanStrategySetters(t *testing.T) {
	current := api.PlanStrategy{Continuous: true, Precondition: 30 * time.Minute}

	for _, tc := range []struct {
		payload string
		res     api.PlanStrategy
	}{
		{`{"precondition":600}`, api.PlanStrategy{Continuous: true, Precondition: 10 * time.Minute}},
		{`{"continuous":false}`, api.PlanStrategy{Continuous: false, Precondition: 30 * time.Minute}},
		{`{"continuous":false,"precondition":0}`, api.PlanStrategy{}},
	} {
		get := func() api.PlanStrategy { return current }

		var res api.PlanStrategy
		set := func(ps api.PlanStrategy) error {
			res = ps
			return nil
		}

		// http
		r := httptest.NewRequest("POST", "/", strings.NewReader(tc.payload))
		require.NoError(t, planStrategyHandlerSetter(r, get, set), tc.payload)
		assert.Equal(t, tc.res, res, "http: "+tc.payload)

		// mqtt
		res = api.PlanStrategy{}
		require.NoError(t, planStrategySetter(get, set)(tc.payload), tc.payload)
		assert.Equal(t, tc.res, res, "mqtt: "+tc.payload)
	}
}
