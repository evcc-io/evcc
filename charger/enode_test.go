package charger

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

type enodeTestServer struct {
	actionPolls         int
	maxCurrent          float64
	state               string
	chargeRate          *float64
	reportMaxCurrent    bool
	forceAlreadyStopped bool
	forceAlreadyCurrent bool
}

func (ts *enodeTestServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/oauth2/token":
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "token",
			"token_type":   "Bearer",
			"expires_in":   3600,
		})
	case r.Method == http.MethodGet && r.URL.Path == "/chargers":
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{ts.charger()},
			"pagination": map[string]any{
				"after":  nil,
				"before": nil,
			},
		})
	case r.Method == http.MethodGet && r.URL.Path == "/chargers/charger-1":
		_ = json.NewEncoder(w).Encode(ts.charger())
	case r.Method == http.MethodPost && r.URL.Path == "/chargers/charger-1/charging":
		if ts.forceAlreadyStopped {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"title":  "Action Rejected",
				"detail": "The asset is already stopped.",
			})
			return
		}
		ts.actionPolls = 0
		var req struct {
			Action string `json:"action"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.Action == "START" {
			ts.state = "PLUGGED_IN:CHARGING"
			chargeRate := 11.04
			ts.chargeRate = &chargeRate
		} else {
			ts.state = "PLUGGED_IN:STOPPED"
			ts.chargeRate = nil
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "action-charging", "state": "PENDING", "kind": req.Action})
	case r.Method == http.MethodPost && r.URL.Path == "/chargers/charger-1/max-current":
		if ts.forceAlreadyCurrent {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"title":  "Action Rejected",
				"detail": "The asset is already at the current setting.",
			})
			return
		}
		ts.actionPolls = 0
		var req struct {
			MaxCurrent float64 `json:"maxCurrent"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		ts.maxCurrent = req.MaxCurrent
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "action-current", "state": "PENDING"})
	case r.Method == http.MethodGet && (r.URL.Path == "/chargers/actions/action-charging" || r.URL.Path == "/chargers/actions/action-current"):
		ts.actionPolls++
		state := "PENDING"
		if ts.actionPolls > 1 {
			state = "CONFIRMED"
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "action", "state": state})
	default:
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprintf(w, "unexpected request %s %s", r.Method, r.URL.Path)
	}
}

func (ts *enodeTestServer) charger() map[string]any {
	return map[string]any{
		"id":          "charger-1",
		"userId":      "user-1",
		"vendor":      "TESLA",
		"isReachable": true,
		"information": map[string]any{
			"brand":        "Tesla",
			"model":        "Wall Connector",
			"serialNumber": "TWC3-1",
		},
		"capabilities": map[string]any{
			"startCharging": map[string]any{"isCapable": true, "interventionIds": []string{}},
			"stopCharging":  map[string]any{"isCapable": true, "interventionIds": []string{}},
			"setMaxCurrent": map[string]any{"isCapable": true, "interventionIds": []string{}},
			"information":   map[string]any{"isCapable": true, "interventionIds": []string{}},
			"chargeState":   map[string]any{"isCapable": true, "interventionIds": []string{}},
		},
		"chargeState": map[string]any{
			"isPluggedIn":        true,
			"isCharging":         ts.state == "PLUGGED_IN:CHARGING",
			"chargeRate":         ts.chargeRate,
			"lastUpdated":        time.Now().UTC().Format(time.RFC3339),
			"maxCurrent":         ts.maxCurrentValue(),
			"powerDeliveryState": ts.state,
		},
	}
}

func (ts *enodeTestServer) maxCurrentValue() any {
	if !ts.reportMaxCurrent {
		return nil
	}
	return ts.maxCurrent
}

