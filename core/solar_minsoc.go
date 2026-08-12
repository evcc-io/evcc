package core

import (
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/server/db/settings"
)

const (
	solarMinSocHorizon   = 72 * time.Hour
	solarMinSocTolerance = 1e-9
)

type solarMinSocPolicy struct {
	api.SolarMinSocStatus
}

func defaultSolarMinSocConfig() api.SolarMinSocConfig {
	return api.SolarMinSocConfig{
		LowThreshold:    5,
		MediumThreshold: 15,
		Vehicles:        make(map[string]api.SolarMinSocVehicleConfig),
	}
}

func validateSolarMinSocConfig(conf api.SolarMinSocConfig) error {
	if conf.LowThreshold < 0 || conf.LowThreshold >= conf.MediumThreshold {
		return fmt.Errorf("thresholds must satisfy 0 <= low < medium")
	}

	for name, values := range conf.Vehicles {
		for state, soc := range map[api.SolarMinSocState]int{
			api.SolarMinSocLow:    values.Low,
			api.SolarMinSocMedium: values.Medium,
			api.SolarMinSocHigh:   values.High,
		} {
			if soc < 0 || soc > 100 {
				return fmt.Errorf("vehicle %s %s soc must be between 0 and 100", name, state)
			}
		}
	}

	return nil
}

func (p *solarMinSocPolicy) configure(conf api.SolarMinSocConfig) error {
	if err := validateSolarMinSocConfig(conf); err != nil {
		return err
	}
	if conf.Vehicles == nil {
		conf.Vehicles = make(map[string]api.SolarMinSocVehicleConfig)
	}
	p.SolarMinSocConfig = conf
	if !conf.Enabled {
		p.Available = false
		p.State = ""
		p.ForecastEnergy = 0
		p.Updated = time.Time{}
	}
	return nil
}

func (p *solarMinSocPolicy) update(rates api.Rates, now time.Time, scale float64) bool {
	if !p.Enabled || !solarMinSocForecastComplete(rates, now) {
		return false
	}

	energy := solarEnergy(rates, now, now.Add(solarMinSocHorizon)) * scale / 1e3
	state := api.SolarMinSocHigh
	if energy+solarMinSocTolerance < p.LowThreshold {
		state = api.SolarMinSocLow
	} else if energy+solarMinSocTolerance < p.MediumThreshold {
		state = api.SolarMinSocMedium
	}

	p.Available = true
	p.ForecastEnergy = energy
	p.State = state
	p.Updated = now
	return true
}

func (p *solarMinSocPolicy) minSoc(vehicle string) int {
	if !p.Enabled || !p.Available {
		return 0
	}

	values := p.Vehicles[vehicle]
	switch p.State {
	case api.SolarMinSocLow:
		return values.Low
	case api.SolarMinSocMedium:
		return values.Medium
	case api.SolarMinSocHigh:
		return values.High
	default:
		return 0
	}
}

func solarMinSocForecastComplete(rates api.Rates, now time.Time) bool {
	if len(rates) < 2 || rates[0].Start.After(now) || rates[len(rates)-1].Start.Before(now.Add(solarMinSocHorizon)) {
		return false
	}

	for i := 1; i < len(rates); i++ {
		if !rates[i].Start.After(rates[i-1].Start) || rates[i].Start.Sub(rates[i-1].Start) > 2*time.Hour {
			return false
		}
	}

	return true
}

func (site *Site) GetSolarMinSoc() api.SolarMinSocStatus {
	vehicles := site.Vehicles().Settings()

	site.RLock()
	status := site.solarMinSoc.SolarMinSocStatus
	site.RUnlock()

	status.AvailableVehicles = make([]api.SolarMinSocVehicle, 0, len(vehicles))
	for _, vehicle := range vehicles {
		status.AvailableVehicles = append(status.AvailableVehicles, api.SolarMinSocVehicle{
			Name:  vehicle.Name(),
			Title: vehicle.Instance().GetTitle(),
		})
	}
	return status
}

func (site *Site) SetSolarMinSoc(conf api.SolarMinSocConfig) error {
	if err := validateSolarMinSocConfig(conf); err != nil {
		return err
	}

	known := make(map[string]bool)
	for _, vehicle := range site.Vehicles().Settings() {
		known[vehicle.Name()] = true
	}
	for name := range conf.Vehicles {
		if !known[name] {
			return fmt.Errorf("unknown vehicle: %s", name)
		}
	}

	site.Lock()
	err := site.solarMinSoc.configure(conf)
	status := site.solarMinSoc.SolarMinSocStatus
	site.Unlock()
	if err != nil {
		return err
	}
	if err := settings.SetJson(keys.SolarMinSoc, conf); err != nil {
		return err
	}

	site.publish(keys.SolarMinSoc, status)
	site.publishVehicles()
	return nil
}

func (site *Site) updateSolarMinSoc() {
	solarTariff := site.GetTariff(api.TariffUsageSolar)
	if solarTariff == nil {
		site.Lock()
		wasAvailable := site.solarMinSoc.Available
		site.solarMinSoc.Available = false
		site.solarMinSoc.State = ""
		site.solarMinSoc.ForecastEnergy = 0
		site.solarMinSoc.Updated = time.Time{}
		status := site.solarMinSoc.SolarMinSocStatus
		site.Unlock()
		if wasAvailable {
			site.publish(keys.SolarMinSoc, status)
			site.publishVehicles()
		}
		return
	}

	rates := currentRates(solarTariff)
	scale := site.effectiveSolarScale()

	site.Lock()
	previous := site.solarMinSoc.State
	updated := site.solarMinSoc.update(rates, time.Now(), scale)
	status := site.solarMinSoc.SolarMinSocStatus
	site.Unlock()

	if updated {
		site.publish(keys.SolarMinSoc, status)
		if previous != status.State {
			site.log.DEBUG.Printf("solar minimum soc: %.1fkWh, state %s", status.ForecastEnergy, status.State)
			site.publishVehicles()
		}
	}
}

func (site *Site) solarMinSocForVehicle(name string) int {
	site.RLock()
	defer site.RUnlock()
	return site.solarMinSoc.minSoc(name)
}
