package eudataact

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testVin = "WVWZZZ"

// testPortal serves datasets whose name is the delivered soc
type testPortal struct {
	files     []dataset
	downloads map[string]int
	fail      map[string]int // remaining failures per dataset
	delay     time.Duration  // per download
}

func (p *testPortal) RoundTrip(req *http.Request) (*http.Response, error) {
	var body []byte
	status := http.StatusOK

	switch {
	case strings.HasSuffix(req.URL.Path, "/list"):
		body, _ = json.Marshal(map[string]any{"files": p.files})

	case strings.HasSuffix(req.URL.Path, "/download"):
		name := req.Header.Get("filename")
		p.downloads[name]++

		select {
		case <-time.After(p.delay):
		case <-req.Context().Done():
			return nil, req.Context().Err()
		}

		if p.fail[name] > 0 {
			p.fail[name]--
			status = http.StatusInternalServerError
		} else {
			body = datasetZip(strings.TrimSuffix(name, ".zip"))
		}
	}

	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewReader(body)),
		Request:    req,
	}, nil
}

func datasetZip(soc string) []byte {
	raw, _ := json.Marshal(datasetFile{Data: []dataPoint{{DataFieldName: FieldSoc, Value: soc}}})

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, _ := zw.Create("data.json")
	_, _ = w.Write(raw)
	_ = zw.Close()

	return buf.Bytes()
}

// addDatasets appends n datasets delivered after start, named by soc
func (p *testPortal) addDatasets(start time.Time, n int) {
	for i := range n {
		p.files = append(p.files, dataset{
			Name:      fmt.Sprintf("%d.zip", len(p.files)+1),
			CreatedOn: start.Add(time.Duration(i+1) * time.Minute),
		})
	}
}

// testStore returns a store for a vehicle that has already been polled once at start
func testStore(start time.Time) (*testPortal, *API, *store) {
	portal := &testPortal{
		downloads: make(map[string]int),
		fail:      make(map[string]int),
	}

	log := util.NewLogger("test")
	a := &API{Helper: request.NewHelper(log), log: log}
	a.Client.Transport = portal

	s := sharedStore(a)

	v := s.state(testVin)
	v.identifier = "id"
	v.after = start
	v.data = []point{{Name: FieldSoc, Value: "0", Seq: 1}}
	v.seq = 1

	return portal, a, s
}

func socOf(data []point) string {
	if p := find(data, FieldSoc); p != nil {
		return p.Value
	}
	return ""
}

var testStart = time.Date(2026, 10, 30, 8, 0, 0, 0, time.UTC)

// slowPortal makes downloads take a third of a shortened update timeout
func slowPortal(t *testing.T, portal *testPortal) {
	timeout := updateTimeout
	t.Cleanup(func() { updateTimeout = timeout })

	updateTimeout = 300 * time.Millisecond
	portal.delay = updateTimeout / 3
}

// timedUpdate fails the test if update exceeds its timeout
func timedUpdate(t *testing.T, s *store) (time.Time, error) {
	t.Helper()

	start := time.Now()
	ts, err := s.update(testVin)
	require.Less(t, time.Since(start), updateTimeout+100*time.Millisecond)

	return ts, err
}

// TestUpdateFewPending keeps the normal case in a single poll
func TestUpdateFewPending(t *testing.T) {
	portal, _, s := testStore(testStart)
	portal.addDatasets(testStart, maxBackfill)

	ts, err := s.update(testVin)
	require.NoError(t, err)
	assert.Equal(t, testStart.Add(maxBackfill*time.Minute), ts)
	assert.Equal(t, fmt.Sprint(maxBackfill), socOf(s.snapshot(testVin)))
}

// TestUpdateCatchUp continues a catch-up over several polls, each within the timeout
func TestUpdateCatchUp(t *testing.T) {
	total := 10

	portal, _, s := testStore(testStart)
	portal.addDatasets(testStart, total)
	slowPortal(t, portal)

	var retries int
	ts, err := timedUpdate(t, s)
	for errors.Is(err, api.ErrMustRetry) && retries < total {
		retries++
		ts, err = timedUpdate(t, s)
	}

	require.NoError(t, err)
	assert.Positive(t, retries)
	assert.Equal(t, testStart.Add(time.Duration(total)*time.Minute), ts)
	assert.Equal(t, fmt.Sprint(total), socOf(s.snapshot(testVin)))
	require.Len(t, portal.downloads, total)
}

// TestUpdateTimeoutNoProgress surfaces the timeout when not a single dataset arrives
func TestUpdateTimeoutNoProgress(t *testing.T) {
	portal, _, s := testStore(testStart)
	portal.addDatasets(testStart, 1)
	slowPortal(t, portal)
	portal.delay = 2 * updateTimeout

	_, err := timedUpdate(t, s)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	assert.NotErrorIs(t, err, api.ErrMustRetry)
}

// TestUpdateCatchUpError keeps the datasets before a failure and resumes from there
func TestUpdateCatchUpError(t *testing.T) {
	total := 3

	portal, _, s := testStore(testStart)
	portal.addDatasets(testStart, total)
	portal.fail[fmt.Sprintf("%d.zip", total)] = 1

	_, err := s.update(testVin)
	require.Error(t, err)
	assert.Equal(t, fmt.Sprint(total-1), socOf(s.snapshot(testVin)))

	_, err = s.update(testVin)
	require.NoError(t, err)
	assert.Equal(t, fmt.Sprint(total), socOf(s.snapshot(testVin)))
	assert.Equal(t, 1, portal.downloads[fmt.Sprintf("%d.zip", total-1)], "not downloaded again")
	assert.Equal(t, 2, portal.downloads[fmt.Sprintf("%d.zip", total)], "retried")
}

// TestProviderCatchUp ensures old data is not served during a catch-up, since the
// loadpoint reads once on plug-in unless it gets ErrMustRetry.
func TestProviderCatchUp(t *testing.T) {
	total := 10

	portal, a, _ := testStore(testStart)
	p := NewProvider(util.NewLogger("test"), a, testVin, time.Hour, nil)

	soc, err := p.Soc()
	require.NoError(t, err, "nothing pending, recent success")
	assert.Equal(t, 0.0, soc)

	portal.addDatasets(testStart, total)
	slowPortal(t, portal)
	util.ResetCached()

	_, err = p.Soc()
	require.ErrorIs(t, err, api.ErrMustRetry)

	for range total {
		if soc, err = p.Soc(); !errors.Is(err, api.ErrMustRetry) {
			break
		}
	}

	require.NoError(t, err)
	assert.Equal(t, float64(total), soc)
}
