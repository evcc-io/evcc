package cardata

import (
	"context"
	"encoding/json"
	"testing"

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

func TestMqttMultiSubscribe(t *testing.T) {
	conn := NewMqttConnector(context.TODO(), util.NewLogger("foo"), t.Name(), nil)

	// loadpoint and config page subscribing to the same vehicle, vin entered in mixed case
	ch1 := conn.Subscribe("wba12345")
	ch2 := conn.Subscribe("WBA12345")

	b, err := json.Marshal(StreamingMessage{Vin: "WBA12345"})
	require.NoError(t, err)

	conn.handler(nil, mockMessage{payload: b})

	for _, ch := range []<-chan StreamingMessage{ch1, ch2} {
		require.Len(t, ch, 1, "message not delivered")
		require.Equal(t, "WBA12345", (<-ch).Vin)
	}

	// unsubscribing one subscriber must not steal the other's channel
	conn.Unsubscribe("WBA12345", ch1)
	conn.handler(nil, mockMessage{payload: b})

	_, ok := <-ch1
	require.False(t, ok, "ch1 not closed")
	require.Len(t, ch2, 1, "message not delivered")

	conn.Unsubscribe("wba12345", ch2)
	require.Empty(t, conn.subscriptions)
}
