package server

import (
	"encoding/json"
	"math"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/evcc-io/evcc/core/types"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

func TestMqttNaNInf(t *testing.T) {
	m := &MQTT{}
	assert.Equal(t, "NaN", m.encode(math.NaN()), "NaN not encoded as string")
	assert.Equal(t, "+Inf", m.encode(math.Inf(0)), "Inf not encoded as string")
}

func TestMqttTypes(t *testing.T) {
	suite.Run(t, new(mqttSuite))
}

type mqttSuite struct {
	suite.Suite
	*MQTT
	topics, payloads []string
}

func (suite *mqttSuite) publish(topic string, retained bool, payload any) {
	suite.MQTT.publish(topic, retained, payload)
}

func (suite *mqttSuite) publisher(topic string, retained bool, payload string) {
	if i := slices.Index(suite.topics, topic); i >= 0 {
		suite.topics[i] = topic
		suite.payloads[i] = payload
	} else {
		suite.topics = append(suite.topics, topic)
		suite.payloads = append(suite.payloads, payload)
	}
}

func (suite *mqttSuite) SetupSuite() {
	suite.MQTT = &MQTT{
		publisher: suite.publisher,
	}
}

func (suite *mqttSuite) SetupTest() {
	suite.topics = suite.topics[:0]
	suite.payloads = suite.payloads[:0]
}

func (suite *mqttSuite) TestTime() {
	now := time.Now()
	suite.publish("test", false, now)
	suite.Require().Len(suite.topics, 1)
	suite.Equal(strconv.FormatInt(now.Unix(), 10), suite.payloads[0], "time not encoded as unix timestamp")
}

func (suite *mqttSuite) TestBool() {
	suite.publish("test", false, false)
	suite.Require().Len(suite.topics, 1)
	suite.Equal("false", suite.payloads[0])
}

func (suite *mqttSuite) TestStruct() {
	suite.publish("test", false, struct {
		Foo string
	}{
		Foo: "bar",
	})
	suite.Equal([]string{"test/foo"}, suite.topics, "topics")
	suite.Equal([]string{"bar"}, suite.payloads, "payloads")
}

func (suite *mqttSuite) TestStructPointer() {
	i := 1
	suite.publish("test", false, struct {
		Foo, Bar *int
	}{
		Foo: &i,
		Bar: nil,
	})
	suite.Equal([]string{"test/foo", "test/bar"}, suite.topics, "topics")
	suite.Equal([]string{"1", ""}, suite.payloads, "payloads")
}

func (suite *mqttSuite) TestSlice() {
	slice := []int{10, 20}
	suite.publish("test", false, slice)
	suite.Require().Len(suite.topics, 3)
	suite.Equal([]string{"test", "test/1", "test/2"}, suite.topics, "topics")
	suite.Equal([]string{"2", "10", "20"}, suite.payloads, "payloads")
}

type jsonSeries [][]float64

func (s jsonSeries) MarshalBytes() ([]byte, error) {
	return json.Marshal(s)
}

// a sharded struct publishes one message per key, BytesMarshaler fields as json
func (suite *mqttSuite) TestSharderBytesMarshaler() {
	p := struct {
		Foo jsonSeries `json:"foo,omitempty"`
		Bar jsonSeries `json:"bar,omitempty"`
	}{
		Foo: jsonSeries{{1, 2, 3}, {4, 5, 6}},
	}

	suite.publish("test", false, util.NewSharder("test", p))
	suite.Equal([]string{"test/foo", "test/bar"}, suite.topics, "topics")
	suite.Equal([]string{"[[1,2,3],[4,5,6]]", ""}, suite.payloads, "payloads")
}

func (suite *mqttSuite) TestNilInterface() {
	var ptr *time.Time
	suite.publish("test", false, ptr)
	suite.Equal([]string{"test"}, suite.topics, "topics")
	suite.Equal([]string{""}, suite.payloads, "payloads")
}

// published returns the last payload per topic
func (suite *mqttSuite) published() map[string]string {
	res := make(map[string]string, len(suite.topics))
	for i, topic := range suite.topics {
		res[topic] = suite.payloads[i]
	}
	return res
}

// TestMeasurement checks the fields under test only, so adding a field does not break it
func (suite *mqttSuite) TestMeasurement() {
	suite.publish("test", false, types.Measurement{})
	for topic, payload := range suite.published() {
		if topic != "test/power" {
			suite.Empty(payload, "unset field must clear retained topic %s", topic)
		}
	}
	suite.Equal("0", suite.published()["test/power"])

	suite.publish("test", false, types.Measurement{Energy: new(1.0)})
	suite.Equal("1", suite.published()["test/energy"])

	suite.publish("test", false, types.Measurement{Controllable: new(false)})
	suite.Equal("false", suite.published()["test/controllable"])
	suite.Empty(suite.published()["test/energy"], "energy cleared")

	suite.publish("test", false, types.Measurement{Currents: []float64{1, 2, 3}})
	suite.Subset(suite.published(), map[string]string{
		"test/currents": "3", "test/currents/1": "1", "test/currents/2": "2", "test/currents/3": "3",
	})

	suite.publish("test", false, types.Measurement{Modes: []string{"normal", "discharge"}})
	suite.Subset(suite.published(), map[string]string{
		"test/modes": "2", "test/modes/1": "normal", "test/modes/2": "discharge",
	})
}

func (suite *mqttSuite) TestBatteryState() {
	suite.publish("test", false, types.BatteryState{
		Power: 2,
		Soc:   20.0,
		Devices: []types.Measurement{{
			Power: 1,
			Soc:   new(10.0),
		}},
	})

	suite.Subset(suite.published(), map[string]string{
		"test/power": "2", "test/soc": "20", "test/forecast": "",
		"test/devices": "1", "test/devices/1/power": "1", "test/devices/1/soc": "10",
	})
}

func (suite *mqttSuite) TestMapKeys() {
	suite.publish("test", false, map[string]int{
		"HomeAssistant (homeassistant.internal)": 1,
	})
	suite.Equal([]string{"test/HomeAssistant_(homeassistant.internal)"}, suite.topics, "topics")
	suite.Equal([]string{"1"}, suite.payloads, "payloads")
}

func TestMqttTopicLevel(t *testing.T) {
	assert.Equal(t, "foo", mqttTopicLevel("foo"))
	assert.Equal(t, "Tibber_(VW_ID.3,_Cupra)", mqttTopicLevel("Tibber (VW ID.3, Cupra)"))
	assert.Equal(t, "a_b_c_d", mqttTopicLevel("a+b#c/d"))
}
