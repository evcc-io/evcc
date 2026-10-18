package server

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/evcc-io/evcc/util/auth"
	"github.com/stretchr/testify/require"
)

type pvResolver struct {
	calls                     int
	battery, action, revision string
	err                       error
}

func (p *pvResolver) ResolveBatteryPVControl(b, a, r string) error {
	p.calls++
	p.battery, p.action, p.revision = b, a, r
	return p.err
}

func TestBatteryPVDecisionValidationAndAuthentication(t *testing.T) {
	valid := `{"battery":"db:3","action":"recover","revision":"current"}`
	for _, tc := range []struct {
		body string
		want int
	}{
		{valid, 200},
		{`{"battery":"db:3","action":"disable","revision":"current"}`, 200},
		{`{"battery":"db:3","action":"recover"}`, 400},
		{`{"battery":"db:3","action":"charge","revision":"current"}`, 400},
		{valid + `{}`, 400}, {`{"extra":true}`, 400},
	} {
		p := new(pvResolver)
		w := httptest.NewRecorder()
		batteryPVDecisionHandler(p)(w, httptest.NewRequest("POST", "/api/batterypvcontrol", strings.NewReader(tc.body)))
		require.Equal(t, tc.want, w.Code)
		if tc.want == 200 {
			require.Equal(t, 1, p.calls)
		} else {
			require.Zero(t, p.calls)
		}
	}
	p := &pvResolver{err: errors.New("stale revision")}
	w := httptest.NewRecorder()
	batteryPVDecisionHandler(p)(w, httptest.NewRequest("POST", "/", strings.NewReader(valid)))
	require.Equal(t, http.StatusConflict, w.Code)
	p = new(pvResolver)
	w = httptest.NewRecorder()
	gate := EnsureAuthHandler(fakeAuth{mode: auth.Enabled, password: "test"})(batteryPVDecisionHandler(p))
	gate.ServeHTTP(w, httptest.NewRequest("POST", "/", strings.NewReader(valid)))
	require.Equal(t, http.StatusUnauthorized, w.Code)
	require.Zero(t, p.calls)
}
