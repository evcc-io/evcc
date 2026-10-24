package ecoflow

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/charger/ecoflow/ecoflowtest"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testUserID    = "1234567890"
	testTelemetry = "/app/device/property/" + ecoflowtest.Serial
	testGetAll    = "/app/" + testUserID + "/" + ecoflowtest.Serial + "/thing/property/get"
)

// fakeBroker records login, dial and backoff sleeps in one event log. Each
// login and dial consumes the next scripted error; an exhausted script
// succeeds.
type fakeBroker struct {
	mu         sync.Mutex
	events     []string
	dialErrs   []error
	loginErrs  []error
	logins     int
	transports []*ecoflowtest.Transport
	lost       []func(error)
	onSleep    func() // runs after recording a sleep, outside the lock
}

func (b *fakeBroker) record(event string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.events = append(b.events, event)
}

func (b *fakeBroker) login() (*Session, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.events = append(b.events, "login")
	if len(b.loginErrs) > 0 {
		err := b.loginErrs[0]
		b.loginErrs = b.loginErrs[1:]
		if err != nil {
			return nil, err
		}
	}
	b.logins++
	return &Session{UserID: testUserID, Account: fmt.Sprintf("account-%d", b.logins)}, nil
}

func (b *fakeBroker) dial(session *Session, onLost func(error)) (Transport, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.events = append(b.events, "dial "+session.Account)

	if len(b.dialErrs) > 0 {
		err := b.dialErrs[0]
		b.dialErrs = b.dialErrs[1:]
		if err != nil {
			return nil, err
		}
	}

	t := ecoflowtest.NewTransport()
	b.transports = append(b.transports, t)
	b.lost = append(b.lost, onLost)
	return t, nil
}

func (b *fakeBroker) sleep(d time.Duration) {
	b.record("sleep " + d.String())
	b.mu.Lock()
	fn := b.onSleep
	b.mu.Unlock()
	if fn != nil {
		fn()
	}
}

// Events returns the recorded events and clears the log.
func (b *fakeBroker) Events() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	res := b.events
	b.events = nil
	return res
}

func (b *fakeBroker) Transport() *ecoflowtest.Transport {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.transports[len(b.transports)-1]
}

func (b *fakeBroker) Dials() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.transports)
}

func newTestConnection(t *testing.T, broker *fakeBroker, clk clock.Clock) (*Connection, *Wallbox) {
	t.Helper()

	conn, err := Connect(util.NewLogger("test"), broker.login, broker.dial, clk, broker.sleep)
	require.NoError(t, err)

	wb, err := conn.Wallbox(ecoflowtest.Serial, ecoflowtest.Gateway, time.Minute)
	require.NoError(t, err)

	require.Equal(t, []string{"login", "dial account-1"}, broker.Events())
	return conn, wb
}

func getAlls(transport *ecoflowtest.Transport) int {
	var n int
	for _, m := range transport.Published() {
		if m.Topic == testGetAll {
			n++
		}
	}
	return n
}

func TestReconnectBacksOffExponentiallyUpTo60s(t *testing.T) {
	broker := &fakeBroker{}
	conn, wb := newTestConnection(t, broker, clock.NewMock())

	unreachable := errors.New("network unreachable")
	broker.dialErrs = []error{unreachable, unreachable, unreachable, unreachable, unreachable}

	conn.reconnect()

	assert.Equal(t, []string{
		"sleep 5s", "dial account-1",
		"sleep 10s", "dial account-1",
		"sleep 20s", "dial account-1",
		"sleep 40s", "dial account-1",
		"sleep 1m0s", "dial account-1",
		"sleep 1m0s", "dial account-1",
	}, broker.Events())

	// the new transport carries the telemetry again
	transport := broker.Transport()
	assert.Equal(t, 1, getAlls(transport), "get-all on reconnect")
	require.NoError(t, transport.Feed(testTelemetry, ecoflowtest.Heartbeat("state2-3p")))
	_, err := wb.Heartbeat()
	assert.NoError(t, err)
}

func TestReconnectLogsInAgainWhenBrokerRejectsCredentials(t *testing.T) {
	for _, code := range []byte{4, 5, 134, 135} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			broker := &fakeBroker{}
			conn, _ := newTestConnection(t, broker, clock.NewMock())

			broker.dialErrs = []error{fmt.Errorf("connect: %w", &ConnackError{Code: code})}
			conn.reconnect()

			assert.Equal(t, []string{
				"sleep 5s", "dial account-1",
				"sleep 10s", "login", "dial account-2",
			}, broker.Events())
		})
	}
}

func TestReconnectKeepsSessionOnOtherConnackCodes(t *testing.T) {
	broker := &fakeBroker{}
	conn, _ := newTestConnection(t, broker, clock.NewMock())

	broker.dialErrs = []error{&ConnackError{Code: 3}} // server unavailable
	conn.reconnect()

	assert.Equal(t, []string{
		"sleep 5s", "dial account-1",
		"sleep 10s", "dial account-1",
	}, broker.Events())
}

