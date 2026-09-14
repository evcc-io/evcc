package cardata

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/require"
)

type mockMessage struct {
	mqtt.Message
	payload []byte
}

func (m mockMessage) Topic() string   { return "test" }
func (m mockMessage) Payload() []byte { return m.payload }

func message(t *testing.T, vin string) mockMessage {
	t.Helper()

	b, err := json.Marshal(StreamingMessage{Vin: vin})
	require.NoError(t, err)

	return mockMessage{payload: b}
}

func recv(t *testing.T, ch <-chan StreamingMessage) StreamingMessage {
	t.Helper()

	select {
	case msg := <-ch:
		return msg
	case <-time.After(time.Second):
		t.Fatal("message not delivered")
		return StreamingMessage{}
	}
}

func TestMqttMultiSubscribe(t *testing.T) {
	conn := NewMqttConnector(context.TODO(), util.NewLogger("foo"), t.Name(), nil)

	// loadpoint and config page subscribing to the same vehicle, vin entered in mixed case
	ch1 := conn.Subscribe("wba12345")
	ch2 := conn.Subscribe("WBA12345")

	conn.handler(nil, message(t, "WBA12345"))

	for _, ch := range []<-chan StreamingMessage{ch1, ch2} {
		require.Equal(t, "WBA12345", recv(t, ch).Vin)
	}

	// unsubscribing one subscriber must not steal the other's channel
	conn.Unsubscribe("WBA12345", ch1)
	conn.handler(nil, message(t, "WBA12345"))

	_, ok := <-ch1
	require.False(t, ok, "ch1 not closed")
	require.Equal(t, "WBA12345", recv(t, ch2).Vin)

	conn.Unsubscribe("wba12345", ch2)
	require.Empty(t, conn.subscriptions)
}

func TestMqttUnboundedQueue(t *testing.T) {
	conn := NewMqttConnector(context.TODO(), util.NewLogger("foo"), t.Name(), nil)

	ch := conn.Subscribe("WBA12345")

	// handler must not block on a receiver that is not reading
	for range 100 {
		conn.handler(nil, message(t, "WBA12345"))
	}

	// queue is drained before the channel is closed
	conn.Unsubscribe("WBA12345", ch)

	var count int
	for range ch {
		count++
	}
	require.Equal(t, 100, count)
}
