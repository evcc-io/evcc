package eudataact

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// zipJSON builds an in-memory dataset zip containing a single JSON document
func zipJSON(t *testing.T, doc datasetFile) []byte {
	t.Helper()

	raw, err := json.Marshal(doc)
	require.NoError(t, err)

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("dataset.json")
	require.NoError(t, err)
	_, err = w.Write(raw)
	require.NoError(t, err)
	require.NoError(t, zw.Close())

	return buf.Bytes()
}

// testProvider returns a provider serving the given static data
func testProvider(data map[string]string) *Provider {
	return &Provider{
		statusG: func() (map[string]string, error) {
			return data, nil
		},
	}
}

func TestParseDataset(t *testing.T) {
	doc := datasetFile{
		VIN: "WVWZZZ123",
		Data: []dataPoint{
			{Key: "ffff", DataFieldName: FieldSoc, Value: "73"},
			{Key: "0001", DataFieldName: FieldSoc, Value: "80"}, // smaller key wins
			{Key: "aaaa", DataFieldName: FieldOdometer, Value: "12345"},
			{Key: "bbbb", DataFieldName: FieldRange, Value: "210"},
			{Key: "cccc", DataFieldName: FieldChargingState, Value: "charging"},
			{Key: "dddd", DataFieldName: FieldPlugState, Value: "connected"},
			{Key: "eeee", DataFieldName: FieldTargetSoc, Value: "90"},
			{Key: "0002", DataFieldName: "", Value: "ignored"}, // empty field name skipped
		},
	}

	data, err := parseDataset(zipJSON(t, doc))
	require.NoError(t, err)

	assert.Equal(t, "80", data[FieldSoc], "smallest key must win")
	assert.Equal(t, "12345", data[FieldOdometer])
	assert.Equal(t, "210", data[FieldRange])

	p := testProvider(data)

	soc, err := p.Soc()
	require.NoError(t, err)
	assert.Equal(t, 80.0, soc)

	rng, err := p.Range()
	require.NoError(t, err)
	assert.Equal(t, int64(210), rng)

	odo, err := p.Odometer()
	require.NoError(t, err)
	assert.Equal(t, 12345.0, odo)

	status, err := p.Status()
	require.NoError(t, err)
	assert.Equal(t, api.StatusC, status)

	limit, err := p.GetLimitSoc()
	require.NoError(t, err)
	assert.Equal(t, int64(90), limit)
}

func TestStatusPlugStates(t *testing.T) {
	tc := []struct {
		plug, charge string
		expected     api.ChargeStatus
	}{
		{"", "", api.StatusA},
		{"disconnected", "off", api.StatusA},
		{"connected", "readyForCharging", api.StatusB},
		{"connected", "charging", api.StatusC},
	}

	for _, tc := range tc {
		data := map[string]string{FieldPlugState: tc.plug, FieldChargingState: tc.charge}
		p := testProvider(data)

		status, err := p.Status()
		require.NoError(t, err)
		assert.Equal(t, tc.expected, status, "plug=%q charge=%q", tc.plug, tc.charge)
	}
}

func TestResolveBrand(t *testing.T) {
	for _, name := range []string{"audi", "AUDI", "Audi", "aUdI"} {
		b, ok := resolveBrand(name)
		require.True(t, ok, "brand %q must resolve", name)
		assert.Equal(t, brands["Audi"], b)
	}

	_, ok := resolveBrand("nope")
	assert.False(t, ok)
}

func TestNewestDataset(t *testing.T) {
	list := []dataset{
		{Name: "2026-05-31T08-00.zip", CreatedOn: "2026-05-31T08:00:00Z"},
		{Name: "2026-05-31T09-00.zip", CreatedOn: "2026-05-31T09:00:00Z"},
		{Name: "2026-05-31T09-15_no_content_found.zip", CreatedOn: "2026-05-31T09:15:00Z"},
	}

	assert.Equal(t, "2026-05-31T09-00.zip", newestDataset(list).Name, "newest with content, no-content skipped")
	assert.Empty(t, newestDataset([]dataset{{Name: "x_no_content_found.zip"}}).Name)
}

func TestDatasetTime(t *testing.T) {
	ref := time.Date(2026, 5, 31, 8, 0, 0, 0, time.UTC)

	tc := []struct {
		d        dataset
		expected time.Time
	}{
		{dataset{Name: "20260531080000_WVWZZZ_no_content_found.zip"}, ref},      // real portal format
		{dataset{Name: "20260531080000_WVWZZZ.zip"}, ref},                       // content file
		{dataset{CreatedOn: "2026-05-31T08:00:00Z"}, ref},                       // createdOn fallback
		{dataset{CreatedOn: "2026-05-31T08:00:00Z", Name: "no-stamp.zip"}, ref}, // name unparseable, createdOn used
		{dataset{Name: "no-timestamp.zip"}, time.Time{}},                        // nothing parseable
	}

	for _, tc := range tc {
		assert.Equal(t, tc.expected, tc.d.time(), "dataset %+v", tc.d)
	}
}

// TestResetDelay verifies the cache reset is scheduled for when the portal is
// expected to deliver the dataset following the one just read.
func TestResetDelay(t *testing.T) {
	now := time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC)

	// fresh dataset: reset one interval + latency later
	assert.Equal(t, portalInterval+portalLatency, resetDelay(now, now))

	// dataset already 5 min old: reset interval + latency after its timestamp
	assert.Equal(t, portalInterval+portalLatency-5*time.Minute, resetDelay(now.Add(-5*time.Minute), now))

	// next dataset already due: never reset sooner than the latency margin
	assert.Equal(t, portalLatency, resetDelay(now.Add(-portalInterval), now))
	assert.Equal(t, portalLatency, resetDelay(now.Add(-time.Hour), now))
}
