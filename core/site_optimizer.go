package core

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	evopt "github.com/andig/evopt/client"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/jinzhu/now"
	"github.com/samber/lo"
)

var (
	eta          = float32(0.9)       // efficiency of the battery charging/discharging
	batteryPower = float32(6000)      // power of the battery in W
	pa           = float32(0.3 / 1e3) // Value per Wh at end of time horizon
	baseLoad     = float32(300)       // base load in W

	updated time.Time
)

func init() {
	os.Setenv("EVOPT_URI", "http://localhost:7050")
}

func (site *Site) optimizerUpdate(battery []measurement) error {
	defer func() {
		updated = time.Now()
	}()

	if time.Since(updated) < 5*time.Minute {
		return nil
	}

	grid := currentSlots(site.GetTariff(api.TariffUsageGrid))
	feedIn := currentSlots(site.GetTariff(api.TariffUsageFeedIn))
	solar := currentSlots(site.GetTariff(api.TariffUsageSolar))

	minLen := lo.Min([]int{len(grid), len(feedIn), len(solar)})
	if minLen < 8 {
		return fmt.Errorf("not enough slots for optimization: %d", minLen)
	}

	gt := lo.RepeatBy(minLen, func(_ int) float32 {
		return baseLoad
	})

	req := evopt.OptimizationInput{
		EtaC: &eta,
		EtaD: &eta,
		TimeSeries: evopt.TimeSeries{
			Gt: gt,
			PN: maxValues(grid, 1e3, minLen),
			PE: maxValues(feedIn, 1e3, minLen),
			Ft: maxValues(solar, 1, minLen),
		},
	}

	for _, lp := range site.Loadpoints() {
		if v := lp.GetVehicle(); v != nil {
			req.Batteries = append(req.Batteries, evopt.BatteryConfig{
				CMax:     float32(lp.EffectiveMaxPower()),
				DMax:     0,
				PA:       pa,
				SMin:     float32(0),
				SMax:     float32(v.Capacity() * 1e3),              // Wh
				SInitial: float32(v.Capacity() * lp.GetSoc() * 10), // Wh
			})
		}
	}

	for _, b := range battery {
		// || !b.Controllable()
		if b.Capacity == nil || b.Soc == nil {
			continue
		}

		req.Batteries = append(req.Batteries, evopt.BatteryConfig{
			CMax:     batteryPower,
			DMax:     batteryPower,
			PA:       pa,
			SMin:     float32(0),
			SMax:     float32(*b.Capacity * 1e3),         // Wh
			SInitial: float32(*b.Capacity * *b.Soc * 10), // Wh
		})
	}

	uri := os.Getenv("EVOPT_URI")
	if uri == "" {
		return nil
	}

	apiClient, err := evopt.NewClientWithResponses(uri, evopt.WithHTTPClient(
		request.NewClient(util.NewLogger("evopt")),
	))
	if err != nil {
		return err
	}

	resp, err := apiClient.PostOptimizeChargeScheduleWithResponse(context.TODO(), req)
	if err != nil {
		return err
	}

	if resp.StatusCode() == http.StatusInternalServerError && resp.JSON500.Message != nil {
		return errors.New(*resp.JSON500.Message)
	}

	if resp.StatusCode() == http.StatusBadRequest && resp.JSON400.Message != nil {
		return errors.New(*resp.JSON400.Message)
	}

	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("invalid status: %d", resp.StatusCode())
	}

	site.publish("evopt", struct {
		Req  evopt.OptimizationInput  `json:"req"`
		Resp evopt.OptimizationResult `json:"resp"`
	}{
		Req:  req,
		Resp: *resp.JSON200,
	})

	return nil
}

func currentSlots(tariff api.Tariff) []api.Rate {
	if tariff == nil {
		return nil
	}

	rates, err := tariff.Rates()
	if err != nil {
		return nil
	}

	now := now.BeginningOfHour()
	return lo.Filter(rates, func(slot api.Rate, _ int) bool {
		return !slot.End.Before(now) // filter past slots
	})
}

func maxValues(rates []api.Rate, div float64, maxLen int) []float32 {
	res := make([]float32, 0, maxLen)

	for _, slot := range rates {
		res = append(res, float32(slot.Value/div))
		if len(res) >= maxLen {
			break
		}
	}

	return res
}
