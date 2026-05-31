package eudataact

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"testing"

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

	p := &Provider{statusG: func() (map[string]string, error) { return data, nil }}

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
		p := &Provider{statusG: func() (map[string]string, error) { return data, nil }}

		status, err := p.Status()
		require.NoError(t, err)
		assert.Equal(t, tc.expected, status, "plug=%q charge=%q", tc.plug, tc.charge)
	}
}

func TestNewestDataset(t *testing.T) {
	list := []dataset{
		{Name: "2026-05-31T08-00.zip", CreatedOn: "2026-05-31T08:00:00Z"},
		{Name: "2026-05-31T09-00.zip", CreatedOn: "2026-05-31T09:00:00Z"},
		{Name: "2026-05-31T09-15_no_content_found.zip", CreatedOn: "2026-05-31T09:15:00Z"},
	}

	assert.Equal(t, "2026-05-31T09-00.zip", newestDataset(list), "newest with content, no-content skipped")
	assert.Empty(t, newestDataset([]dataset{{Name: "x_no_content_found.zip"}}))
}
