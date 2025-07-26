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
	"github.com/evcc-io/evcc/util/sponsor"
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

// func init() {
// 	os.Setenv("EVOPT_URI", "http://localhost:7050")
// }

func (site *Site) optimizerUpdate(battery []measurement) error {
	defer func() {
		updated = time.Now()
	}()

	if time.Since(updated) < 5*time.Minute {
		return nil
	}

	log := util.NewLogger("evopt")

	grid := currentSlots(site.GetTariff(api.TariffUsageGrid))
	feedIn := currentSlots(site.GetTariff(api.TariffUsageFeedIn))
	solar := currentSlots(site.GetTariff(api.TariffUsageSolar))

	minLen := lo.Min([]int{len(grid), len(feedIn), len(solar)})
	if minLen < 8 {
		return fmt.Errorf("not enough slots for optimization: %d", minLen)
	}

	log.DEBUG.Printf("optimizing %d slots until %v: grid=%d, feedIn=%d, solar=%d",
		minLen,
		grid[minLen-1].End.Local(),
		len(grid), len(feedIn), len(solar),
	)

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
				CMin:     float32(lp.EffectiveMinPower()),
				CMax:     float32(lp.EffectiveMaxPower()),
				DMax:     0,
				SMin:     0,
				SMax:     float32(v.Capacity() * 1e3),              // Wh
				SInitial: float32(v.Capacity() * lp.GetSoc() * 10), // Wh
				PA:       pa,
			})
		}
	}

	for _, b := range battery {
		// || !b.Controllable()
		if b.Capacity == nil || b.Soc == nil {
			continue
		}

		req.Batteries = append(req.Batteries, evopt.BatteryConfig{
			CMin:     0,
			CMax:     batteryPower,
			DMax:     batteryPower,
			SMin:     0,
			SMax:     float32(*b.Capacity * 1e3),         // Wh
			SInitial: float32(*b.Capacity * *b.Soc * 10), // Wh
			PA:       pa,
		})
	}

	uri := os.Getenv("EVOPT_URI")
	if uri == "" {
		return nil
	}

	apiClient, err := evopt.NewClientWithResponses(uri, evopt.WithHTTPClient(
		request.NewClient(log),
	))
	if err != nil {
		return err
	}

	resp, err := apiClient.PostOptimizeChargeScheduleWithResponse(context.TODO(), req, func(_ context.Context, req *http.Request) error {
		if token := sponsor.Token; token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		return nil
	})
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
		Req evopt.OptimizationInput  `json:"req"`
		Res evopt.OptimizationResult `json:"res"`
	}{
		Req: req,
		Res: *resp.JSON200,
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
