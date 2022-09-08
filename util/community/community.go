package community

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
	enabled bool
)

func Create(token string) error {
	if token == "" {
		return errors.New("community requires sponsorship")
	}

	enabled = true
	return nil
}

func ChargeProgress(log *util.Logger, power, energy float64) {
	if !enabled {
		return
	}

	go func() {
		data := struct {
			Power, Energy float64
		}{
			Power:  power,
			Energy: energy,
		}

		uri := fmt.Sprintf("%s/%s/%s", api, "charged", sponsor.Token)
		req, err := request.New(http.MethodPost, uri, request.MarshalJSON(data))

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
			log.ERROR.Printf("community api: %v", err)
		}
	}()
}
