package server

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/globalconfig"
	"github.com/evcc-io/evcc/plugin/mqtt"
	"github.com/evcc-io/evcc/util/config"
	"github.com/evcc-io/evcc/util/templates"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// blockingMeter is an api.Meter whose CurrentPower blocks until the test ends.
type blockingMeter struct {
	done chan struct{}
}

func (m blockingMeter) CurrentPower() (float64, error) {
	<-m.done
	return 0, nil
}

// TestInstanceBoundedByContext ensures testInstance returns promptly when a
// getter blocks, bounded by the context deadline rather than the getter itself.
func TestInstanceBoundedByContext(t *testing.T) {
	done := make(chan struct{})
	defer close(done)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	testInstance(ctx, blockingMeter{done: done})
	require.Less(t, time.Since(start), 500*time.Millisecond, "testInstance must not wait for blocking getter")
}

// slowPowerMeter blocks on CurrentPower but returns TotalEnergy immediately.
type slowPowerMeter struct {
	done chan struct{}
}

func (m slowPowerMeter) CurrentPower() (float64, error) {
	<-m.done
	return 0, nil
}

func (m slowPowerMeter) TotalEnergy() (float64, error) {
	return 42, nil
}

// TestInstanceParallelProbes ensures a responsive getter returns even while a
// sibling blocks; sequential probing would starve energy behind power.
func TestInstanceParallelProbes(t *testing.T) {
	done := make(chan struct{})
	defer close(done)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	res := testInstance(ctx, slowPowerMeter{done: done})
	require.Contains(t, res, "energy", "fast getter must return despite a blocking sibling")
	require.NotContains(t, res, "power", "blocking getter must be abandoned")
}

// asleepVehicle returns api.ErrAsleep from its getters.
type asleepVehicle struct{}

func (v asleepVehicle) Soc() (float64, error) {
	return 0, api.ErrAsleep
}

func (v asleepVehicle) Range() (int64, error) {
	return 0, api.ErrAsleep
}

// TestInstanceAsleep ensures asleep is flagged as state, not as error.
func TestInstanceAsleep(t *testing.T) {
	res := testInstance(context.Background(), asleepVehicle{})
	assert.Equal(t, testResult{Value: 0.0, Asleep: true}, res["soc"])
	assert.Equal(t, testResult{Value: int64(0), Asleep: true}, res["range"])
}

func TestConfigReqUnmarshal(t *testing.T) {
	var req configReq
	require.NoError(t, json.Unmarshal([]byte(`{
		"type": "template",
		"deviceTitle": "bar",
		"template": "foo",
		"deviceProduct": "baz",
		"property": 1}
	`), &req))
	assert.Equal(t, config.Properties{
		Type:    "template",
		Title:   "bar",
		Product: "baz",
	}, req.Properties)
	assert.Equal(t, map[string]any{
		"template": "foo",
		"property": 1.0,
	}, req.Other)
}

func TestConfigReqMarshalToMap(t *testing.T) {
	props := config.Properties{
		Type:    "type",
		Title:   "title",
		Product: "product",
	}

	res, err := propsToMap(props)
	require.NoError(t, err)

	assert.Equal(t, map[string]any{
		"deviceTitle":   "title",
		"deviceProduct": "product",
	}, res)
}

type testStruct struct {
	Field1 string
	Field2 int
}

type testStructWithBool struct {
	Field1 string
	Field2 int
	Field3 bool
}

func TestMergeMaskedAny(t *testing.T) {
	tests := []struct {
		old           any
		new, expected *testStruct
	}{
		{
			old:      &testStruct{"oldValue1", 24},
			new:      &testStruct{"newValue1", 42},
			expected: &testStruct{"newValue1", 42},
		},
		{
			old:      &testStruct{"oldValue1", 24},
			new:      &testStruct{masked, 42},
			expected: &testStruct{"oldValue1", 42},
		},
	}

	for _, tc := range tests {
		require.NoError(t, mergeMaskedAny(tc.old, tc.new))
		assert.Equal(t, tc.expected, tc.new)
	}

	// Test boolean field handling
	boolTests := []struct {
		old           any
		new, expected *testStructWithBool
	}{
		{
			// Boolean false should not be overwritten by true
			old:      &testStructWithBool{"oldValue", 24, true},
			new:      &testStructWithBool{"newValue", 42, false},
			expected: &testStructWithBool{"newValue", 42, false},
		},
		{
			// Boolean true should be preserved
			old:      &testStructWithBool{"oldValue", 24, false},
			new:      &testStructWithBool{"newValue", 42, true},
			expected: &testStructWithBool{"newValue", 42, true},
		},
		{
			// Masked string should be restored, boolean should not be merged
			old:      &testStructWithBool{"oldValue", 24, true},
			new:      &testStructWithBool{masked, 42, false},
			expected: &testStructWithBool{"oldValue", 42, false},
		},
	}

	for _, tc := range boolTests {
		require.NoError(t, mergeMaskedAny(tc.old, tc.new))
		assert.Equal(t, tc.expected, tc.new)
	}
}

