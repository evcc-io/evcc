package server

import (
	"math"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMqttNaNInf(t *testing.T) {
	m := &MQTT{}
	assert.Equal(t, "NaN", m.encode(math.NaN()), "NaN not encoded as string")
	assert.Equal(t, "+Inf", m.encode(math.Inf(0)), "Inf not encoded as string")
}

func TestPublishTypes(t *testing.T) {
	var topics, payloads []string

	reset := func() {
		topics = topics[:0]
		payloads = payloads[:0]
	}

	m := &MQTT{
		publisher: func(topic string, retained bool, payload string) {
			topics = append(topics, topic)
			payloads = append(payloads, payload)
		},
	}

	now := time.Now()
	m.publish("test", false, now)
	require.Len(t, topics, 1)
	assert.Equal(t, strconv.FormatInt(now.Unix(), 10), payloads[0], "time not encoded as unix timestamp")
	reset()

	m.publish("test", false, struct {
		Foo string
	}{
		Foo: "bar",
	})
	require.Len(t, topics, 1)
	assert.Equal(t, `test/foo`, topics[0], "struct mismatch")
	assert.Equal(t, `bar`, payloads[0], "struct mismatch")
	reset()

	slice := []int{10, 20}
	m.publish("test", false, slice)
	require.Len(t, topics, 3)
	assert.Equal(t, []string{`test`, `test/1`, `test/2`}, topics, "slice mismatch")
	assert.Equal(t, []string{`2`, `10`, `20`}, payloads, "slice mismatch")
	reset()
}
