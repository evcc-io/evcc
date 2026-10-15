package ocpp

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/lorenzodonini/ocpp-go/ocppj"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setMirrorRules installs a forwarder rule with the given mirrors for the charger and reconciles them
func setMirrorRules(t *testing.T, id string, mirrors ...MirrorRule) {
	t.Helper()

	forwarderMu.Lock()
	forwarderRules = []ForwarderRule{{StationID: id, UpstreamURL: "ws://main.invalid", Mirrors: mirrors}}
	forwarderMu.Unlock()

	startMirrors(id)
}

const testMeterFrame = `[2,"m-1","MeterValues",{"connectorId":1,"meterValue":[]}]`

func TestMirrorReceivesCopyAndRejectsCalls(t *testing.T) {
	const id = "mirror-copy-test"

	frames := make(chan []byte, 16)

	srv := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, &websocket.AcceptOptions{Subprotocols: []string{"ocpp1.6"}})
		if err != nil {
			return
		}
		defer c.CloseNow()

		_ = c.Write(r.Context(), websocket.MessageText, []byte(`[2,"srv-1","TriggerMessage",{"requestedMessage":"Heartbeat"}]`))

		for {
			_, data, err := c.Read(r.Context())
			if err != nil {
				return
			}
			frames <- data
		}
	}))
	srv.Start()

	cleanup := forwarderTestSetup(t, id)
	defer cleanup()

	setMirrorRules(t, id, MirrorRule{UpstreamURL: "ws" + strings.TrimPrefix(srv.URL, "http")})
	defer stopMirrors(id)

	mirrorCall(id, "MeterValues", []byte(testMeterFrame))

	var boot, meter, rejected bool
	deadline := time.After(5 * time.Second)
	for !(boot && meter && rejected) {
		select {
		case data := <-frames:
			msgType, msgID, action, err := parseOCPPFrame(data)
			require.NoError(t, err)

			switch {
			case msgType == ocppj.CALL && action == "BootNotification":
				boot = true
			case msgType == ocppj.CALL && action == "MeterValues":
				meter = true
			case msgType == ocppj.CALL_ERROR && msgID == "srv-1":
				rejected = true
			}
		case <-deadline:
			t.Fatalf("timeout: boot=%v meter=%v rejected=%v", boot, meter, rejected)
		}
	}
}

func TestMirrorReconcile(t *testing.T) {
	const id = "mirror-reconcile-test"

	srv := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, &websocket.AcceptOptions{Subprotocols: []string{"ocpp1.6"}})
		if err != nil {
			return
		}
		defer c.CloseNow()
		for {
			if _, _, err := c.Read(r.Context()); err != nil {
				return
			}
		}
	}))
	srv.Start()

	cleanup := forwarderTestSetup(t, id)
	defer cleanup()

	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	a := MirrorRule{UpstreamURL: url + "/a"}
	b := MirrorRule{UpstreamURL: url + "/b"}

	setMirrorRules(t, id, a, b)
	defer stopMirrors(id)

	count := func() int {
		mirrorsMu.Lock()
		defer mirrorsMu.Unlock()
		return len(mirrorSessions[id])
	}
	assert.Equal(t, 2, count())

	setMirrorRules(t, id, b)
	assert.Equal(t, 1, count())

	setMirrorRules(t, id)
	assert.Equal(t, 0, count())
}

func TestMirrorStatusConnector(t *testing.T) {
	connector, ok := statusConnector([]byte(`[2,"s-1","StatusNotification",{"connectorId":2,"status":"Available"}]`))
	assert.True(t, ok)
	assert.Equal(t, 2, connector)

	_, ok = statusConnector([]byte(`{}`))
	assert.False(t, ok)
}

func TestMirrorRedacted(t *testing.T) {
	rule := ForwarderRule{StationID: "s", UpstreamURL: "ws://a", Mirrors: []MirrorRule{{UpstreamURL: "ws://b", Password: "secret"}}}

	redacted := rule.Redacted()
	assert.NotEqual(t, "secret", redacted.Mirrors[0].Password)
	assert.Equal(t, "secret", rule.Mirrors[0].Password, "original must stay untouched")
}

type memMirrorStats struct {
	data map[string]mirrorStats
}

func (m *memMirrorStats) load() (map[string]mirrorStats, error) { return m.data, nil }
func (m *memMirrorStats) save(d map[string]mirrorStats) error {
	m.data = d
	return nil
}

func init() {
	mirrorStatsBackend = &memMirrorStats{}
}

