package core

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/optimizer"
	"github.com/jinzhu/now"
	"github.com/samber/lo"
)

func (site *Site) optimizerUpdate(pv, battery []measurement) {
	capacity := lo.SumBy(battery, func(m measurement) float64 {
		if m.Capacity == nil {
			return 0
		}
		return *m.Capacity
	})
	charged := lo.SumBy(battery, func(m measurement) float64 {
		if m.Capacity == nil || m.Soc == nil {
			return 0
		}
		return *m.Capacity * (*m.Soc / 100)
	})

	req := optimizer.Request{
		Ems: optimizer.Ems{
			StrompreisEuroProWh:          tariffTo48Slots(site.GetTariff(api.TariffUsageGrid)).Div(1e3),
			EinspeiseverguetungEuroProWh: tariffTo48Slots(site.GetTariff(api.TariffUsageFeedIn)).Div(1e3),
			PvPrognoseWh:                 tariffTo48Slots(site.GetTariff(api.TariffUsageSolar)),
		},
		PvAkku: optimizer.PvAkku{
			DeviceID:   "akku",
			CapacityWh: int(capacity * 1e3),
		},
	}

	if capacity > 0 {
		req.PvAkku.InitialSocPercentage = int(charged / capacity * 100)
	}

	if lps := site.Loadpoints(); len(lps) > 0 {
		lp := lps[0]

		if v := lp.GetVehicle(); v != nil {
			req.EAuto = optimizer.EAuto{
				DeviceID:             "ev",
				CapacityWh:           int(v.Capacity() * 1e3),
				MaxChargePowerW:      int(lp.EffectiveMaxPower()),
				InitialSocPercentage: int(lp.GetSoc() * 100),
			}
		}
	}

	_ = req
}

func tariffTo48Slots(tariff api.Tariff) optimizer.Prediction {
	var res optimizer.Prediction

	if tariff == nil {
		return res // all slots will be 0 (zero value)
	}

	rates, err := tariff.Rates()
	if err != nil {
		return res // all slots will be 0 on error
	}

	// Start at 00:00 today
	start := now.BeginningOfDay()

	for i := range 48 {
		slotTime := start.Add(time.Duration(i) * time.Hour)

		if rate, err := rates.At(slotTime); err == nil {
			res[i] = rate.Value
		}
	}

	return res
}