func TestEnode(t *testing.T) {
	serverState := &enodeTestServer{maxCurrent: 16, state: "PLUGGED_IN:STOPPED", reportMaxCurrent: true}
	srv := httptest.NewServer(serverState)
	defer srv.Close()

	wb, err := newEnode(context.Background(), enodeEnvironment{
		apiURL:   srv.URL,
		tokenURL: srv.URL + "/oauth2/token",
	}, "client", "secret", "", "", 0, time.Second)
	if err != nil {
		t.Fatal(err)
	}

	status, err := wb.Status()
	if err != nil {
		t.Fatal(err)
	}
	if status != api.StatusB {
		t.Fatalf("unexpected status: %v", status)
	}

	enabled, err := wb.Enabled()
	if err != nil {
		t.Fatal(err)
	}
	if enabled {
		t.Fatal("expected charger to be disabled")
	}

	if err := wb.Enable(true); err != nil {
		t.Fatal(err)
	}

	enabled, err = wb.Enabled()
	if err != nil {
		t.Fatal(err)
	}
	if !enabled {
		t.Fatal("expected charger to be enabled")
	}

	status, err = wb.Status()
	if err != nil {
		t.Fatal(err)
	}
	if status != api.StatusC {
		t.Fatalf("unexpected status after start: %v", status)
	}

	power, err := wb.CurrentPower()
	if err != nil {
		t.Fatal(err)
	}
	if power != 11040 {
		t.Fatalf("unexpected power: %.0f", power)
	}

	if err := wb.MaxCurrent(24); err != nil {
		t.Fatal(err)
	}

	current, err := wb.GetMaxCurrent()
	if err != nil {
		t.Fatal(err)
	}
	if current != 24 {
		t.Fatalf("unexpected max current: %.0f", current)
	}

	if err := wb.Enable(false); err != nil {
		t.Fatal(err)
	}

	enabled, err = wb.Enabled()
	if err != nil {
		t.Fatal(err)
	}
	if enabled {
		t.Fatal("expected charger to be disabled after stop")
	}
}

func TestEnodeUnpluggedAndMaxCurrentFallback(t *testing.T) {
	serverState := &enodeTestServer{maxCurrent: 16, state: "UNPLUGGED", reportMaxCurrent: false}
	srv := httptest.NewServer(serverState)
	defer srv.Close()

	wb, err := newEnode(context.Background(), enodeEnvironment{
		apiURL:   srv.URL,
		tokenURL: srv.URL + "/oauth2/token",
	}, "client", "secret", "", "", 0, time.Second)
	if err != nil {
		t.Fatal(err)
	}

	enabled, err := wb.Enabled()
	if err != nil {
		t.Fatal(err)
	}
	if enabled {
		t.Fatal("expected unplugged charger to report disabled")
	}

	if err := wb.MaxCurrent(20); err != nil {
		t.Fatal(err)
	}

	current, err := wb.GetMaxCurrent()
	if err != nil {
		t.Fatal(err)
	}
	if current != 20 {
		t.Fatalf("unexpected fallback max current: %.0f", current)
	}
}

func TestEnodeRequiresCharger(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth2/token":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "token", "token_type": "Bearer", "expires_in": 3600})
		case "/chargers":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data":       []map[string]any{},
				"pagination": map[string]any{"after": nil, "before": nil},
			})
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	_, err := newEnode(context.Background(), enodeEnvironment{
		apiURL:   server.URL,
		tokenURL: server.URL + "/oauth2/token",
	}, "client", "secret", "", "", 0, time.Second)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestEnodeListChargersReturnsDecodedData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/chargers":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{"id": "charger-1", "userId": "user-1", "vendor": "TESLA"},
					{"id": "charger-2", "userId": "user-1", "vendor": "EASEE"},
				},
				"pagination": map[string]any{"after": nil, "before": nil},
			})
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	wb := &Enode{
		Helper:  request.NewHelper(util.NewLogger("test")),
		baseURL: server.URL,
	}

	chargers, err := wb.listChargers("")
	if err != nil {
		t.Fatal(err)
	}

	if len(chargers) != 2 {
		t.Fatalf("expected 2 chargers, got %d", len(chargers))
	}

	if chargers[0].ID != "charger-1" || chargers[1].ID != "charger-2" {
		t.Fatalf("unexpected chargers: %+v", chargers)
	}
}

