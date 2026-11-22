package fnn

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/evcc-io/evcc/db"
	"github.com/evcc-io/evcc/util/templates"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEhiveFlexibleInputs(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	var response atomic.Value
	response.Store(ehiveResponse(false, false))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response.Load()); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	functions := []string{"unused", "w4", "w3", "s1", "s2"}
	polarity := []string{"normal", "inverted"}
	checks := 0
	for _, first := range functions {
		for _, second := range functions {
			params := map[string]any{
				"template": "ehive-one-fnn", "di1function": first, "di2function": second,
				"maxconsumptionpower": 4200, "productionnominalmax": 10000,
			}
			if first == second {
				_, err := templates.RenderInstance(templates.Hems, params)
				require.Error(t, err, "%s/%s must be rejected", first, second)
				continue
			}
			for _, p1 := range polarity {
				for _, p2 := range polarity {
					params["di1polarity"], params["di2polarity"] = p1, p2
					for bits := range 4 {
						t.Run(fmt.Sprintf("%s-%s/%s-%s/%d", first, second, p1, p2, bits), func(t *testing.T) {
							instance, err := templates.RenderInstance(templates.Hems, params)
							require.NoError(t, err)
							require.Equal(t, "fnn", instance.Type)
							instance.Other["signals"].(map[string]any)["uri"] = server.URL
							set := func(value int) {
								response.Store(ehiveResponse(value&1 != 0, value&2 != 0))
							}
							set(bits)
							fnn, err := NewFromConfig(t.Context(), instance.Other, &stubSite{})
							require.NoError(t, err)
							verify := func(value int) {
								active := map[string]bool{}
								active[first] = (value&1 != 0) != (p1 == "inverted")
								active[second] = (value&2 != 0) != (p2 == "inverted")
								if first == "w4" || second == "w4" {
									limit := 0.0
									if active["w4"] {
										limit = 4200
									}
									require.NotNil(t, fnn.MaxConsumptionPower())
									assert.Equal(t, limit, *fnn.MaxConsumptionPower())
								} else {
									assert.Nil(t, fnn.MaxConsumptionPower())
								}
								curtail := (first != "w4" && first != "unused") || (second != "w4" && second != "unused")
								if !curtail {
									assert.Nil(t, fnn.CurtailedPercent())
									return
								}
								percent := 100
								for _, rule := range []struct {
									signal string
									value  int
								}{{"w3", 0}, {"s2", 30}, {"s1", 60}} {
									if active[rule.signal] {
										percent = rule.value
										break
									}
								}
								require.NotNil(t, fnn.CurtailedPercent())
								assert.Equal(t, percent, *fnn.CurtailedPercent())
							}
							verify(bits)
							set(bits ^ 3)
							require.NoError(t, fnn.runCycle())
							verify(bits ^ 3)
							checks++
						})
					}
				}
			}
		}
	}
	assert.Equal(t, 320, checks)
}

func TestEhiveFlexibleValidation(t *testing.T) {
	for _, tc := range []struct {
		name string
		conf map[string]any
	}{
		{"missing Pmin", map[string]any{}},
		{"zero Pmin", map[string]any{"maxconsumptionpower": 0}},
		{"negative Pmin", map[string]any{"maxconsumptionpower": -1}},
		{"unknown function", map[string]any{"di1function": "w5", "maxconsumptionpower": 4200}},
		{"unknown polarity", map[string]any{"di1polarity": "other", "maxconsumptionpower": 4200}},
		{"missing generator", map[string]any{"di1function": "s1"}},
		{"zero generator", map[string]any{"di1function": "w3", "productionnominalmax": 0}},
		{"negative generator", map[string]any{"di1function": "s2", "productionnominalmax": -1}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.conf["template"] = "ehive-one-fnn"
			_, err := templates.RenderInstance(templates.Hems, tc.conf)
			require.Error(t, err)
		})
	}
	for _, value := range []any{-1, 0, "NaN", "+Inf"} {
		_, err := templates.RenderInstance(templates.Hems, map[string]any{
			"template": "ehive-one-fnn", "di1function": "w3", "productionnominalmax": value,
		})
		require.Error(t, err)
	}
}

