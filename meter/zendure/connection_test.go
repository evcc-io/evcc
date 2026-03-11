package zendure

import (
	"testing"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
)

func TestHandler(t *testing.T) {
	conn := &Connection{
		log:    util.NewLogger("test"),
		data:   util.NewMonitor[Data](time.Millisecond),
		serial: "serial",
	}

	{
		// command
		conn.handler(`{"name":"foo"}`)
		_, err := conn.data.Get()
		assert.Error(t, err)
	}
	{
		conn.handler(`{"solarInputPower":113,"sn":"serial"}`)
		res, err := conn.data.Get()
		assert.NoError(t, err)

		assert.Equal(t, Data{SolarInputPower: 113, Sn: "serial"}, res)
	}
	{
		conn.handler(`{"solarInputPower":125,"sn":"serial"}`)
		res, err := conn.data.Get()
		assert.NoError(t, err)

		assert.Equal(t, Data{SolarInputPower: 125, Sn: "serial"}, res)
	}
}

func TestHandlerMinSoc(t *testing.T) {
	conn := &Connection{
		log:    util.NewLogger("test"),
		data:   util.NewMonitor[Data](time.Millisecond),
		serial: "serial",
	}

	conn.handler(`{"minSoc":200,"outputLimit":600,"acMode":2,"sn":"serial"}`)
	res, err := conn.data.Get()
	assert.NoError(t, err)

	assert.Equal(t, 200, res.MinSoc)
	assert.Equal(t, 600, res.OutputLimit)
	assert.Equal(t, 2, res.AcMode)
}

func TestHandlerMerge(t *testing.T) {
	conn := &Connection{
		log:    util.NewLogger("test"),
		data:   util.NewMonitor[Data](time.Millisecond),
		serial: "serial",
	}

	// first message with some fields
	conn.handler(`{"electricLevel":75,"outputLimit":600,"sn":"serial"}`)
	res, err := conn.data.Get()
	assert.NoError(t, err)
	assert.Equal(t, 75, res.ElectricLevel)
	assert.Equal(t, 600, res.OutputLimit)

	// second message merges minSoc without losing outputLimit
	conn.handler(`{"minSoc":100,"electricLevel":74,"sn":"serial"}`)
	res, err = conn.data.Get()
	assert.NoError(t, err)
	assert.Equal(t, 100, res.MinSoc)
	assert.Equal(t, 74, res.ElectricLevel)
	assert.Equal(t, 600, res.OutputLimit) // preserved from previous message
}
