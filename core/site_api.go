package core

import (
	"errors"

	"github.com/mark-sch/evcc/api"
)

// SiteAPI is the external site API
type SiteAPI interface {
	Healthy() bool
	LoadPoints() []LoadPointAPI
	GetResidualPower() float64
	SetResidualPower(float64) error
	GetPrioritySoC() float64
	SetPrioritySoC(float64) error
	GetMinSoC() int
	SetMinSoC(int) error
}


// GetPrioritySoC returns the PrioritySoC
func (site *Site) GetPrioritySoC() float64 {
	site.Lock()
	defer site.Unlock()
	return site.PrioritySoC
}


// SetPrioritySoC sets the PrioritySoC
func (site *Site) SetPrioritySoC(soc float64) error {
	site.Lock()
	defer site.Unlock()

	if _, ok := site.batteryMeter.(api.Battery); !ok {
		return errors.New("battery not configured")
	}

	//force immediate reaction to mode change
	site.count = 30;

	site.log.INFO.Println("set global priority soc:", soc)
	site.PrioritySoC = soc
	site.publish("prioritySoC", site.PrioritySoC)

	return nil
}


// GetResidualPower
func (site *Site) GetResidualPower() float64 {
	site.Lock()
	defer site.Unlock()
	return site.ResidualPower
}


// SetResidualPower
func (site *Site) SetResidualPower(power float64) error {
	site.Lock()
	defer site.Unlock()
	
	site.log.INFO.Println("set residual power:", power)
	site.ResidualPower = power
	site.publish("residualPower", power)

	return nil
}


// GetMinSoC gets loadpoint charge minimum soc
func (site *Site) GetMinSoC() int {
	return site.loadpoints[0].GetMinSoC()
}

// SetMinSoC sets loadpoint charge minimum soc
func (site *Site) SetMinSoC(soc int) error {
	site.log.INFO.Println("set global min soc:", soc)
	for _, lp := range site.loadpoints {
		if err := lp.SetMinSoC(soc); err != nil {
			return err
		}
	}
	return nil
}