func TestMeterEnergyWh(t *testing.T) {
	wh, ok := meterEnergyWh([]byte(`[2,"m","MeterValues",{"meterValue":[{"sampledValue":[{"value":"7.5","measurand":"Current.Import"},{"value":"15440.5","measurand":"Energy.Active.Import.Register","unit":"kWh"}]}]}]`))
	assert.True(t, ok)
	assert.InDelta(t, 15440500, wh, 1e-6)

	wh, ok = meterEnergyWh([]byte(`[2,"m","MeterValues",{"meterValue":[{"sampledValue":[{"value":"1200"}]}]}]`))
	assert.True(t, ok, "no measurand means the energy register")
	assert.Equal(t, 1200.0, wh)

	_, ok = meterEnergyWh([]byte(`[2,"m","MeterValues",{"meterValue":[{"sampledValue":[{"value":"5","phase":"L1","measurand":"Energy.Active.Import.Register"}]}]}]`))
	assert.False(t, ok, "per phase values are not the register")

	_, ok = meterEnergyWh([]byte(`[2,"m","MeterValues",{"meterValue":[{"sampledValue":[{"value":"NaN"}]}]}]`))
	assert.False(t, ok)
}

func TestMirrorRecord(t *testing.T) {
	ms := &mirrorSession{chargerID: "stats-test", rule: MirrorRule{UpstreamURL: "ws://x"}}
	meter := func(wh string) []byte {
		return []byte(`[2,"m","MeterValues",{"meterValue":[{"sampledValue":[{"value":"` + wh + `"}]}]}]`)
	}

	now := time.Date(2026, 10, 15, 10, 5, 0, 0, time.UTC)
	ms.record(meter("1000"), now)
	ms.record(meter("1500"), now.Add(time.Minute))
	ms.record([]byte(`[2,"s","StatusNotification",{"connectorId":1}]`), now.Add(2*time.Minute))
	ms.record(meter("2000"), now.Add(time.Hour))
	ms.record(meter("10"), now.Add(time.Hour+time.Minute))

	mirrorsMu.Lock()
	defer mirrorsMu.Unlock()

	st := statsLocked(mirrorStatsKey("stats-test", ms.rule.Key()))
	require.Len(t, st.Hours, 2)

	assert.Equal(t, 3, st.Hours[0].Messages)
	assert.Equal(t, 2, st.Hours[0].MeterValues)
	assert.Equal(t, 500.0, st.Hours[0].EnergyWh, "the first sample has nothing to step from")

	assert.Equal(t, 2, st.Hours[1].Messages)
	assert.Equal(t, 500.0, st.Hours[1].EnergyWh, "a register reset adds no energy")
	assert.Equal(t, 10.0, st.LastWh)
}

func TestMirrorNotStartedForDisconnectedCharger(t *testing.T) {
	const id = "mirror-disconnected-test"

	forwarderMu.Lock()
	forwarderRules = []ForwarderRule{{StationID: id, UpstreamURL: "ws://main.invalid", Mirrors: []MirrorRule{{UpstreamURL: "ws://copy.invalid"}}}}
	forwarderMu.Unlock()
	defer func() {
		forwarderMu.Lock()
		forwarderRules = nil
		forwarderMu.Unlock()
	}()

	startMirrors(id)

	mirrorsMu.Lock()
	defer mirrorsMu.Unlock()
	assert.Empty(t, mirrorSessions[id])
}

func TestEffectiveMirrors(t *testing.T) {
	rule := ForwarderRule{
		StationID:   "EVB-1",
		UpstreamURL: "wss://main.example.com/",
		Mirrors: []MirrorRule{
			{UpstreamURL: "wss://main.example.com", UpstreamStationID: "EVB-1"}, // the main connection under its effective id
			{UpstreamURL: "wss://copy.example.com"},
			{UpstreamURL: "wss://copy.example.com/", UpstreamStationID: "EVB-1"}, // the same station as the previous one
			{UpstreamURL: "wss://copy.example.com", UpstreamStationID: "other"},
			{},
		},
	}

	got := rule.effectiveMirrors("EVB-1")
	require.Len(t, got, 2)
	assert.Equal(t, "wss://copy.example.com", got[0].UpstreamURL)
	assert.Equal(t, "other", got[1].UpstreamStationID)
}

