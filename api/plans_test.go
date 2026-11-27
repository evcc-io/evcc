package api

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlanStrategyUnmarshalPartial(t *testing.T) {
	for _, tc := range []struct {
		payload string
		res     PlanStrategy
	}{
		{`{}`, PlanStrategy{Continuous: true, Precondition: 30 * time.Minute}},
		{`{"precondition":600}`, PlanStrategy{Continuous: true, Precondition: 10 * time.Minute}},
		{`{"continuous":false}`, PlanStrategy{Continuous: false, Precondition: 30 * time.Minute}},
		{`{"continuous":false,"precondition":0}`, PlanStrategy{}},
	} {
		ps := PlanStrategy{Continuous: true, Precondition: 30 * time.Minute}
		require.NoError(t, json.Unmarshal([]byte(tc.payload), &ps), tc.payload)
		assert.Equal(t, tc.res, ps, tc.payload)
	}
}

func TestPlanStrategyUnmarshalInvalid(t *testing.T) {
	ps := PlanStrategy{Continuous: true, Precondition: 30 * time.Minute}
	require.Error(t, json.Unmarshal([]byte(`{"continuous":1}`), &ps))
}
