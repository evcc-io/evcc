package server

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core"
	"github.com/evcc-io/evcc/core/site"
	"github.com/stretchr/testify/assert"
)

type riskProfileSite struct {
	site.API
	profile *api.RiskProfile
}

func (s riskProfileSite) GetOptimizerRiskProfile() *api.RiskProfile {
	return s.profile
}

func TestOptimizerRiskProfileGet(t *testing.T) {
	for _, tc := range []struct {
		profile *api.RiskProfile
		want    string
	}{
		{nil, "null"},
		{&api.RiskProfile{Solar: 0.3, Demand: 0.2, Budget: 8}, `{"solar":0.3,"demand":0.2,"budget":8}`},
	} {
		w := httptest.NewRecorder()
		optimizerRiskProfileHandler(riskProfileSite{profile: tc.profile})(w, httptest.NewRequest(http.MethodGet, "/optimizerriskprofile", nil))

		assert.Equal(t, http.StatusOK, w.Code)
		assert.JSONEq(t, tc.want, w.Body.String())
	}
}

func TestErrorStatus(t *testing.T) {
	// settings the optimizer decides on are rejected as conflict, not bad request
	assert.Equal(t, http.StatusConflict, errorStatus(core.ErrOptimizerAutomatic))
	assert.Equal(t, http.StatusConflict, errorStatus(fmt.Errorf("smart cost limit: %w", core.ErrOptimizerAutomatic)))
	assert.Equal(t, http.StatusBadRequest, errorStatus(errors.New("boom")))
}
