package eudataact

import (
	"fmt"
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testProvider returns a provider serving the given static data
func testProvider(data map[string]point) *Provider {
	return &Provider{
		statusG: func() (map[string]point, error) {
			return data, nil
		},
	}
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

func TestPending(t *testing.T) {
	content := make([]dataset, 0, 11)
	for i := range 10 {
		content = append(content, dataset{
			Name:      fmt.Sprintf("20260531%02d0000_WVWZZZ.zip", i),
			CreatedOn: time.Date(2026, 5, 31, i, 0, 0, 0, time.UTC),
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
	after := content[len(content)-1].CreatedOn
	assert.Empty(t, pending(content, after))

	// a newer dataset arrives
	content = append(content, dataset{
		Name:      "20260531100000_WVWZZZ.zip",
		CreatedOn: time.Date(2026, 5, 31, 10, 0, 0, 0, time.UTC),
	})
	got = pending(content, after)
	require.Len(t, got, 1)
	assert.Equal(t, "20260531100000_WVWZZZ.zip", got[0].Name, "only the newer dataset is pending")
}

func TestMerge(t *testing.T) {
	t0 := time.Date(2026, 5, 31, 7, 0, 0, 0, time.UTC)
	t1 := time.Date(2026, 5, 31, 8, 0, 0, 0, time.UTC)

	dst := map[string]point{
		FieldSoc:      {Value: "70", Timestamp: t1},
		FieldOdometer: {Value: "100", Timestamp: t1},
	}
	src := map[string]point{
		FieldSoc:            {Value: "80", Timestamp: t0},  // newer dataset wins despite older capture
		FieldRangeSecondary: {Value: "200", Timestamp: t1}, // new field -> added
	}

	merge(dst, src, 1)

	assert.Equal(t, "80", dst[FieldSoc].Value, "newer dataset wins, even with an older timestampUtc")
	assert.Equal(t, "100", dst[FieldOdometer].Value, "field absent from src is retained")
	assert.Equal(t, "200", dst[FieldRangeSecondary].Value, "new field added")
}

// TestMergeDeliveryOrder guards the case where the portal stamps fresh values
// with older capture times: precedence must follow delivery order, not the timestamp.
func TestMergeDeliveryOrder(t *testing.T) {
	parse := func(local string) time.Time {
		ts, err := time.ParseInLocation("2006-01-02 15:04:05", local, time.Local)
		require.NoError(t, err)
		return ts
	}

	data := map[string]point{}
	// datasets in delivery order; capture timestamps go backwards as SoC rises
	for i, d := range []struct{ value, capture string }{
		{"60", "2026-06-13 14:03:37"}, // delivered 14:10
		{"65", "2026-06-13 12:44:31"}, // delivered 14:55, older capture
		{"66", "2026-06-13 13:03:16"}, // delivered 15:11
		{"75", "2026-06-13 13:52:11"}, // delivered 16:54
	} {
		merge(data, map[string]point{FieldSoc: {Value: d.value, Timestamp: parse(d.capture)}}, uint64(i+1))
	}

	assert.Equal(t, "75", data[FieldSoc].Value, "the newest delivered SoC wins")
}

// TestSocFreshestField reproduces issue #30877: a higher-priority SoC field that
// stops being delivered must not shadow a lower-priority field that keeps rising.
func TestSocFreshestField(t *testing.T) {
	data := map[string]point{}
	var seq uint64
	deliver := func(fields map[string]point) {
		seq++
		merge(data, fields, seq)
	}

	// first datasets carry both SoC fields at 57, the high-priority field winning
	deliver(map[string]point{
		FieldBatteryStateReportSoc: {Value: "57"},
		FieldHvBatteryLevel:        {Value: "57.0"},
	})
	deliver(map[string]point{
		FieldBatteryStateReportSoc: {Value: "57"},
		FieldHvBatteryLevel:        {Value: "57.0"},
	})

	// later datasets only refresh the fallback field as the car charges
	for _, v := range []string{"58.0", "59.0", "61.0"} {
		deliver(map[string]point{FieldHvBatteryLevel: {Value: v}})
	}

	soc, err := testProvider(data).Soc()
	require.NoError(t, err)
	assert.Equal(t, 61.0, soc, "the still-updating fallback wins over the stale high-priority field")
}

// TestPoints guards that a data point with a generic field name ("value") is
// indexed by its unique key while the name stays indexed (and thus logged).
func TestPoints(t *testing.T) {
	data := points([]dataPoint{
		{Key: KeyRangeID3, DataFieldName: "value", Value: "317"},
		{DataFieldName: FieldOdometer, Value: "22164"},
	})

	assert.Equal(t, "317", data[KeyRangeID3].Value, "ID.3 range indexed by key")
	assert.Equal(t, "317", data["value"].Value, "field name remains indexed")
	assert.Equal(t, "22164", data[FieldOdometer].Value, "named field indexed by name")
}