func TestSquashedMergeMaskedAny(t *testing.T) {
	old := globalconfig.Mqtt{
		Config: mqtt.Config{
			Broker: "host",
			User:   "user",
		},
		Topic: "test",
	}
	{
		new := old
		new.User = masked

		require.NoError(t, mergeMaskedAny(old, &new))
		assert.Equal(t, "user", new.User)
	}
	{
		new := old
		new.User = "new"

		require.NoError(t, mergeMaskedAny(old, &new))
		assert.Equal(t, "new", new.User)
	}
}

func TestMergeMaskedFiltersBehavior(t *testing.T) {
	conf := map[string]any{
		"template": "demo-meter",
		"power":    200.0,
	}

	old := map[string]any{
		"template":      "demo-meter",
		"power":         100.0,
		"outdatedField": "old-value",
	}

	result, err := mergeMasked(templates.Meter, conf, old)
	require.NoError(t, err)

	assert.Equal(t, 200.0, result["power"])
	assert.Equal(t, "demo-meter", result["template"])
	assert.NotContains(t, result, "outdatedField")
}

func TestConfigHasCriticalPlugin(t *testing.T) {
	tc := []struct {
		name string
		yaml string
		want bool
	}{
		{"script top level", "power:\n  source: script\n  cmd: echo 1", true},
		{"script case insensitive", "power:\n  Source: SCRIPT\n  cmd: echo 1", true},
		{"script nested in calc", "power:\n  source: calc\n  add:\n    - source: const\n      value: 1\n    - source: script\n      cmd: echo 1", true},
		{"script nested in sequence set", "power:\n  source: sequence\n  set:\n    - source: script\n      cmd: echo 1", true},
		{"script nested in js transformation", "power:\n  source: js\n  script: x\n  in:\n    - name: x\n      type: float\n      source: script\n      cmd: echo 1", true},
		{"js without script", "power:\n  source: js\n  script: \"x = 1\"", false},
		{"go without script", "power:\n  source: go\n  script: \"return 1\"", false},
		{"http without script", "power:\n  source: http\n  uri: http://localhost", false},
		{"plain template", "power: 100", false},
		{"script in yaml list", "- name: main\n  getmaxcurrent:\n    source: script\n    cmd: echo 1", true},
		{"benign yaml list", "- name: main\n  maxcurrent: 16", false},
	}

	for _, tc := range tc {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, configHasCriticalPlugin(configReq{Yaml: tc.yaml}))
		})
	}

	// non-yaml custom config carried in Other
	assert.True(t, configHasCriticalPlugin(configReq{
		Other: map[string]any{"power": map[string]any{"source": "script", "cmd": "echo 1"}},
	}))
	assert.False(t, configHasCriticalPlugin(configReq{
		Other: map[string]any{"template": "tesla"},
	}))
}

func TestFilterValidTemplateParams(t *testing.T) {
	conf := map[string]any{
		"template":      "generic",
		"usage":         "grid",
		"capacity":      50.0,
		"power":         100.0,
		"outdatedField": "should-be-removed",
	}

	result := filterValidTemplateParams(&templates.Template{
		Params: []templates.Param{
			{Name: "usage"},
			{Name: "power"},
			{Name: "capacity"},
		},
	}, conf)

	assert.Equal(t, "generic", result["template"], "template")
	assert.Equal(t, "grid", result["usage"], "usage")
	assert.NotContains(t, result, "outdatedField")
}
