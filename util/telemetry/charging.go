package telemetry

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/sponsor"
)

var (
	api     = "https://api.evcc.io"
	Enabled bool
)

func Create(token string) error {
	if token == "" {
		return errors.New("telemetry requires sponsorship")
	}

	Enabled = true
	return nil
}

func ChargeProgress(log *util.Logger, power, deltaCharged, deltaGreen float64) {
	log.DEBUG.Printf("telemetry: charge: Δ%.0f/%.0fWh @ %.0fW", deltaGreen*1e3, deltaCharged*1e3, power)

	data := struct {
		ChargePower  float64 `json:"chargePower"`
		ChargeEnergy float64 `json:"chargeEnergy"`
		GreenEnergy  float64 `json:"greenEnergy"`
	}{
		ChargePower:  power,
		ChargeEnergy: deltaCharged,
		GreenEnergy:  deltaGreen,
	}

	uri := fmt.Sprintf("%s/v1/charge", api)
	req, err := request.New(http.MethodPost, uri, request.MarshalJSON(data), map[string]string{
		"Authorization": "Bearer " + sponsor.Token,
	})

	var res struct {
		Error string
	}

	if err == nil {
		client := request.NewHelper(log)
		if err = client.DoJSON(req, &res); err == nil && res.Error != "" {
			err = errors.New(res.Error)
		}
	}

	if err != nil {
		log.ERROR.Printf("telemetry: charge: %v", err)
	}
}
