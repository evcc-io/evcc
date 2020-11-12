package core

import "github.com/andig/evcc/api"

// SiteAPI is the external site API
type SiteAPI interface {
	Healthy() bool
	LoadPoints() []LoadPointAPI
	LoadPointSettingsAPI
}

// LoadPoints returns the array of associated loadpoints
func (site *Site) LoadPoints() []LoadPointAPI {
	res := make([]LoadPointAPI, len(site.loadpoints))
	for id, lp := range site.loadpoints {
		res[id] = lp
	}
	return res
}

// GetMode gets loadpoint charge mode
func (site *Site) GetMode() api.ChargeMode {
	return site.loadpoints[0].GetMode()
}

// SetMode sets loadpoint charge mode
func (site *Site) SetMode(mode api.ChargeMode) {
	site.log.INFO.Printf("set global charge mode: %s", string(mode))
	for _, lp := range site.loadpoints {
		lp.SetMode(mode)
	}
}

// GetTargetSoC gets loadpoint charge target soc
func (site *Site) GetTargetSoC() int {
	return site.loadpoints[0].GetTargetSoC()
}

// SetTargetSoC sets loadpoint charge target soc
func (site *Site) SetTargetSoC(soc int) error {
	site.log.INFO.Println("set global target soc:", soc)
	for _, lp := range site.loadpoints {
		if err := lp.SetTargetSoC(soc); err != nil {
			return err
		}
	}
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
