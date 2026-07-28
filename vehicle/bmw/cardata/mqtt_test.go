package cardata

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/require"
)

// mockMessage implements mqtt.Message for testing
type mockMessage struct {
	payload []byte
}

func (m mockMessage) Duplicate() bool { return false }
func (m mockMessage) Qos() byte       { return 0 }
func (m mockMessage) Retained() bool  { return false }
func (m mockMessage) Topic() string   { return "test" }
func (m mockMessage) MessageID() uint16 { return 0 }
func (m mockMessage) Payload() []byte { return m.payload }
func (m mockMessage) Ack()            {}

func TestMqttMultiSubscribe(t *testing.T) {
	log := util.NewLogger("test")

	// Create MqttConnector (using empty gcid)
	mqttConn := NewMqttConnector(nil, log, "test-client-id", nil)

	// Simulate multiple vehicles (e.g. loadpoint vs config page) subscribing to the same VIN
	ch1 := mqttConn.Subscribe("WBA12345")
	ch2 := mqttConn.Subscribe("WBA12345")

	// Create a dummy JSON payload matching the expected format
	msgData := StreamingMessage{
		Vin: "WBA12345",
		Data: map[string]StreamingData{
			"vehicle.powertrain.electric.battery.stateOfCharge.displayed": {
				TimeStamp: time.Now(),
				Value:     50.0,
			},
		},
	}
	b, err := json.Marshal(msgData)
	require.NoError(t, err)

	// Simulate MQTT incoming message
	mqttConn.handler(nil, mockMessage{payload: b})

	// Both channels should receive the message without blocking each other
	select {
	case res1 := <-ch1:
		require.Equal(t, "WBA12345", res1.Vin)
	case <-time.After(time.Second):
		t.Fatal("ch1 did not receive message (stolen subscription bug)")
	}

	select {
	case res2 := <-ch2:
		require.Equal(t, "WBA12345", res2.Vin)
	case <-time.After(time.Second):
		t.Fatal("ch2 did not receive message (stolen subscription bug)")
	}

	// Test Unsubscribe removes only one channel
	mqttConn.Unsubscribe("WBA12345", ch1)

	// Send another message
	mqttConn.handler(nil, mockMessage{payload: b})

	// ch1 should not receive it, but ch2 should
	select {
	case _, ok := <-ch1:
		if ok {
			t.Fatal("ch1 should not receive message after unsubscribe")
		}
	default:
	}

	select {
	case <-ch2:
		// success
	case <-time.After(time.Second):
		t.Fatal("ch2 should still receive message")
	}
}