func TestEnodeRequiresChargerIDOnAmbiguousSelection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth2/token":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "token", "token_type": "Bearer", "expires_in": 3600})
		case "/chargers":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{"id": "charger-1", "userId": "user-1", "vendor": "TESLA"},
					{"id": "charger-2", "userId": "user-1", "vendor": "EASEE"},
				},
				"pagination": map[string]any{"after": nil, "before": nil},
			})
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	_, err := newEnode(context.Background(), enodeEnvironment{
		apiURL:   server.URL,
		tokenURL: server.URL + "/oauth2/token",
	}, "client", "secret", "", "", 0, time.Second)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestEnodeAlreadyStoppedIsNoOp(t *testing.T) {
	serverState := &enodeTestServer{maxCurrent: 16, state: "PLUGGED_IN:STOPPED", reportMaxCurrent: true, forceAlreadyStopped: true}
	srv := httptest.NewServer(serverState)
	defer srv.Close()

	wb, err := newEnode(context.Background(), enodeEnvironment{
		apiURL:   srv.URL,
		tokenURL: srv.URL + "/oauth2/token",
	}, "client", "secret", "", "", 0, time.Second)
	if err != nil {
		t.Fatal(err)
	}

	serverState.state = "PLUGGED_IN:CHARGING"
	if err := wb.Enable(false); err != nil {
		t.Fatal(err)
	}
}

func TestEnodeAlreadyAtCurrentSettingIsNoOp(t *testing.T) {
	serverState := &enodeTestServer{maxCurrent: 16, state: "PLUGGED_IN:STOPPED", reportMaxCurrent: true, forceAlreadyCurrent: true}
	srv := httptest.NewServer(serverState)
	defer srv.Close()

	wb, err := newEnode(context.Background(), enodeEnvironment{
		apiURL:   srv.URL,
		tokenURL: srv.URL + "/oauth2/token",
	}, "client", "secret", "", "", 0, time.Second)
	if err != nil {
		t.Fatal(err)
	}

	serverState.maxCurrent = 20
	serverState.reportMaxCurrent = false
	if err := wb.MaxCurrent(20); err != nil {
		t.Fatal(err)
	}

	current, err := wb.GetMaxCurrent()
	if err != nil {
		t.Fatal(err)
	}
	if current != 20 {
		t.Fatalf("unexpected max current: %.0f", current)
	}
}

func TestResolveEnodeEnvironmentDefaultProduction(t *testing.T) {
	env := resolveEnodeEnvironment("")

	if env.apiURL != enodeProductionAPI {
		t.Fatalf("unexpected api url: %s", env.apiURL)
	}

	if env.tokenURL != enodeProductionToken {
		t.Fatalf("unexpected token url: %s", env.tokenURL)
	}
}

func TestResolveEnodeEnvironmentCustomAPIKeepsDefaultToken(t *testing.T) {
	apiURL := "https://enode-api.custom.example"
	env := resolveEnodeEnvironment(apiURL)

	if env.apiURL != apiURL {
		t.Fatalf("unexpected api url: %s", env.apiURL)
	}

	if env.tokenURL != enodeProductionToken {
		t.Fatalf("unexpected token url: %s", env.tokenURL)
	}
}

func TestResolveEnodeEnvironmentKnownSandboxAPIInfersTokenURL(t *testing.T) {
	env := resolveEnodeEnvironment(enodeSandboxAPI)

	if env.apiURL != enodeSandboxAPI {
		t.Fatalf("unexpected api url: %s", env.apiURL)
	}

	if env.tokenURL != enodeSandboxToken {
		t.Fatalf("unexpected token url: %s", env.tokenURL)
	}
}

func TestResolveEnodeEnvironmentKnownSandboxAPIWithTrailingSlash(t *testing.T) {
	env := resolveEnodeEnvironment(enodeSandboxAPI + "/")

	if env.apiURL != enodeSandboxAPI {
		t.Fatalf("unexpected api url: %s", env.apiURL)
	}

	if env.tokenURL != enodeSandboxToken {
		t.Fatalf("unexpected token url: %s", env.tokenURL)
	}
}

func TestResolveEnodeEnvironmentKnownProductionAPIInfersTokenURL(t *testing.T) {
	env := resolveEnodeEnvironment(enodeProductionAPI)

	if env.apiURL != enodeProductionAPI {
		t.Fatalf("unexpected api url: %s", env.apiURL)
	}

	if env.tokenURL != enodeProductionToken {
		t.Fatalf("unexpected token url: %s", env.tokenURL)
	}
}
