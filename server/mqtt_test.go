package server

import (
	"math"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

func TestMqttNaNInf(t *testing.T) {
	m := &MQTT{}
	assert.Equal(t, "NaN", m.encode(math.NaN()), "NaN not encoded as string")
	assert.Equal(t, "+Inf", m.encode(math.Inf(0)), "Inf not encoded as string")
}

type measurement struct {
	Power        float64   `json:"power"`
	Energy       float64   `json:"energy,omitempty"`
	Currents     []float64 `json:"currents,omitempty"`
	Soc          *float64  `json:"soc,omitempty"`
	Controllable *bool     `json:"controllable,omitempty"`
}

func TestMqttTypes(t *testing.T) {
	suite.Run(t, new(mqttSuite))
}

type mqttSuite struct {
	suite.Suite
	*MQTT
	topics, payloads []string
}

func (suite *mqttSuite) publish(topic string, retained bool, payload interface{}) {
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

func (suite *mqttSuite) TestGrid() {
	topics := []string{"test/power", "test/energy", "test/currents", "test/soc", "test/controllable"}

	suite.publish("test", false, measurement{})
	suite.Equal(topics, suite.topics, "topics")
	suite.Equal([]string{"0", "", "", "", ""}, suite.payloads, "payloads")

	suite.publish("test", false, measurement{Energy: 1})
	suite.Equal(topics, suite.topics, "topics")
	suite.Equal([]string{"0", "1", "", "", ""}, suite.payloads, "payloads")

	suite.publish("test", false, measurement{Controllable: lo.ToPtr(false)})
	suite.Equal(topics, suite.topics, "topics")
	suite.Equal([]string{"0", "", "", "", "false"}, suite.payloads, "payloads")

	suite.publish("test", false, measurement{Currents: []float64{1, 2, 3}})
	suite.Equal(append(topics, "test/currents/1", "test/currents/2", "test/currents/3"), suite.topics, "topics")
	suite.Equal([]string{"0", "", "3", "", "", "1", "2", "3"}, suite.payloads, "payloads")
}