func TestReconnectRetriesFailedLoginWithBackoff(t *testing.T) {
	broker := &fakeBroker{}
	conn, _ := newTestConnection(t, broker, clock.NewMock())

	broker.dialErrs = []error{&ConnackError{Code: 5}}
	broker.loginErrs = []error{errors.New("login: code 500")}
	conn.reconnect()

	assert.Equal(t, []string{
		"sleep 5s", "dial account-1",
		"sleep 10s", "login",
		"sleep 20s", "login", "dial account-2",
	}, broker.Events())
}

func TestConnectionLossReconnectsOnce(t *testing.T) {
	broker := &fakeBroker{}
	_, _ = newTestConnection(t, broker, clock.NewMock())

	broker.lost[0](errors.New("EOF"))
	require.Eventually(t, func() bool { return broker.Dials() == 2 }, time.Second, time.Millisecond)

	// a late loss report of the replaced connection is ignored
	broker.lost[0](errors.New("EOF"))
	time.Sleep(10 * time.Millisecond)

	assert.Equal(t, 2, broker.Dials())
	assert.Equal(t, []string{"sleep 5s", "dial account-1"}, broker.Events())
}

func TestPublishFailsWhileDisconnected(t *testing.T) {
	broker := &fakeBroker{}
	conn, wb := newTestConnection(t, broker, clock.NewMock())

	conn.transport.detach()
	assert.ErrorIs(t, wb.requestAll(), ErrNotConnected)
}

func TestClientIDIsNewForEveryConnect(t *testing.T) {
	app := AppKey{Key: "0123456789abcdef0123456789abcdef", Secret: "secret"}
	a, b := clientID(testUserID, app), clientID(testUserID, app)
	assert.NotEqual(t, a, b)
	assert.Regexp(t, `^WEB_[0-9a-f-]{36}_`+testUserID+`_`+app.Key+`_\d{13}_[0-9A-F]{32}$`, a)
}

func TestLoginIsRenewedEvery20hWithoutReconnect(t *testing.T) {
	clk := clock.NewMock()
	broker := &fakeBroker{}
	conn, wb := newTestConnection(t, broker, clk)
	transport := broker.Transport()

	heartbeat := func() {
		require.NoError(t, transport.Feed(testTelemetry, ecoflowtest.Heartbeat("state2-3p")))
	}

	clk.Add(20*time.Hour - time.Minute)
	heartbeat()
	conn.check()
	assert.Empty(t, broker.Events(), "too early")

	clk.Add(time.Minute)
	heartbeat()
	conn.check()
	assert.Equal(t, []string{"login"}, broker.Events())

	// telemetry keeps flowing on the same connection
	heartbeat()
	_, err := wb.Heartbeat()
	assert.NoError(t, err)
	assert.Equal(t, 1, broker.Dials())

	// the renewed credentials are used for the next connect
	conn.reconnect()
	assert.Equal(t, []string{"sleep 5s", "dial account-2"}, broker.Events())
}

func TestFailedLoginRenewalKeepsSession(t *testing.T) {
	clk := clock.NewMock()
	broker := &fakeBroker{loginErrs: []error{nil, errors.New("login: code 500")}}
	conn, _ := newTestConnection(t, broker, clk)

	clk.Add(20 * time.Hour)
	require.NoError(t, broker.Transport().Feed(testTelemetry, ecoflowtest.Heartbeat("state2-3p")))
	conn.check()
	assert.Equal(t, []string{"login"}, broker.Events())

	conn.reconnect()
	assert.Equal(t, []string{"sleep 5s", "dial account-1"}, broker.Events())
}

func TestStaleTelemetryRequestsSnapshotThenReconnects(t *testing.T) {
	clk := clock.NewMock()
	broker := &fakeBroker{}
	conn, _ := newTestConnection(t, broker, clk)
	transport := broker.Transport()

	clk.Add(20 * time.Minute)
	require.NoError(t, transport.Feed(testTelemetry, ecoflowtest.Heartbeat("state2-3p")))

	clk.Add(1200*time.Second - time.Second)
	conn.check()
	assert.Equal(t, 1, getAlls(transport), "heartbeat younger than 1200 s")

	clk.Add(time.Second)
	conn.check()
	assert.Equal(t, 2, getAlls(transport), "get-all after 1200 s")

	clk.Add(5*time.Minute - time.Second)
	conn.check()
	assert.Equal(t, 2, getAlls(transport), "get-all only once")
	assert.Empty(t, broker.Events(), "waiting for the snapshot")

	clk.Add(time.Second)
	conn.check() // reconnects in the background
	require.Eventually(t, func() bool {
		return broker.Dials() == 2 && getAlls(broker.Transport()) == 1
	}, time.Second, time.Millisecond, "get-all on the new connection")
	assert.Equal(t, []string{"sleep 5s", "dial account-1"}, broker.Events())
}

