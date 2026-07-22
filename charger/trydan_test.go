package charger

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util/sponsor"
)

// withSponsor authorizes the sponsor-gated Trydan charger for the duration of the test,
// restoring the previous value afterwards so tests don't leak global state between them.
func withSponsor(t *testing.T) {
	t.Helper()
	orig := sponsor.Subject
	sponsor.Subject = "foo"
	t.Cleanup(func() { sponsor.Subject = orig })
}

func trydanTestServerWithBody(body string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, body)
	}))
}

// ChargeState maps directly to api.ChargeStatus, except firmware 2.5.0 keeps it at
// "charging" even after Paused=1, so that specific combination must fall back to StatusB.
func TestTrydanStatus(t *testing.T) {
	withSponsor(t)

	tests := []struct {
		name    string
		json    string
		want    api.ChargeStatus
		wantErr bool
	}{
		{"not connected", `{"ChargeState":0,"Paused":0}`, api.StatusA, false},
		{"connected, not charging", `{"ChargeState":1,"Paused":0}`, api.StatusB, false},
		{"charging", `{"ChargeState":2,"Paused":0}`, api.StatusC, false},
		// firmware 2.5.0 keeps ChargeState at "charging" even after Paused=1
		{"paused mid-session", `{"ChargeState":2,"Paused":1}`, api.StatusB, false},
		{"unknown state", `{"ChargeState":9,"Paused":0}`, api.StatusNone, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := trydanTestServerWithBody(tc.json)
			defer srv.Close()

			wb, err := NewTrydan(srv.URL, 0)
			if err != nil {
				t.Fatal(err)
			}

			status, err := wb.Status()
			if tc.wantErr != (err != nil) {
				t.Fatalf("got err %v, wantErr %v", err, tc.wantErr)
			}
			if status != tc.want {
				t.Errorf("got status %v, want %v", status, tc.want)
			}
		})
	}
}

// Currents() must trust a zero reading while idle, but treat it as unavailable
// (older firmware without these fields) whenever real power is flowing, since
// ChargePower>0 with all phases at zero is otherwise physically impossible.
// Voltages() is always unavailable when all phases read zero, since a
// grid-connected charger always sees mains voltage.
func TestTrydanPhaseMeasurementsUnavailable(t *testing.T) {
	withSponsor(t)

	tests := []struct {
		name            string
		json            string
		wantUnavailable bool
		wantCurrentL1   float64
		wantVoltageL1   float64
	}{
		{
			name: "idle, zero current readings are trusted",
			json: `{"ChargeState":1,"ChargePower":0,
				"IntensityMeasure_L1":0,"IntensityMeasure_L2":0,"IntensityMeasure_L3":0,
				"VoltageMeasure_L1":230.2,"VoltageMeasure_L2":229.8,"VoltageMeasure_L3":231.1}`,
			wantVoltageL1: 230.2,
		},
		{
			name: "charging but all phases read zero - unsupported firmware",
			json: `{"ChargeState":2,"ChargePower":4600,
				"IntensityMeasure_L1":0,"IntensityMeasure_L2":0,"IntensityMeasure_L3":0,
				"VoltageMeasure_L1":0,"VoltageMeasure_L2":0,"VoltageMeasure_L3":0}`,
			wantUnavailable: true,
		},
		{
			name: "charging with real per-phase readings",
			json: `{"ChargeState":2,"ChargePower":4600,
				"IntensityMeasure_L1":20.5,"IntensityMeasure_L2":0,"IntensityMeasure_L3":0,
				"VoltageMeasure_L1":228.1,"VoltageMeasure_L2":0,"VoltageMeasure_L3":0}`,
			wantCurrentL1: 20.5,
			wantVoltageL1: 228.1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := trydanTestServerWithBody(tc.json)
			defer srv.Close()

			wb, err := NewTrydan(srv.URL, 0)
			if err != nil {
				t.Fatal(err)
			}

			pc, ok := api.Cap[api.PhaseCurrents](wb)
			if !ok {
				t.Fatal("missing api.PhaseCurrents")
			}
			pv, ok := api.Cap[api.PhaseVoltages](wb)
			if !ok {
				t.Fatal("missing api.PhaseVoltages")
			}

			i1, _, _, err := pc.Currents()
			switch {
			case tc.wantUnavailable && !errors.Is(err, api.ErrNotAvailable):
				t.Errorf("Currents: got err %v, want ErrNotAvailable", err)
			case !tc.wantUnavailable && err != nil:
				t.Fatal(err)
			case !tc.wantUnavailable && i1 != tc.wantCurrentL1:
				t.Errorf("Currents: got L1=%v, want %v", i1, tc.wantCurrentL1)
			}

			v1, _, _, err := pv.Voltages()
			switch {
			case tc.wantUnavailable && !errors.Is(err, api.ErrNotAvailable):
				t.Errorf("Voltages: got err %v, want ErrNotAvailable", err)
			case !tc.wantUnavailable && err != nil:
				t.Fatal(err)
			case !tc.wantUnavailable && v1 != tc.wantVoltageL1:
				t.Errorf("Voltages: got L1=%v, want %v", v1, tc.wantVoltageL1)
			}
		})
	}
}
