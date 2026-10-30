package eudataact

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
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

const testVin = "WVWZZZ"

// testPortal serves datasets whose name is the delivered soc
type testPortal struct {
	mu        sync.Mutex
	files     []dataset
	lists     int
	downloads map[string]int
	fail      map[string]int // remaining failures per dataset
	gate      chan struct{}  // blocks downloads until closed
}

func (p *testPortal) RoundTrip(req *http.Request) (*http.Response, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	var body []byte
	status := http.StatusOK

	switch {
	case strings.HasSuffix(req.URL.Path, "/list"):
		p.lists++
		body, _ = json.Marshal(map[string]any{"files": p.files})

	case strings.HasSuffix(req.URL.Path, "/download"):
		name := req.Header.Get("filename")
		p.downloads[name]++

		if p.gate != nil {
			gate := p.gate
			p.mu.Unlock()
			<-gate
			p.mu.Lock()
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

// addDatasets appends n datasets delivered after the newest one, named by soc
func (p *testPortal) addDatasets(start time.Time, n int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for i := range n {
		p.files = append(p.files, dataset{
			Name:      fmt.Sprintf("%d.zip", len(p.files)+1),
			CreatedOn: start.Add(time.Duration(i+1) * time.Minute),
		})
	}
}

func (p *testPortal) counts() (int, map[string]int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.lists, maps.Clone(p.downloads)
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

// returns fails the test if f is still blocked after a while
func returns(t *testing.T, f func()) {
	t.Helper()

	done := make(chan struct{})
	go func() {
		defer close(done)
		f()
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		require.FailNow(t, "call blocked")
	}
}

func socOf(data []point) string {
	if p := find(data, FieldSoc); p != nil {
		return p.Value
	}
	return ""
}

// TestUpdateFewPending keeps the normal case synchronous, the read returns fresh data.
func TestUpdateFewPending(t *testing.T) {
	start := time.Date(2026, 10, 30, 8, 0, 0, 0, time.UTC)
	portal, _, s := testStore(start)
	portal.addDatasets(start, maxBackfill)

	ts, err := s.update(testVin)
	require.NoError(t, err)
	assert.Equal(t, start.Add(maxBackfill*time.Minute), ts)
	assert.Equal(t, fmt.Sprint(maxBackfill), socOf(s.snapshot(testVin)))
}

// TestUpdateCatchUp guards the site loop: a long catch-up must not block the
// caller while downloads wait for the rate limit, and reads must not block either.
func TestUpdateCatchUp(t *testing.T) {
	start := time.Date(2026, 10, 30, 8, 0, 0, 0, time.UTC)
	total := maxBackfill + 2

	portal, _, s := testStore(start)
	portal.addDatasets(start, total)
	portal.gate = make(chan struct{})

	returns(t, func() {
		_, err := s.update(testVin)
		assert.ErrorIs(t, err, api.ErrMustRetry)
	})

	returns(t, func() {
		_, err := s.update(testVin)
		assert.ErrorIs(t, err, api.ErrMustRetry)
		assert.Equal(t, "0", socOf(s.snapshot(testVin)))
	})

	lists, _ := portal.counts()
	assert.Equal(t, 1, lists, "no further portal requests while the catch-up runs")

	close(portal.gate)

	var ts time.Time
	require.Eventually(t, func() bool {
		var err error
		ts, err = s.update(testVin)
		return err == nil
	}, 5*time.Second, time.Millisecond)

	assert.Equal(t, start.Add(time.Duration(total)*time.Minute), ts)
	assert.Equal(t, fmt.Sprint(total), socOf(s.snapshot(testVin)), "newest dataset wins")

	_, downloads := portal.counts()
	require.Len(t, downloads, total)
	for name, n := range downloads {
		assert.Equal(t, 1, n, name)
	}
}

// TestUpdateCatchUpError surfaces a failed catch-up once and resumes where it stopped.
func TestUpdateCatchUpError(t *testing.T) {
	start := time.Date(2026, 10, 30, 8, 0, 0, 0, time.UTC)
	total := maxBackfill + 5

	portal, _, s := testStore(start)
	portal.addDatasets(start, total)
	portal.fail["2.zip"] = 1

	_, err := s.update(testVin)
	require.ErrorIs(t, err, api.ErrMustRetry)

	var failed error
	require.Eventually(t, func() bool {
		_, failed = s.update(testVin)
		return !errors.Is(failed, api.ErrMustRetry)
	}, 5*time.Second, time.Millisecond)

	require.Error(t, failed)
	assert.NotErrorIs(t, failed, api.ErrMustRetry)
	assert.Equal(t, "1", socOf(s.snapshot(testVin)), "datasets before the failure are kept")

	require.Eventually(t, func() bool {
		_, err := s.update(testVin)
		return err == nil
	}, 5*time.Second, time.Millisecond)

	assert.Equal(t, fmt.Sprint(total), socOf(s.snapshot(testVin)))

	_, downloads := portal.counts()
	assert.Equal(t, 1, downloads["1.zip"], "not downloaded again")
	assert.Equal(t, 2, downloads["2.zip"], "retried")
}

// TestProviderCatchUp ensures old data is not served during a catch-up, since the
// loadpoint reads once on plug-in unless it gets ErrMustRetry.
func TestProviderCatchUp(t *testing.T) {
	start := time.Date(2026, 10, 30, 8, 0, 0, 0, time.UTC)
	total := maxBackfill + 2

	portal, a, _ := testStore(start)
	p := NewProvider(util.NewLogger("test"), a, testVin, time.Hour)

	soc, err := p.Soc()
	require.NoError(t, err, "nothing pending, recent success")
	assert.Equal(t, 0.0, soc)

	portal.addDatasets(start, total)
	portal.gate = make(chan struct{})
	util.ResetCached()

	returns(t, func() {
		_, err := p.Soc()
		assert.ErrorIs(t, err, api.ErrMustRetry)
	})

	close(portal.gate)

	require.Eventually(t, func() bool {
		soc, err = p.Soc()
		return err == nil
	}, 5*time.Second, time.Millisecond)

	assert.Equal(t, float64(total), soc)
}
