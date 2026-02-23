package tariff

import (
	"encoding/json"
	"fmt" // used in TestHATariffPriceModeError
	"testing"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockHASource implements haStateSource for testing.
// attributes maps entity → attribute name → value.
// priceState is the raw state string returned for price-mode entities.
type mockHASource struct {
	priceState string
	attributes map[string]map[string]any
	jsonErr    error
}

func (m *mockHASource) GetJSON(uri string, result any) error {
	if m.jsonErr != nil {
		return m.jsonErr
	}
	// extract entity from URI: .../api/states/<entity>
	parts := splitURI(uri)
	entity := parts[len(parts)-1]

	attrs, ok := m.attributes[entity]
	if !ok {
		attrs = map[string]any{}
	}

	b, err := json.Marshal(map[string]any{"state": m.priceState, "attributes": attrs})
	if err != nil {
		return err
	}
	return json.Unmarshal(b, result)
}

func splitURI(uri string) []string {
	parts := []string{}
	cur := ""
	for _, c := range uri {
		if c == '/' {
			if cur != "" {
				parts = append(parts, cur)
			}
			cur = ""
		} else {
			cur += string(c)
		}
	}
	if cur != "" {
		parts = append(parts, cur)
	}
	return parts
}

func newTestTariff(t *testing.T, source haStateSource, entities []string, attrs []string, format string) api.Tariff {
	t.Helper()
	emb := &embed{}
	require.NoError(t, emb.init())
	tar, err := newHATariff(emb, util.NewLogger("test"), source,
		"http://ha.local:8123", entities, attrs, format,
		0, time.Hour, time.Minute)
	require.NoError(t, err)
	return tar
}

// --- price mode ---

func TestHATariffPriceMode(t *testing.T) {
	source := &mockHASource{priceState: "0.25"}
	tar := newTestTariff(t, source, []string{"sensor.price"}, nil, "")

	assert.Equal(t, api.TariffTypePriceDynamic, tar.Type())

	rates, err := tar.Rates()
	require.NoError(t, err)
	assert.Equal(t, 48*4, len(rates))
	assert.Equal(t, 0.25, rates[0].Value)
	assert.Equal(t, rates[0].End, rates[1].Start)
}

func TestHATariffPriceModeError(t *testing.T) {
	source := &mockHASource{jsonErr: fmt.Errorf("connection refused")}
	tar := newTestTariff(t, source, []string{"sensor.price"}, nil, "")
	_, err := tar.Rates()
	require.Error(t, err)
}

func TestHATariffPriceModeUnavailable(t *testing.T) {
	source := &mockHASource{priceState: "unavailable"}
	tar := newTestTariff(t, source, []string{"sensor.price"}, nil, "")
	_, err := tar.Rates()
	require.Error(t, err)
}

func TestHATariffPriceModeMultiEntityRejected(t *testing.T) {
	emb := &embed{}
	require.NoError(t, emb.init())
	_, err := newHATariff(emb, util.NewLogger("test"), &mockHASource{},
		"http://ha.local:8123", []string{"sensor.a", "sensor.b"}, nil, "",
		0, time.Hour, time.Minute)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exactly one entity")
}

// --- native forecast (Nordpool) ---

func TestHATariffNativeForecast(t *testing.T) {
	now := time.Now().Truncate(time.Hour)
	rates := api.Rates{
		{Start: now, End: now.Add(time.Hour), Value: 0.25},
		{Start: now.Add(time.Hour), End: now.Add(2 * time.Hour), Value: 0.20},
	}
	raw, _ := json.Marshal(rates)

	source := &mockHASource{
		attributes: map[string]map[string]any{
			"sensor.nordpool": {"raw_today": json.RawMessage(raw)},
		},
	}
	tar := newTestTariff(t, source, []string{"sensor.nordpool"}, []string{"raw_today"}, "nordpool")

	assert.Equal(t, api.TariffTypePriceForecast, tar.Type())
	result, err := tar.Rates()
	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, 0.25, result[0].Value)
}

