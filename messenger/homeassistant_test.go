package messenger

import (
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockHASender struct {
	mu      sync.Mutex
	domain  string
	service string
	data    map[string]any
	err     error
	callFn  func(domain, service string, data map[string]any) error
	called  chan struct{}
}

func newMockHASender() *mockHASender {
	return &mockHASender{called: make(chan struct{}, 10)}
}

func (m *mockHASender) CallService(domain, service string, data map[string]any) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.domain = domain
	m.service = service
	m.data = data
	select {
	case m.called <- struct{}{}:
	default:
	}
	if m.callFn != nil {
		return m.callFn(domain, service, data)
	}
	return m.err
}

func (m *mockHASender) waitCall(t *testing.T) {
	t.Helper()
	select {
	case <-m.called:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for CallService")
	}
}

func newTestHAMessenger(notify string, sender *mockHASender) *HAMessenger {
	return &HAMessenger{
		log:    util.NewLogger("ha-test"),
		conn:   sender,
		notify: notify,
	}
}

func newTestHAMessengerWithData(notify string, data map[string]any, sender *mockHASender) *HAMessenger {
	return &HAMessenger{
		log:    util.NewLogger("ha-test"),
		conn:   sender,
		notify: notify,
		data:   data,
	}
}

func TestHAMessengerNotifyPath(t *testing.T) {
	sender := newMockHASender()
	m := newTestHAMessenger("notify.mobile_app_android", sender)

	m.Send("Test Title", "Test Message")
	sender.waitCall(t)

	sender.mu.Lock()
	defer sender.mu.Unlock()

	assert.Equal(t, "notify", sender.domain)
	assert.Equal(t, "mobile_app_android", sender.service)
	assert.Equal(t, "Test Title", sender.data["title"])
	assert.Equal(t, "Test Message", sender.data["message"])
}

func TestHAMessengerNotifyFallback(t *testing.T) {
	// simulate an integration that returns 400 on the legacy call (e.g. Telegram in HA 2024+)
	sender := newMockHASender()
	calls := 0
	sender.callFn = func(domain, service string, data map[string]any) error {
		calls++
		if calls == 1 {
			return request.NewStatusError(&http.Response{
				StatusCode: http.StatusBadRequest,
				Request:    &http.Request{},
			})
		}
		return nil
	}
	m := newTestHAMessenger("notify.telegram_bot", sender)

	m.Send("Test Title", "Test Message")
	sender.waitCall(t)
	sender.waitCall(t)

	sender.mu.Lock()
	defer sender.mu.Unlock()

	assert.Equal(t, 2, calls)
	assert.Equal(t, "notify", sender.domain)
	assert.Equal(t, "send_message", sender.service)
	assert.Equal(t, "notify.telegram_bot", sender.data["entity_id"])
}

func TestHAMessengerNotifyData(t *testing.T) {
	sender := newMockHASender()
	m := newTestHAMessengerWithData("notify.mobile_app_android", map[string]any{
		"ttl":      0,
		"priority": "high",
	}, sender)

	m.Send("Test Title", "Test Message")
	sender.waitCall(t)

	sender.mu.Lock()
	defer sender.mu.Unlock()

	assert.Equal(t, map[string]any{"ttl": 0, "priority": "high"}, sender.data["data"])
}

func TestHAMessengerPersistentNotification(t *testing.T) {
	sender := newMockHASender()
	m := newTestHAMessenger("", sender)

	m.Send("Test Title", "Test Message")
	sender.waitCall(t)

	sender.mu.Lock()
	defer sender.mu.Unlock()

	assert.Equal(t, "persistent_notification", sender.domain)
	assert.Equal(t, "create", sender.service)
	assert.Equal(t, "Test Title", sender.data["title"])
	assert.Equal(t, "Test Message", sender.data["message"])
	assert.Equal(t, "evcc", sender.data["notification_id"])
}

func TestHAMessengerSendErrorLogged(t *testing.T) {
	sender := newMockHASender()
	sender.err = assert.AnError
	m := newTestHAMessenger("", sender)

	assert.NotPanics(t, func() {
		m.Send("Title", "Message")
		sender.waitCall(t)
	})
}

func TestHAMessengerMissingURI(t *testing.T) {
	_, err := NewHAMessengerFromConfig(map[string]any{})
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "uri"))
}

func TestHAMessengerInvalidNotify(t *testing.T) {
	for _, notify := range []string{"notify", "notify."} {
		_, err := NewHAMessengerFromConfig(map[string]any{
			"uri":    "ws://localhost:8123",
			"notify": notify,
		})
		require.Error(t, err, "expected error for notify=%q", notify)
		assert.Contains(t, err.Error(), "domain.service", "notify=%q", notify)
	}
}
