package messenger

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockHASender struct {
	mu      sync.Mutex
	domain  string
	service string
	data    map[string]any
	err     error
}

func (m *mockHASender) CallService(domain, service string, data map[string]any) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.domain = domain
	m.service = service
	m.data = data
	return m.err
}

func newTestHAMessenger(notify string, sender *mockHASender) *HAMessenger {
	return &HAMessenger{
		log:    util.NewLogger("ha-test"),
		conn:   sender,
		notify: notify,
	}
}

func TestHAMessengerNotifyPath(t *testing.T) {
	sender := &mockHASender{}
	m := newTestHAMessenger("notify.mobile_app_android", sender)

	m.Send("Test Title", "Test Message")
	time.Sleep(50 * time.Millisecond)

	sender.mu.Lock()
	defer sender.mu.Unlock()

	assert.Equal(t, "notify", sender.domain)
	assert.Equal(t, "mobile_app_android", sender.service)
	assert.Equal(t, "Test Title", sender.data["title"])
	assert.Equal(t, "Test Message", sender.data["message"])
}

func TestHAMessengerPersistentNotification(t *testing.T) {
	sender := &mockHASender{}
	m := newTestHAMessenger("", sender)

	m.Send("Test Title", "Test Message")
	time.Sleep(50 * time.Millisecond)

	sender.mu.Lock()
	defer sender.mu.Unlock()

	assert.Equal(t, "persistent_notification", sender.domain)
	assert.Equal(t, "create", sender.service)
	assert.Equal(t, "Test Title", sender.data["title"])
	assert.Equal(t, "Test Message", sender.data["message"])
	assert.Equal(t, "evcc", sender.data["notification_id"])
}

func TestHAMessengerSendErrorLogged(t *testing.T) {
	sender := &mockHASender{err: assert.AnError}
	m := newTestHAMessenger("", sender)

	assert.NotPanics(t, func() {
		m.Send("Title", "Message")
		time.Sleep(50 * time.Millisecond)
	})
}

func TestHAMessengerMissingURI(t *testing.T) {
	_, err := NewHAMessengerFromConfig(map[string]any{})
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "uri"))
}