func TestEhiveInvalidInputKeepsActiveLimit(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	var response atomic.Value
	response.Store(ehiveResponse(true, true))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewEncoder(w).Encode(response.Load()); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}))
	defer server.Close()
	for _, input := range []string{"di1", "di2"} {
		for _, polarity := range []string{"normal", "inverted"} {
			for _, signal := range []string{"w4", "w3", "s1", "s2"} {
				params := map[string]any{
					"template": "ehive-one-fnn", "di1function": "unused", "di2function": "unused",
					"maxconsumptionpower": 4200, "productionnominalmax": 10000,
					input + "function": signal, input + "polarity": polarity,
				}
				instance, err := templates.RenderInstance(templates.Hems, params)
				require.NoError(t, err)
				instance.Other["signals"].(map[string]any)["uri"] = server.URL
				for _, bad := range []any{nil, 0, 1, "true", "false", []any{}, map[string]any{}} {
					good := ehiveResponse(false, false)
					good[input] = polarity != "inverted"
					response.Store(good)
					fnn, err := NewFromConfig(t.Context(), instance.Other, &stubSite{})
					require.NoError(t, err)
					beforeDim, beforePercent := fnn.MaxConsumptionPower(), fnn.CurtailedPercent()
					invalid := ehiveResponse(false, false)
					invalid[input] = bad
					response.Store(invalid)
					require.Error(t, fnn.runCycle())
					assert.Equal(t, beforeDim, fnn.MaxConsumptionPower())
					assert.Equal(t, beforePercent, fnn.CurtailedPercent())
				}
				response.Store(map[string]any{})
				_, err = NewFromConfig(t.Context(), instance.Other, &stubSite{})
				require.Error(t, err, "missing selected input must be rejected")
			}
		}
	}
}

func ehiveResponse(di1, di2 bool) map[string]any {
	return map[string]any{"version": 1, "sequence": 1, "valid": true, "age_ms": 0, "di1": di1, "di2": di2}
}

func TestEhiveSnapshotContract(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	body := ehiveResponse(true, true)
	status, calls := http.StatusOK, 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(body)
	}))
	defer server.Close()
	instance, err := templates.RenderInstance(templates.Hems, map[string]any{
		"template": "ehive-one-fnn", "di1function": "w3", "di2function": "w4",
		"maxconsumptionpower": 4200, "productionnominalmax": 10000,
	})
	require.NoError(t, err)
	instance.Other["signals"].(map[string]any)["uri"] = server.URL
	c, err := NewFromConfig(t.Context(), instance.Other, &stubSite{})
	require.NoError(t, err)
	require.Equal(t, 1, calls)
	for _, tc := range []struct {
		key   string
		value any
	}{
		{"version", nil}, {"version", 0}, {"valid", false}, {"valid", nil}, {"valid", "true"},
		{"age_ms", nil}, {"age_ms", -1}, {"age_ms", 5001}, {"age_ms", "0"},
		{"sequence", nil}, {"sequence", 0}, {"sequence", "1"},
		{"di1", nil}, {"di2", nil},
	} {
		t.Run(fmt.Sprintf("%s/%v", tc.key, tc.value), func(t *testing.T) {
			body = ehiveResponse(false, false)
			body[tc.key] = tc.value
			before := calls
			require.Error(t, c.runCycle())
			assert.Equal(t, before+1, calls)
			assert.Equal(t, 0, *c.CurtailedPercent())
			assert.Equal(t, 4200.0, *c.MaxConsumptionPower())
			_, err := NewFromConfig(t.Context(), instance.Other, &stubSite{})
			require.Error(t, err)
		})
	}
	body = ehiveResponse(false, false)
	status = http.StatusServiceUnavailable
	require.Error(t, c.runCycle())
	assert.Equal(t, 0, *c.CurtailedPercent())
	assert.Equal(t, 4200.0, *c.MaxConsumptionPower())
	status = http.StatusOK
	require.NoError(t, c.runCycle())
	assert.Equal(t, 100, *c.CurtailedPercent())
	assert.Zero(t, *c.MaxConsumptionPower())
}
