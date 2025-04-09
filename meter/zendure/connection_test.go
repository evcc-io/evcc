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
		data:   util.NewMonitor[Data](time.Minute),
		serial: "serial",
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
