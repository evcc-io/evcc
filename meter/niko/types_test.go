package niko

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/server/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const listResponse = `{"Method":"devices.list","Params":[{"Devices":[
	{"Uuid":"a1","Type":"smartplug","Model":"naso","Name":"Garage","Properties":[{"Status":"On"},{"ElectricalPower":"1234.5"}]},
	{"Uuid":"b2","Type":"action","Model":"light","Name":"Kitchen","Properties":[{"Status":"Off"}]},
	{"Uuid":"c3","Type":"smartplug","Model":"generic","Name":"","Properties":[{"Status":"Off"}]}
]}]}`

const statusEvent = `{"Method":"devices.status","Params":[{"Devices":[{"Uuid":"a1","Properties":[{"Status":"Off"}]}]}]}`

func TestParseList(t *testing.T) {
	var msg Message
	require.NoError(t, json.Unmarshal([]byte(listResponse), &msg))

	assert.Equal(t, MethodList, msg.Method)
	devices := msg.Devices()
	require.Len(t, devices, 3)

	status, ok := devices[0].Status()
	assert.True(t, ok)
	assert.True(t, status)

	power, ok := devices[0].Power()
	assert.True(t, ok)
	assert.Equal(t, 1234.5, power)

	_, ok = devices[2].Power()
	assert.False(t, ok)

	opts := smartPlugOptions(devices)
	require.Len(t, opts, 2)
	assert.Equal(t, service.Option{Value: "c3", Label: "c3", Hint: "Off"}, opts[1])
	assert.Equal(t, service.Option{Value: "a1", Label: "Garage", Hint: "On"}, opts[0])
}

func TestUpdate(t *testing.T) {
	c := &Connection{uuid: "a1"}

	var msg Message
	require.NoError(t, json.Unmarshal([]byte(listResponse), &msg))
	assert.True(t, c.update(msg))
	assert.True(t, c.status)
	assert.Equal(t, 1234.5, c.power)

	require.NoError(t, json.Unmarshal([]byte(statusEvent), &msg))
	assert.True(t, c.update(msg))
	assert.False(t, c.status)
	assert.Equal(t, 1234.5, c.power, "power retained when not reported")

	_, err := c.Enabled()
	assert.NoError(t, err)

	c.updated = time.Now().Add(-2 * maxAge)
	_, err = c.CurrentPower()
	assert.ErrorIs(t, err, api.ErrOutdated)

	c.uuid = "x"
	assert.False(t, c.update(msg))
}

func TestInvalidPower(t *testing.T) {
	for _, v := range []string{"", "abc", "NaN", "Inf"} {
		_, ok := Device{Properties: []Property{{"ElectricalPower": v}}}.Power()
		assert.False(t, ok, v)
	}
}

func TestControlRequest(t *testing.T) {
	assert.JSONEq(t,
		`{"Method":"devices.control","Params":[{"Devices":[{"Uuid":"a1","Properties":[{"Status":"On"}]}]}]}`,
		ControlRequest("a1", "Status", "On"),
	)
	assert.JSONEq(t, `{"Method":"devices.list"}`, ListRequest())
}
