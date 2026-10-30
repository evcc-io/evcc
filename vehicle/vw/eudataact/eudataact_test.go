package eudataact

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testProvider returns a provider serving the given static data
func testProvider(data []point) *Provider {
	return &Provider{
		statusG: func() ([]point, error) {
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
		data := []point{
			{Name: FieldPlugState, Value: tc.plug},
			{Name: FieldChargingState, Value: tc.charge},
		}
		p := testProvider(data)

		status, err := p.Status()
		require.NoError(t, err)
		assert.Equal(t, tc.expected, status, "plug=%q charge=%q", tc.plug, tc.charge)
	}
}

func TestStatusConservationCharging(t *testing.T) {
	tc := []struct {
		field    string
		value    string
		expected api.ChargeStatus
	}{
		{FieldChargingState, "conservationCharging", api.StatusC},
		{FieldCurrentChargeState, "CHARGE_STATE_CONSERVATION_CHARGING", api.StatusC},
	}

	for _, tc := range tc {
		data := []point{{Name: tc.field, Value: tc.value}}
		p := testProvider(data)

		status, err := p.Status()
		require.NoError(t, err)
		assert.Equal(t, tc.expected, status, "field=%q value=%q", tc.field, tc.value)
	}
}

func TestStatusChargingScenario(t *testing.T) {
	tc := []struct {
		scenario string
		expected api.ChargeStatus
	}{
		{"CHARGING_SCENARIO_IMMEDIATELY_CHARGING_FINISHED", api.StatusB},
		{"CHARGING_SCENARIO_OPTIMISED_CHARGING_FINISHED", api.StatusB},
		{"CHARGING_SCENARIO_CHARGING_TO_DEPARTURE_TIME_FINISHED", api.StatusB},
		{"CHARGING_SCENARIO_IMMEDIATELY_CHARGING_ACTIVE", api.StatusC},
		{"CHARGING_SCENARIO_CHARGING_TO_DEPARTURE_TIME_ACTIVE", api.StatusC},
		{"CHARGING_SCENARIO_OFF", api.StatusA},
		// case-insensitivity: mixed-case value should behave like its uppercased equivalent
		{"Charging_Scenario_Off", api.StatusA},
	}

	for _, tc := range tc {
		data := []point{{Name: FieldChargingScenario, Value: tc.scenario}}
		p := testProvider(data)

		status, err := p.Status()
		require.NoError(t, err)
		assert.Equal(t, tc.expected, status, "scenario=%q", tc.scenario)
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

	// IsBrand drives the cloud server's brand -> drivesomethinggreater routing
	for _, name := range []string{"audi", "Volkswagen", "seat", "CUPRA", "skoda"} {
		assert.True(t, IsBrand(name), "%q is a VW group brand", name)
	}
	assert.False(t, IsBrand("tesla"))
}

func TestPending(t *testing.T) {
	content := make([]dataset, 0, 11)
	for i := range 10 {
		content = append(content, dataset{
			Name:      fmt.Sprintf("20260531%02d0000_WVWZZZ.zip", i),
			CreatedOn: time.Date(2026, 5, 31, i, 0, 0, 0, time.UTC),
		}) // hour i, oldest first
	}

	// first poll (zero high-water): only the latest maxDownloads
	got := pending(content, time.Time{})
	require.Len(t, got, maxDownloads)
	assert.Equal(t, content[len(content)-maxDownloads].Name, got[0].Name, "oldest within the backfill window")
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

	// catching up after a long pause: only the latest maxDownloads of the newer datasets
	got = pending(content, content[0].CreatedOn)
	require.Len(t, got, maxDownloads)
	assert.Equal(t, content[len(content)-maxDownloads].Name, got[0].Name, "oldest within the catch-up window")
	assert.Equal(t, content[len(content)-1].Name, got[len(got)-1].Name, "newest")

	// first poll without delivery times: datasets are not lost
	assert.Len(t, pending([]dataset{{Name: "a.zip"}, {Name: "b.zip"}}, time.Time{}), 2)
}

func TestMerge(t *testing.T) {
	t0 := time.Date(2026, 5, 31, 7, 0, 0, 0, time.UTC)
	t1 := time.Date(2026, 5, 31, 8, 0, 0, 0, time.UTC)

	dst := []point{
		{Name: FieldSoc, Value: "70", Timestamp: t1},
		{Name: FieldOdometer, Value: "100", Timestamp: t1},
	}
	src := []point{
		{Name: FieldSoc, Value: "80", Timestamp: t0},             // newer dataset wins despite older capture
		{Name: FieldRangeSecondary, Value: "200", Timestamp: t1}, // new field -> added
	}

	dst = merge(dst, src, 1)

	assert.Equal(t, "80", find(dst, FieldSoc).Value, "newer dataset wins, even with an older timestampUtc")
	assert.Equal(t, "100", find(dst, FieldOdometer).Value, "field absent from src is retained")
	assert.Equal(t, "200", find(dst, FieldRangeSecondary).Value, "new field added")
}

// TestMergeDeliveryOrder guards the case where the portal stamps fresh values
// with older capture times: precedence must follow delivery order, not the timestamp.
func TestMergeDeliveryOrder(t *testing.T) {
	parse := func(local string) time.Time {
		ts, err := time.ParseInLocation("2006-01-02 15:04:05", local, time.Local)
		require.NoError(t, err)
		return ts
	}

	var data []point
	// datasets in delivery order; capture timestamps go backwards as SoC rises
	for i, d := range []struct{ value, capture string }{
		{"60", "2026-06-13 14:03:37"}, // delivered 14:10
		{"65", "2026-06-13 12:44:31"}, // delivered 14:55, older capture
		{"66", "2026-06-13 13:03:16"}, // delivered 15:11
		{"75", "2026-06-13 13:52:11"}, // delivered 16:54
	} {
		data = merge(data, []point{{Name: FieldSoc, Value: d.value, Timestamp: parse(d.capture)}}, uint64(i+1))
	}

	assert.Equal(t, "75", find(data, FieldSoc).Value, "the newest delivered SoC wins")
}

// TestSocFreshestField reproduces issue #30877: a higher-priority SoC field that
// stops being delivered must not shadow a lower-priority field that keeps rising.
func TestSocFreshestField(t *testing.T) {
	var data []point
	var seq uint64
	deliver := func(fields []point) {
		seq++
		data = merge(data, fields, seq)
	}

	// first datasets carry both SoC fields at 57, the high-priority field winning
	deliver([]point{
		{Name: FieldHvBatteryLevelValue, Value: "57.0"},
		{Name: FieldSoc, Value: "57"},
	})
	deliver([]point{
		{Name: FieldHvBatteryLevelValue, Value: "57.0"},
		{Name: FieldSoc, Value: "57"},
	})

	// later datasets only refresh the lower-priority field as the car charges
	for _, v := range []string{"58", "59", "61"} {
		deliver([]point{{Name: FieldSoc, Value: v}})
	}

	soc, err := testProvider(data).Soc()
	require.NoError(t, err)
	assert.Equal(t, 61.0, soc, "the still-updating field wins over the stale higher-priority field")
}

func TestSocBatteryStateReportField(t *testing.T) {
	data := []point{{Key: KeyBatteryStateReportSoc, Name: "battery_state_report.soc", Value: "36"}}

	soc, err := testProvider(data).Soc()
	require.NoError(t, err)
	assert.Equal(t, 36.0, soc)
}

func TestOdometerMileageKey(t *testing.T) {
	data := []point{
		{Key: "30cc36fd-71ca-3c09-9296-e94ebd47bd2b", Name: FieldOdometerValue, Value: "1875"},
		{Key: KeyMileageValue, Name: FieldOdometerValue, Value: "1879"},
	}

	odo, err := testProvider(data).Odometer()
	require.NoError(t, err)
	assert.Equal(t, 1879.0, odo)
}

func TestSocEnyaqField(t *testing.T) {
	data := []point{{Key: KeyEnyaqSoc, Name: "currentSoc", Value: "60"}}

	soc, err := testProvider(data).Soc()
	require.NoError(t, err)
	assert.Equal(t, 60.0, soc)
}

// TestSocBatteryStateReportPreferred reproduces #35100: battery_level_HV.value
// stays stale while the car sleeps, the current soc key keeps updating.
func TestSocBatteryStateReportPreferred(t *testing.T) {
	data := []point{
		{Name: FieldHvBatteryLevelValue, Value: "94.0"},
		{Key: "7bddd5e7-43a4-3878-bd63-9502782f77a5", Name: "battery_state_report.soc", Value: "77"}, // soc at charge start
		{Key: KeyBatteryStateReportSoc, Name: "battery_state_report.soc", Value: "100"},
	}

	soc, err := testProvider(data).Soc()
	require.NoError(t, err)
	assert.Equal(t, 100.0, soc)
}

// TestGetLimitSocBcamThreshold reproduces issue #32241: the Audi Q4 e-tron
// reports its battery care mode SoC limit only under the battery care mode
// field, not under the generic target soc field, while battery care mode is
// active.
func TestGetLimitSocBcamThreshold(t *testing.T) {
	data := []point{
		{Name: FieldChargeBcamThreshold, Value: "80"},
		{Name: FieldBcamActivation, Value: BcamActivationActivated},
	}

	limit, err := testProvider(data).GetLimitSoc()
	require.NoError(t, err)
	assert.Equal(t, int64(80), limit)
}

// TestGetLimitSocBcamThresholdInactive reproduces issue #32520: with battery
// care mode inactive, the care mode threshold is unrelated to the actual
// target soc and must not shadow it.
func TestGetLimitSocBcamThresholdInactive(t *testing.T) {
	data := []point{
		{Name: FieldTargetSoc, Value: "100"},
		{Name: FieldChargeBcamThreshold, Value: "80"},
		{Name: FieldBcamActivation, Value: "BCAM_ACTIVATION_DEACTIVATED"},
	}

	limit, err := testProvider(data).GetLimitSoc()
	require.NoError(t, err)
	assert.Equal(t, int64(100), limit)
}

// TestPoints guards that a data point with a generic field name ("value") is
// stored once yet found by both its unique key and its name.
func TestPoints(t *testing.T) {
	data := points([]dataPoint{
		{Key: KeyRangeID3, DataFieldName: "value", Value: "317"},
		{DataFieldName: FieldOdometer, Value: "22164"},
	})

	require.Len(t, data, 2, "ID.3 range stored once, not duplicated under key and name")
	assert.Equal(t, "317", find(data, KeyRangeID3).Value, "found by key")
	assert.Equal(t, "317", find(data, "value").Value, "found by generic name")
	assert.Equal(t, "22164", find(data, FieldOdometer).Value, "named field found by name")
}

func TestKeepData(t *testing.T) {
	data := []point{{Value: "1"}}

	assert.True(t, keepData(data, time.Now()), "recent success")
	assert.False(t, keepData(data, time.Now().Add(-2*errorGrace)), "persistent failure")
	assert.False(t, keepData(data, time.Time{}), "never succeeded")
	assert.False(t, keepData(nil, time.Now()), "no data")
}

func TestRetryAfter(t *testing.T) {
	statusError := func(status int, header http.Header) error {
		_, err := request.ReadBody(&http.Response{StatusCode: status, Header: header, Body: http.NoBody})
		return err
	}

	d, ok := retryAfter(statusError(http.StatusTooManyRequests, http.Header{"Retry-After": {"5"}}))
	assert.True(t, ok)
	assert.Equal(t, 5*time.Second, d)

	_, ok = retryAfter(statusError(http.StatusTooManyRequests, nil))
	assert.False(t, ok, "no header")

	_, ok = retryAfter(statusError(http.StatusServiceUnavailable, http.Header{"Retry-After": {"5"}}))
	assert.False(t, ok, "not rate limited")

	_, ok = retryAfter(io.EOF)
	assert.False(t, ok, "no status error")
}

// testArchive returns a dataset zip archive holding a single data point
func testArchive(t *testing.T) []byte {
	t.Helper()

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("data.json")
	require.NoError(t, err)
	_, err = w.Write([]byte(`{"Data":[{"dataFieldName":"soc","value":"50"}]}`))
	require.NoError(t, err)
	require.NoError(t, zw.Close())

	return buf.Bytes()
}

func TestProviderRateLimit(t *testing.T) {
	archive := testArchive(t)
	created := time.Now().Add(-3 * time.Hour)

	var (
		mu        sync.Mutex
		names     = []string{"a.zip"}
		limited   bool
		downloads []string
	)

	respond := func(req *http.Request, status int, header http.Header, body []byte) *http.Response {
		return &http.Response{
			StatusCode: status,
			Header:     header,
			Body:       io.NopCloser(bytes.NewReader(body)),
			Request:    req,
		}
	}

	log := util.NewLogger("test-ratelimit")
	var errorLog bytes.Buffer
	log.ERROR.SetOutput(&errorLog)

	v := &API{Helper: request.NewHelper(log), log: log}
	v.Client.Transport = loginRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		mu.Lock()
		defer mu.Unlock()

		switch {
		case strings.HasSuffix(req.URL.Path, "/metadata/partial"):
			return respond(req, http.StatusOK, nil, []byte(`{"Identifier":"id"}`)), nil

		case strings.HasSuffix(req.URL.Path, "/list"):
			files := make([]string, 0, len(names))
			for i, name := range names {
				files = append(files, fmt.Sprintf(`{"name":%q,"createdOn":%q}`, name, created.Add(time.Duration(i)*time.Minute).Format(time.RFC3339)))
			}
			return respond(req, http.StatusOK, nil, []byte("["+strings.Join(files, ",")+"]")), nil

		case strings.HasSuffix(req.URL.Path, "/download"):
			if limited {
				limited = false
				return respond(req, http.StatusTooManyRequests, http.Header{"Retry-After": {"0"}}, nil), nil
			}
			downloads = append(downloads, req.Header.Get("filename"))
			return respond(req, http.StatusOK, nil, archive), nil
		}

		return respond(req, http.StatusNotFound, nil, nil), nil
	})

	p := NewProvider(log, v, "VIN", time.Hour)

	_, err := p.statusG()
	require.NoError(t, err)

	mu.Lock()
	names = append(names, "b.zip")
	limited = true
	mu.Unlock()

	util.ResetCached()

	_, err = p.statusG()
	require.NoError(t, err, "last data is served while rate limited")
	assert.Empty(t, errorLog.String(), "rate limit is not an error")

	require.Eventually(t, func() bool {
		if _, err := p.statusG(); err != nil {
			return false
		}

		mu.Lock()
		defer mu.Unlock()
		return len(downloads) == 2
	}, 3*time.Second, 10*time.Millisecond, "retried without waiting for the cache interval")

	assert.Equal(t, []string{"a.zip", "b.zip"}, downloads)
}
