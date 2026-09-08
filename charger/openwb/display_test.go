package openwb

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type displayTestClient struct {
	values    map[string]string
	listeners map[string]func(string)
	writes    []string
	retained  []bool
	onWrite   func(string)
}

func (client *displayTestClient) Listen(topic string, callback func(string)) error {
	client.listeners[topic] = callback
	if payload, ok := client.values[topic]; ok {
		callback(payload)
	}
	return nil
}

func (client *displayTestClient) Publish(topic string, retained bool, payload string) {
	client.writes = append(client.writes, topic+"="+payload)
	client.retained = append(client.retained, retained)
	if client.onWrite != nil {
		client.onWrite(topic)
		return
	}
	client.listeners[strings.Replace(topic, "openWB/set/", "openWB/", 1)](payload)
}

func TestConfigureDisplay(t *testing.T) {
	const uri = "http://192.168.1.10:7070"
	const desired = `{"type":"url_display","configuration":{"url":"` + uri + `"}}`
	for _, test := range []struct {
		name   string
		topic  string
		value  string
		writes []string
		fail   bool
	}{
		{name: "configure", writes: []string{"openWB/set/optional/int_display/theme=" + desired, `openWB/set/general/extern_display_mode="local"`}},
		{name: "primary", topic: displaySecondaryTopic, value: "false"},
		{name: "inactive", topic: displayActiveTopic, value: "false"},
		{name: "unsupported", topic: displayThemesTopic, value: `[{"value":"cards"}]`},
		{name: "invalid secondary", topic: displaySecondaryTopic, value: "invalid", fail: true},
		{name: "missing secondary", topic: displaySecondaryTopic, value: "", fail: true},
		{name: "invalid themes", topic: displayThemesTopic, value: "invalid", fail: true},
		{name: "invalid active", topic: displayActiveTopic, value: `"true"`, fail: true},
		{name: "null secondary", topic: displaySecondaryTopic, value: "null"},
		{name: "null theme", topic: displayThemeTopic, value: "null", fail: true},
		{name: "invalid mode", topic: displayModeTopic, value: "{}", fail: true},
		{name: "theme matches", topic: displayThemeTopic, value: desired, writes: []string{`openWB/set/general/extern_display_mode="local"`}},
		{name: "mode matches", topic: displayModeTopic, value: `"local"`, writes: []string{"openWB/set/optional/int_display/theme=" + desired}},
	} {
		t.Run(test.name, func(t *testing.T) {
			client := newDisplayTestClient()
			if test.topic != "" {
				client.values[test.topic] = test.value
			}
			timeout := 5 * time.Second
			if test.name == "missing secondary" {
				timeout = 100 * time.Millisecond
			}
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()
			err := ConfigureDisplay(ctx, util.NewLogger("test"), client, uri, "")
			if test.fail {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, test.writes, client.writes)
			for _, retained := range client.retained {
				assert.False(t, retained)
			}
		})
	}
}

func newDisplayTestClient() *displayTestClient {
	return &displayTestClient{
		values: map[string]string{
			displaySecondaryTopic: "true",
			displayActiveTopic:    "true",
			displayThemesTopic:    `[{"value":"url_display"}]`,
			displayThemeTopic:     `{"type":"cards","configuration":{}}`,
			displayModeTopic:      `"primary"`,
		},
		listeners: make(map[string]func(string)),
	}
}

func TestConfigureDisplayUnchanged(t *testing.T) {
	client := newDisplayTestClient()
	client.values[displayThemeTopic] = `{"type":"url_display","configuration":{"url":"http://evcc:7070"},"name":"URL Display","official":false}`
	client.values[displayModeTopic] = `"local"`
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	require.NoError(t, ConfigureDisplay(ctx, util.NewLogger("test"), client, "http://evcc:7070", ""))
	assert.Empty(t, client.writes)
}

func TestConfigureDisplayQuery(t *testing.T) {
	for _, test := range []struct {
		query, url string
	}{
		{"lp=1", "http://evcc:7070/?lp=1"},
		{"?lp=1", "http://evcc:7070/?lp=1"},
		{"/?lp=1", "http://evcc:7070/?lp=1"},
		{"#/?lp=1", "http://evcc:7070/?lp=1"},
	} {
		t.Run(test.query, func(t *testing.T) {
			client := newDisplayTestClient()
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			require.NoError(t, ConfigureDisplay(ctx, util.NewLogger("test"), client, "http://evcc:7070", test.query))
			require.Len(t, client.writes, 2)
			assert.Equal(t, `openWB/set/optional/int_display/theme={"type":"url_display","configuration":{"url":"`+test.url+`"}}`, client.writes[0])
		})
	}
}

func TestConfigureDisplayRejected(t *testing.T) {
	client := newDisplayTestClient()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	client.onWrite = func(topic string) {}
	require.ErrorIs(t, ConfigureDisplay(ctx, util.NewLogger("test"), client, "http://evcc:7070", ""), context.DeadlineExceeded)
	require.Len(t, client.writes, 1)
	assert.True(t, strings.HasPrefix(client.writes[0], "openWB/set/optional/int_display/theme="))
}

func TestConfigureDisplaySecondaryChanges(t *testing.T) {
	client := newDisplayTestClient()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	client.onWrite = func(topic string) {
		client.listeners[displaySecondaryTopic]("false")
		client.listeners[displayThemeTopic](`{"type":"url_display","configuration":{"url":"http://evcc:7070"}}`)
	}
	require.ErrorContains(t, ConfigureDisplay(ctx, util.NewLogger("test"), client, "http://evcc:7070", ""), "is not true")
	require.Len(t, client.writes, 1)
}

func TestConfigureDisplayInvalidURL(t *testing.T) {
	for _, uri := range []string{"", "file:///tmp/evcc", "http://evcc:0", "http://user:password@evcc"} {
		t.Run(uri, func(t *testing.T) {
			client := newDisplayTestClient()
			require.Error(t, ConfigureDisplay(context.Background(), util.NewLogger("test"), client, uri, ""))
			assert.Empty(t, client.listeners)
			assert.Empty(t, client.writes)
		})
	}
}

func TestConfigureDisplayCanceled(t *testing.T) {
	client := newDisplayTestClient()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	require.ErrorIs(t, ConfigureDisplay(ctx, util.NewLogger("test"), client, "http://evcc:7070", ""), context.Canceled)
	assert.Empty(t, client.writes)
}
