package shelly

import (
	"strconv"
	"time"

	"github.com/evcc-io/evcc/util"
)

var _ Generation = (*gen2Switch)(nil)

type gen2Switch struct {
	conn    *gen2Conn
	channel int
	status  util.Cacheable[Gen2SwitchStatus]
}

type Gen2SwitchStatus struct {
	Output  bool
	Apower  float64
	Voltage float64
	Current float64
	Aenergy struct {
		Total float64
	}
	Ret_Aenergy struct {
		Total float64
	}
}

// gen2InitApi initializes the connection to the shelly gen2+ api and sets up the cached gen2SwitchStatus, gen2EM1Status and gen2EMStatus
func newGen2Switch(conn *gen2Conn, cache time.Duration) *gen2Switch {
	c := &gen2Switch{
		conn: conn,
	}

	c.status = util.ResettableCached(apiCall[Gen2SwitchStatus](c, "Switch.GetStatus"), cache)

	return c
}

// CurrentPower implements the api.Meter interface
func (c *gen2Switch) CurrentPower() (float64, error) {
	res, err := c.status.Get()
	return res.Apower, err
}

// Gen2Enabled implements the Gen2 api.Charger interface
func (c *gen2Switch) Enabled() (bool, error) {
	res, err := c.status.Get()
	return res.Output, err
}

// Gen2Enable implements the api.Charger interface
func (c *gen2Switch) Enable(enable bool) error {
	var res Gen2SwitchStatus
	c.status.Reset()
	return c.execCmd("Switch.Set?id="+strconv.Itoa(c.conn.channel), enable, &res)
}

// TotalEnergy implements the api.Meter interface
func (c *gen2Switch) TotalEnergy() (float64, error) {
	res, err := c.status.Get()
	return res.Aenergy.Total / 1000, err
}
