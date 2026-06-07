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
		FieldSoc:      {Value: "70", Timestamp: t0},
		FieldOdometer: {Value: "100", Timestamp: t1},
	}
	src := map[string]point{
		FieldSoc:            {Value: "80", Timestamp: t1},  // newer -> wins
		FieldOdometer:       {Value: "90", Timestamp: t0},  // older -> ignored
		FieldRangeSecondary: {Value: "200", Timestamp: t1}, // new field -> added
	}

	merge(dst, src)

	assert.Equal(t, "80", dst[FieldSoc].Value, "newer datapoint wins")
	assert.Equal(t, "100", dst[FieldOdometer].Value, "older datapoint ignored")
	assert.Equal(t, "200", dst[FieldRangeSecondary].Value, "new field added")
}
