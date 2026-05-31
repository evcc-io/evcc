package eudataact

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
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
func testProvider(data map[string]point) *Provider {
	return &Provider{
		statusG: func() (map[string]point, error) {
			return data, nil
		},
	}
}

func TestParseDataset(t *testing.T) {
	doc := datasetFile{
		VIN: "WVWZZZ123",
		Data: []dataPoint{
			{DataFieldName: FieldSoc, Value: "73", TimestampUtc: "2026-05-31T07:00:00Z"},
			{DataFieldName: FieldSoc, Value: "80", TimestampUtc: "2026-05-31T08:00:00Z"}, // newest timestamp wins
			{DataFieldName: FieldOdometer, Value: "12345", TimestampUtc: "2026-05-31T08:00:00Z"},
			{DataFieldName: FieldRange, Value: "210", TimestampUtc: "2026-05-31T08:00:00Z"},
			{DataFieldName: FieldChargingState, Value: "charging", TimestampUtc: "2026-05-31T08:00:00Z"},
			{DataFieldName: FieldPlugState, Value: "connected", TimestampUtc: "2026-05-31T08:00:00Z"},
			{DataFieldName: FieldTargetSoc, Value: "90", TimestampUtc: "2026-05-31T08:00:00Z"},
			{DataFieldName: "", Value: "ignored"}, // empty field name skipped
		},
	}

	data, err := parseDataset(zipJSON(t, doc))
	require.NoError(t, err)

	assert.Equal(t, "80", data[FieldSoc].Value, "newest timestamp must win")
	assert.Equal(t, "12345", data[FieldOdometer].Value)
	assert.Equal(t, "210", data[FieldRange].Value)

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
		data := map[string]point{
			FieldPlugState:     {Value: tc.plug},
			FieldChargingState: {Value: tc.charge},
		}
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

func TestContentDatasets(t *testing.T) {
	list := []dataset{
		{Name: "20260531090000_WVWZZZ.zip"},
		{Name: "20260531080000_WVWZZZ.zip"},
		{Name: "20260531091500_WVWZZZ_no_content_found.zip"},
	}

	content, err := contentDatasets(list)
	require.NoError(t, err)
	require.Len(t, content, 2, "no-content placeholder dropped")
	assert.Equal(t, "20260531080000_WVWZZZ.zip", content[0].Name, "oldest first")
	assert.Equal(t, "20260531090000_WVWZZZ.zip", content[1].Name, "newest last")
	assert.Equal(t, time.Date(2026, 5, 31, 8, 0, 0, 0, time.UTC), content[0].Timestamp, "timestamp parsed")

	// no-content placeholders are skipped without parsing
	empty, err := contentDatasets([]dataset{{Name: "x_no_content_found.zip"}})
	require.NoError(t, err)
	assert.Empty(t, empty)

	// a content dataset with an unparseable timestamp is an error
	_, err = contentDatasets([]dataset{{Name: "no-timestamp.zip"}})
	require.Error(t, err)
}

func TestPending(t *testing.T) {
	content := make([]dataset, 0, 11)
	for i := range 10 {
		content = append(content, dataset{
			Name:      fmt.Sprintf("20260531%02d0000_WVWZZZ.zip", i),
			Timestamp: time.Date(2026, 5, 31, i, 0, 0, 0, time.UTC),
		}) // hour i, oldest first
	}

	// first poll (zero high-water): only the latest maxBackfill
	got := pending(content, time.Time{})
	require.Len(t, got, maxBackfill)
	assert.Equal(t, content[len(content)-maxBackfill].Name, got[0].Name, "oldest within the backfill window")
	assert.Equal(t, content[len(content)-1].Name, got[len(got)-1].Name, "newest")

	// fewer datasets than the backfill cap: all returned
	assert.Len(t, pending(content[:3], time.Time{}), 3)

	// high-water at the newest merged dataset: nothing new to download
	after := content[len(content)-1].Timestamp
	assert.Empty(t, pending(content, after))

	// a newer dataset arrives
	content = append(content, dataset{
		Name:      "20260531100000_WVWZZZ.zip",
		Timestamp: time.Date(2026, 5, 31, 10, 0, 0, 0, time.UTC),
	})
	got = pending(content, after)
	require.Len(t, got, 1)
	assert.Equal(t, "20260531100000_WVWZZZ.zip", got[0].Name, "only the newer dataset is pending")
}

func TestMerge(t *testing.T) {
	t0 := time.Date(2026, 5, 31, 7, 0, 0, 0, time.UTC)
	t1 := time.Date(2026, 5, 31, 8, 0, 0, 0, time.UTC)

	dst := map[string]point{
		FieldSoc:      {Value: "70", Timestamp: t0},
		FieldOdometer: {Value: "100", Timestamp: t1},
	}
	src := map[string]point{
		FieldSoc:      {Value: "80", Timestamp: t1},  // newer -> wins
		FieldOdometer: {Value: "90", Timestamp: t0},  // older -> ignored
		FieldRange:    {Value: "200", Timestamp: t1}, // new field -> added
	}

	merge(dst, src)

	assert.Equal(t, "80", dst[FieldSoc].Value, "newer datapoint wins")
	assert.Equal(t, "100", dst[FieldOdometer].Value, "older datapoint ignored")
	assert.Equal(t, "200", dst[FieldRange].Value, "new field added")
}

func TestDatasetTime(t *testing.T) {
	ref := time.Date(2026, 5, 31, 8, 0, 0, 0, time.UTC)

	tc := []struct {
		d        dataset
		expected time.Time
		err      bool
	}{
		{dataset{Name: "20260531080000_WVWZZZ_no_content_found.zip"}, ref, false},      // real portal format
		{dataset{Name: "20260531080000_WVWZZZ.zip"}, ref, false},                       // content file
		{dataset{CreatedOn: "2026-05-31T08:00:00Z"}, ref, false},                       // createdOn fallback
		{dataset{CreatedOn: "2026-05-31T08:00:00Z", Name: "no-stamp.zip"}, ref, false}, // name unparseable, createdOn used
		{dataset{Name: "no-timestamp.zip"}, time.Time{}, true},                         // nothing parseable
	}

	for _, tc := range tc {
		got, err := tc.d.time()
		if tc.err {
			assert.Error(t, err, "dataset %+v", tc.d)
			continue
		}
		require.NoError(t, err, "dataset %+v", tc.d)
		assert.Equal(t, tc.expected, got, "dataset %+v", tc.d)
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