func TestMirrorFrameSuperseded(t *testing.T) {
	status := mirrorFrame{seq: 5, action: "StatusNotification"}
	assert.True(t, status.superseded(5), "covered by the snapshot")
	assert.True(t, status.superseded(9))
	assert.False(t, status.superseded(4), "newer than the snapshot")

	meter := mirrorFrame{seq: 3, action: "MeterValues"}
	assert.False(t, meter.superseded(9), "samples are never replayed")
}

type failingMirrorStats struct {
	loadErr error
	saveErr error
	saved   int
}

func (f *failingMirrorStats) load() (map[string]mirrorStats, error) { return nil, f.loadErr }
func (f *failingMirrorStats) save(map[string]mirrorStats) error {
	f.saved++
	return f.saveErr
}

func useMirrorStats(t *testing.T, backend mirrorStatsStore) {
	t.Helper()

	previous := mirrorStatsBackend

	mirrorsMu.Lock()
	mirrorStatsBackend = backend
	mirrorStatsData = make(map[string]*mirrorStats)
	mirrorStatsLoaded, mirrorStatsDirty, mirrorStatsReadOnly = false, false, false
	mirrorsMu.Unlock()

	t.Cleanup(func() {
		mirrorsMu.Lock()
		mirrorStatsBackend = previous
		mirrorStatsData = make(map[string]*mirrorStats)
		mirrorStatsLoaded, mirrorStatsDirty, mirrorStatsReadOnly = false, false, false
		mirrorsMu.Unlock()
	})
}

func TestMirrorStatsFailedSaveStaysDirty(t *testing.T) {
	backend := &failingMirrorStats{saveErr: assert.AnError}
	useMirrorStats(t, backend)

	ms := &mirrorSession{chargerID: "stats-save-test", rule: MirrorRule{UpstreamURL: "ws://x"}}
	ms.record([]byte(`[2,"s","StatusNotification",{"connectorId":1}]`), time.Now())

	flushMirrorStats()
	assert.Equal(t, 1, backend.saved)

	backend.saveErr = nil
	flushMirrorStats()
	assert.Equal(t, 2, backend.saved, "the failed save is retried")

	flushMirrorStats()
	assert.Equal(t, 2, backend.saved, "nothing left to save")
}

func TestMirrorStatsUnreadableStoreIsNotOverwritten(t *testing.T) {
	backend := &failingMirrorStats{loadErr: assert.AnError}
	useMirrorStats(t, backend)

	ms := &mirrorSession{chargerID: "stats-corrupt-test", rule: MirrorRule{UpstreamURL: "ws://x"}}
	ms.record([]byte(`[2,"s","StatusNotification",{"connectorId":1}]`), time.Now())

	flushMirrorStats()
	assert.Zero(t, backend.saved)
}

func TestMirrorStatsKeepLastSent(t *testing.T) {
	useMirrorStats(t, &memMirrorStats{})

	rule := MirrorRule{UpstreamURL: "ws://x"}
	ms := &mirrorSession{chargerID: "stats-lastsent-test", rule: rule}
	at := time.Date(2026, 10, 15, 10, 5, 0, 0, time.UTC)
	ms.record([]byte(`[2,"s","StatusNotification",{"connectorId":1}]`), at)

	status := mirrorStatus("stats-lastsent-test", []MirrorRule{rule})
	require.Len(t, status, 1)
	require.NotNil(t, status[0].LastSent, "without a running session")
	assert.True(t, status[0].LastSent.Equal(at))
}

func TestPruneMirrorStats(t *testing.T) {
	const id = "stats-prune-test"
	useMirrorStats(t, &memMirrorStats{})

	kept := MirrorRule{UpstreamURL: "ws://kept"}
	removed := MirrorRule{UpstreamURL: "ws://removed"}

	for _, r := range []MirrorRule{kept, removed} {
		ms := &mirrorSession{chargerID: id, rule: r}
		ms.record([]byte(`[2,"s","StatusNotification",{"connectorId":1}]`), time.Now())
	}

	forwarderMu.Lock()
	forwarderRules = []ForwarderRule{{StationID: id, UpstreamURL: "ws://main.invalid", Mirrors: []MirrorRule{kept}}}
	forwarderMu.Unlock()
	defer func() {
		forwarderMu.Lock()
		forwarderRules = nil
		forwarderMu.Unlock()
	}()

	pruneMirrorStats()

	mirrorsMu.Lock()
	defer mirrorsMu.Unlock()
	assert.Contains(t, mirrorStatsData, mirrorStatsKey(id, kept.Key()))
	assert.NotContains(t, mirrorStatsData, mirrorStatsKey(id, removed.Key()))
}