func TestSnapshotAfterStaleTelemetryKeepsConnection(t *testing.T) {
	clk := clock.NewMock()
	broker := &fakeBroker{}
	conn, _ := newTestConnection(t, broker, clk)
	transport := broker.Transport()

	clk.Add(1200 * time.Second)
	conn.check()
	assert.Equal(t, 2, getAlls(transport))

	clk.Add(time.Minute)
	require.NoError(t, transport.Feed(testTelemetry, ecoflowtest.Heartbeat("state2-3p")))

	clk.Add(10 * time.Minute)
	conn.check()
	assert.Empty(t, broker.Events())
	assert.Equal(t, 2, getAlls(transport))
}

func TestClosedConnectionDoesNotReconnect(t *testing.T) {
	broker := &fakeBroker{}
	conn, _ := newTestConnection(t, broker, clock.NewMock())

	conn.Close()
	conn.reconnect()
	assert.Empty(t, broker.Events())
}

func TestCloseStopsRunningReconnect(t *testing.T) {
	broker := &fakeBroker{}
	conn, _ := newTestConnection(t, broker, clock.NewMock())

	broker.dialErrs = []error{errors.New("network unreachable")}
	broker.onSleep = func() {
		if broker.Dials() == 1 {
			conn.Close()
		}
	}
	conn.reconnect()

	assert.Equal(t, []string{"sleep 5s"}, broker.Events())
}

func TestCloseStopsWatch(t *testing.T) {
	broker := &fakeBroker{}
	conn, _ := newTestConnection(t, broker, clock.NewMock())

	done := make(chan struct{})
	go func() {
		conn.watch()
		close(done)
	}()

	conn.Close()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("watch still running")
	}
}

// A broker that stays down must not stall the watchdog.
func TestCheckDoesNotWaitForReconnect(t *testing.T) {
	clk := clock.NewMock()
	broker := &fakeBroker{}
	conn, _ := newTestConnection(t, broker, clk)

	release := make(chan struct{})
	broker.onSleep = func() { <-release }

	clk.Add(1200 * time.Second)
	conn.check() // get-all
	clk.Add(5 * time.Minute)

	checked := make(chan struct{})
	go func() {
		conn.check()
		close(checked)
	}()
	select {
	case <-checked:
	case <-time.After(time.Second):
		t.Fatal("check blocked on reconnect")
	}

	// a second reconnect while one is running returns at once
	require.Eventually(t, conn.reconnecting.Load, time.Second, time.Millisecond)
	conn.reconnect()

	close(release)
	require.Eventually(t, func() bool { return broker.Dials() == 2 }, time.Second, time.Millisecond)
	time.Sleep(10 * time.Millisecond)
	assert.Equal(t, 2, broker.Dials())
}

// Re-creating a wallbox (config reload) replaces it instead of adding a second one.
func TestWallboxIsReplacedForSameSerial(t *testing.T) {
	broker := &fakeBroker{}
	conn, _ := newTestConnection(t, broker, clock.NewMock())

	wb, err := conn.Wallbox(ecoflowtest.Serial, ecoflowtest.Gateway, time.Minute)
	require.NoError(t, err)

	conn.reconnect()
	transport := broker.Transport()
	assert.Equal(t, 1, getAlls(transport), "one get-all per serial")

	require.NoError(t, transport.Feed(testTelemetry, ecoflowtest.Heartbeat("state2-3p")))
	_, err = wb.Heartbeat()
	assert.NoError(t, err, "the new wallbox receives telemetry")
}

// sharedTest counts the connects of a fake shared-connection factory.
type sharedTest struct {
	connects int
}

// uniqueUser keeps repeated test runs apart in the package-wide connection map.
func uniqueUser() string {
	return fmt.Sprintf("user-%d@example.com", time.Now().UnixNano())
}

func (s *sharedTest) get(ctx context.Context, user, password string) *Connection {
	conn, err := sharedConnection(ctx, user, password, func() (*Connection, error) {
		s.connects++
		return NewConnection(util.NewLogger("test"), ecoflowtest.NewTransport(), testUserID), nil
	})
	if err != nil {
		panic(err)
	}
	return conn
}

func TestSharedConnectionLivesWhileUsed(t *testing.T) {
	var s sharedTest
	ctx1, cancel1 := context.WithCancel(context.Background())
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()

	user := uniqueUser()
	a := s.get(ctx1, user, "secret")
	b := s.get(ctx2, user, "secret")
	assert.Same(t, a, b)
	assert.Equal(t, 1, s.connects)

	cancel1()
	time.Sleep(10 * time.Millisecond)
	assert.False(t, a.isClosed(), "still used by the second charger")

	cancel2()
	require.Eventually(t, a.isClosed, time.Second, time.Millisecond)

	c := s.get(context.Background(), user, "secret")
	assert.NotSame(t, a, c)
	assert.Equal(t, 2, s.connects)
}

func TestSharedConnectionChangedPasswordReconnects(t *testing.T) {
	var s sharedTest

	user := uniqueUser()
	a := s.get(context.Background(), user, "old")
	b := s.get(context.Background(), user, "new")

	assert.NotSame(t, a, b)
	assert.True(t, a.isClosed(), "EcoFlow allows one session per account")
	assert.Equal(t, 2, s.connects)
}
