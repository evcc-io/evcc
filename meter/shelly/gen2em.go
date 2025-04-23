package shelly

import (
	"time"

	"github.com/evcc-io/evcc/util"
)

var _ Generation = (*gen2EM)(nil)

type gen2EM struct {
	conn   *gen2Conn
	status func() (Gen2EMStatus, error)
	data   func() (Gen2EMData, error)
}

type Gen2EMStatus struct {
	TotalActPower float64 `json:"total_act_power"`
	ACurrent      float64 `json:"a_current"`
	BCurrent      float64 `json:"b_current"`
	CCurrent      float64 `json:"c_current"`
	AVoltage      float64 `json:"a_voltage"`
	BVoltage      float64 `json:"b_voltage"`
	CVoltage      float64 `json:"c_voltage"`
	AActPower     float64 `json:"a_act_power"`
	BActPower     float64 `json:"b_act_power"`
	CActPower     float64 `json:"c_act_power"`
}

type Gen2EMData struct {
	TotalAct    float64 `json:"total_act"`
	TotalActRet float64 `json:"total_act_ret"`
}

func newGen2EM(conn *gen2Conn, cache time.Duration) *gen2EM {
	c := &gen2EM{
		conn: conn,
	}

	c.status = util.Cached(apiCall[Gen2EMStatus](c.conn, "EM.GetStatus"), cache)
	c.data = util.Cached(apiCall[Gen2EMData](c.conn, "EMData.GetStatus"), cache)

	return c
}

// CurrentPower implements the api.Meter interface
func (c *gen2EM) CurrentPower() (float64, error) {
	res, err := c.status()
	return res.TotalActPower, err
}

// TotalEnergy implements the api.Meter interface
func (c *gen2EM) TotalEnergy() (float64, error) {
	res, err := c.data()
	return res.TotalAct / 1000, err
}

// Currents implements the api.PhaseCurrents interface
func (c *gen2EM) Currents() (float64, float64, float64, error) {
	res, err := c.status()
	return res.ACurrent, res.BCurrent, res.CCurrent, err
}

// Voltages implements the api.PhaseVoltages interface
func (c *gen2EM) Voltages() (float64, float64, float64, error) {
	res, err := c.status()
	return res.AVoltage, res.BVoltage, res.CVoltage, err
}

// Powers implements the api.PhasePowers interface
func (c *gen2EM) Powers() (float64, float64, float64, error) {
	res, err := c.status()
	return res.AActPower, res.BActPower, res.CActPower, err
}
