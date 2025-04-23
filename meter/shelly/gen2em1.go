package shelly

import (
	"time"

	"github.com/evcc-io/evcc/util"
)

var _ Generation = (*gen2EM1)(nil)

type gen2EM1 struct {
	conn   *gen2Conn
	status func() (Gen2EM1Status, error)
	data   func() (Gen2EM1Data, error)
}

type Gen2EM1Status struct {
	Current  float64 `json:"current"`
	Voltage  float64 `json:"voltage"`
	ActPower float64 `json:"act_power"`
}

type Gen2EM1Data struct {
	TotalActEnergy    float64 `json:"total_act_energy"`
	TotalActRetEnergy float64 `json:"total_act_ret_energy"`
}

func newGen2EM1(conn *gen2Conn, cache time.Duration) *gen2EM1 {
	c := &gen2EM1{
		conn: conn,
	}

	c.status = util.Cached(apiCall[Gen2EM1Status](c.conn, "EM1.GetStatus"), cache)
	c.data = util.Cached(apiCall[Gen2EM1Data](c.conn, "EM1Data.GetStatus"), cache)

	return c
}

// CurrentPower implements the api.Meter interface
func (c *gen2EM1) CurrentPower() (float64, error) {
	res, err := c.status()
	return res.ActPower, err
}

// TotalEnergy implements the api.Meter interface
func (c *gen2EM1) TotalEnergy() (float64, error) {
	res, err := c.data()
	return res.TotalActEnergy / 1000, err
}