func TestHATariffMultiAttributeMerge(t *testing.T) {
	now := time.Now().Truncate(time.Hour)
	today := api.Rates{{Start: now, End: now.Add(time.Hour), Value: 0.25}}
	tomorrow := api.Rates{{Start: now.Add(24 * time.Hour), End: now.Add(25 * time.Hour), Value: 0.20}}
	rawToday, _ := json.Marshal(today)
	rawTomorrow, _ := json.Marshal(tomorrow)

	source := &mockHASource{
		attributes: map[string]map[string]any{
			"sensor.nordpool": {
				"raw_today":    json.RawMessage(rawToday),
				"raw_tomorrow": json.RawMessage(rawTomorrow),
			},
		},
	}
	tar := newTestTariff(t, source, []string{"sensor.nordpool"}, []string{"raw_today", "raw_tomorrow"}, "nordpool")

	result, err := tar.Rates()
	require.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestHATariffMissingAttributeSkipped(t *testing.T) {
	now := time.Now().Truncate(time.Hour)
	today := api.Rates{{Start: now, End: now.Add(time.Hour), Value: 0.25}}
	raw, _ := json.Marshal(today)

	source := &mockHASource{
		attributes: map[string]map[string]any{
			"sensor.nordpool": {"raw_today": json.RawMessage(raw)},
			// raw_tomorrow absent — silently skipped
		},
	}
	tar := newTestTariff(t, source, []string{"sensor.nordpool"}, []string{"raw_today", "raw_tomorrow"}, "nordpool")

	result, err := tar.Rates()
	require.NoError(t, err)
	assert.Len(t, result, 1)
}

// --- multi-entity (solar day-ahead) ---

func TestHATariffMultiEntityMerge(t *testing.T) {
	makeRaw := func(start time.Time, w float64) json.RawMessage {
		key := start.Format(time.RFC3339)
		b, _ := json.Marshal(map[string]float64{key: w})
		return b
	}

	now := time.Now().Truncate(time.Hour)
	d1 := now.Add(24 * time.Hour)

	source := &mockHASource{
		attributes: map[string]map[string]any{
			"sensor.solar_today": {"watts": makeRaw(now, 500)},
			"sensor.solar_d1":    {"watts": makeRaw(d1, 450)},
		},
	}
	tar := newTestTariff(t, source,
		[]string{"sensor.solar_today", "sensor.solar_d1"},
		[]string{"watts"}, "watts")

	result, err := tar.Rates()
	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, float64(500), result[0].Value)
	assert.Equal(t, float64(450), result[1].Value)
}

// --- Zonneplan ---

func TestParseZonneplan(t *testing.T) {
	raw := json.RawMessage(`[
		{"datetime":"2026-02-23T10:00:00.000000Z","electricity_price":1830020},
		{"datetime":"2026-02-23T11:00:00.000000Z","electricity_price":1413750},
		{"datetime":"2026-02-23T12:00:00.000000Z","electricity_price":1364957}
	]`)

	rates, err := parseZonneplan(raw)
	require.NoError(t, err)
	require.Len(t, rates, 3)

	assert.InDelta(t, 0.183002, rates[0].Value, 1e-6)
	assert.Equal(t, rates[0].End, rates[1].Start)
	assert.Equal(t, rates[1].End, rates[2].Start)
	assert.Equal(t, rates[2].Start.Add(time.Hour), rates[2].End)
}

func TestHATariffZonneplan(t *testing.T) {
	raw := json.RawMessage(`[
		{"datetime":"2026-02-23T10:00:00.000000Z","electricity_price":1830020},
		{"datetime":"2026-02-23T11:00:00.000000Z","electricity_price":1413750}
	]`)

	source := &mockHASource{
		attributes: map[string]map[string]any{
			"sensor.zonneplan_current_electricity_tariff": {"forecast": json.RawMessage(raw)},
		},
	}
	tar := newTestTariff(t, source,
		[]string{"sensor.zonneplan_current_electricity_tariff"},
		[]string{"forecast"}, "zonneplan")

	result, err := tar.Rates()
	require.NoError(t, err)
	require.Len(t, result, 2)
	assert.InDelta(t, 0.183002, result[0].Value, 1e-6)
}

// --- ENTSO-E ---

func TestParseEntsoe(t *testing.T) {
	raw := json.RawMessage(`[
		{"time":"2026-02-23 10:00:00+01:00","price":0.23093},
		{"time":"2026-02-23 11:00:00+01:00","price":0.21048},
		{"time":"2026-02-23 12:00:00+01:00","price":0.16886}
	]`)

	rates, err := parseEntsoe(raw)
	require.NoError(t, err)
	require.Len(t, rates, 3)

	assert.Equal(t, 0.23093, rates[0].Value)
	assert.Equal(t, rates[0].End, rates[1].Start)
	assert.Equal(t, rates[2].Start.Add(time.Hour), rates[2].End)
}

// --- Forecast.Solar / Open-Meteo ---

func TestParseWatts(t *testing.T) {
	raw := json.RawMessage(`{
		"2026-02-23T08:45:00+01:00": 220,
		"2026-02-23T09:00:00+01:00": 358,
		"2026-02-23T09:15:00+01:00": 492,
		"2026-02-23T08:30:00+01:00": 114
	}`)

	rates, err := parseWatts(raw)
	require.NoError(t, err)
	require.Len(t, rates, 4)

	assert.True(t, rates[0].Start.Before(rates[1].Start)) // sorted
	assert.Equal(t, float64(114), rates[0].Value)         // 08:30 first
	assert.Equal(t, rates[0].End, rates[1].Start)         // contiguous
	assert.Equal(t, rates[3].Start.Add(SlotDuration), rates[3].End)
}

func TestHATariffWatts(t *testing.T) {
	raw := json.RawMessage(`{
		"2026-02-23T13:00:00+01:00": 789,
		"2026-02-23T13:15:00+01:00": 1005,
		"2026-02-23T13:30:00+01:00": 1046
	}`)

	source := &mockHASource{
		attributes: map[string]map[string]any{
			"sensor.energy_production_today": {"watts": json.RawMessage(raw)},
		},
	}
	tar := newTestTariff(t, source,
		[]string{"sensor.energy_production_today"},
		[]string{"watts"}, "watts")

	result, err := tar.Rates()
	require.NoError(t, err)
	require.Len(t, result, 3)
	assert.Equal(t, float64(789), result[0].Value)
}

// --- Solcast ---

func TestParseSolcast(t *testing.T) {
	raw := json.RawMessage(`[
		{"period_start":"2026-02-23T08:00:00+00:00","pv_estimate":0.5,"pv_estimate10":0.3,"pv_estimate90":0.7},
		{"period_start":"2026-02-23T08:30:00+00:00","pv_estimate":1.2,"pv_estimate10":0.9,"pv_estimate90":1.5},
		{"period_start":"2026-02-23T09:00:00+00:00","pv_estimate":2.0,"pv_estimate10":1.5,"pv_estimate90":2.5}
	]`)

	rates, err := parseSolcast(raw)
	require.NoError(t, err)
	require.Len(t, rates, 3)

	assert.InDelta(t, 500.0, rates[0].Value, 1e-6) // 0.5 kW → 500 W
	assert.InDelta(t, 1200.0, rates[1].Value, 1e-6)
	assert.Equal(t, 30*time.Minute, rates[0].End.Sub(rates[0].Start))
	assert.Equal(t, rates[0].End, rates[1].Start)
}

func TestHATariffSolcast(t *testing.T) {
	raw := json.RawMessage(`[
		{"period_start":"2026-02-23T08:00:00+00:00","pv_estimate":1.5},
		{"period_start":"2026-02-23T08:30:00+00:00","pv_estimate":2.3}
	]`)

	source := &mockHASource{
		attributes: map[string]map[string]any{
			"sensor.solcast_pv_forecast": {"detailedForecast": json.RawMessage(raw)},
		},
	}
	tar := newTestTariff(t, source,
		[]string{"sensor.solcast_pv_forecast"},
		[]string{"detailedForecast"}, "solcast")

	result, err := tar.Rates()
	require.NoError(t, err)
	require.Len(t, result, 2)
	assert.InDelta(t, 1500.0, result[0].Value, 1e-6)
}

// --- forecast fetch error ---

func TestHATariffForecastHAFetchError(t *testing.T) {
	// wrap as permanent so the backoff loop exits immediately
	source := &mockHASource{jsonErr: backoff.Permanent(fmt.Errorf("connection refused"))}
	emb := &embed{}
	require.NoError(t, emb.init())
	_, err := newHATariff(emb, util.NewLogger("test"), source,
		"http://ha.local:8123", []string{"sensor.nordpool"}, []string{"raw_today"}, "nordpool",
		0, time.Hour, time.Minute)
	require.Error(t, err)
}

// --- unknown format ---

func TestHATariffUnknownFormat(t *testing.T) {
	source := &mockHASource{
		attributes: map[string]map[string]any{
			"sensor.foo": {"data": json.RawMessage(`[]`)},
		},
	}
	emb := &embed{}
	require.NoError(t, emb.init())
	_, err := newHATariff(emb, util.NewLogger("test"), source,
		"http://ha.local:8123", []string{"sensor.foo"}, []string{"data"}, "unknown",
		0, time.Hour, time.Minute)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported")
}

// --- parser negative paths ---

func TestParseZonneplan_InvalidPayload(t *testing.T) {
	_, err := parseZonneplan(json.RawMessage(`not json`))
	require.Error(t, err)
}

func TestParseZonneplan_InvalidTimestamp(t *testing.T) {
	_, err := parseZonneplan(json.RawMessage(`[{"datetime":"not-a-time","electricity_price":1000}]`))
	require.Error(t, err)
}

func TestParseEntsoe_InvalidPayload(t *testing.T) {
	_, err := parseEntsoe(json.RawMessage(`not json`))
	require.Error(t, err)
}

func TestParseEntsoe_InvalidTimestamp(t *testing.T) {
	_, err := parseEntsoe(json.RawMessage(`[{"time":"not-a-time","price":0.23}]`))
	require.Error(t, err)
}

func TestParseWatts_InvalidPayload(t *testing.T) {
	_, err := parseWatts(json.RawMessage(`not json`))
	require.Error(t, err)
}

func TestParseWatts_InvalidTimestamp(t *testing.T) {
	_, err := parseWatts(json.RawMessage(`{"not-a-timestamp": 123}`))
	require.Error(t, err)
}

func TestParseSolcast_InvalidPayload(t *testing.T) {
	_, err := parseSolcast(json.RawMessage(`not json`))
	require.Error(t, err)
}

func TestParseSolcast_InvalidTimestamp(t *testing.T) {
	_, err := parseSolcast(json.RawMessage(`[{"period_start":"not-a-time","pv_estimate":1.5}]`))
	require.Error(t, err)
}

// --- config validation ---

func TestHATariffMissingURI(t *testing.T) {
	_, err := NewHATariffFromConfig(map[string]any{"entities": []string{"sensor.price"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "uri")
}

func TestHATariffMissingEntities(t *testing.T) {
	_, err := NewHATariffFromConfig(map[string]any{"uri": "http://ha.local:8123"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "entities")
}

func TestHATariffEntitiesListDecoding(t *testing.T) {
	emb := &embed{}
	require.NoError(t, emb.init())

	now := time.Now().Truncate(time.Hour)
	raw, _ := json.Marshal(map[string]float64{now.Format(time.RFC3339): 500})

	source := &mockHASource{
		attributes: map[string]map[string]any{
			"sensor.solar_today": {"watts": json.RawMessage(raw)},
			"sensor.solar_d1":    {"watts": json.RawMessage(raw)},
		},
	}

	tar, err := newHATariff(emb, util.NewLogger("test"), source,
		"http://ha.local:8123",
		[]string{"sensor.solar_today", "sensor.solar_d1"},
		[]string{"watts"}, "watts",
		0, time.Hour, time.Minute)
	require.NoError(t, err)
	assert.Equal(t, []string{"sensor.solar_today", "sensor.solar_d1"}, tar.(*HATariff).entities)
}
