package tariff

import (
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	octopusde "github.com/evcc-io/evcc/tariff/octopus-de"
	"github.com/evcc-io/evcc/util"
	"golang.org/x/exp/slices"
)

type OctopusDEClient struct {
	*octopusde.OctopusDEGraphQLClient
	data *util.Monitor[api.Rates]
	log  *util.Logger
}

func (t *OctopusDEClient) run(done chan error) {
	var once sync.Once

	// Initial fetch
	if err := t.updateRates(); err != nil {
		once.Do(func() { done <- err })
		t.log.ERROR.Println(err)
		return
	}

	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for ; true; <-ticker.C {
		if err := t.updateRates(); err != nil {
			t.log.ERROR.Println(err)
		}
	}
}

func (t *OctopusDEClient) updateRates() error {
	// Refresh the token
	if err := t.RefreshToken(); err != nil {
		return err
	}

	// Fetch the account number
	accountNumber, err := t.AccountNumber()
	if err != nil {
		return err
	}

	// Fetch and store gross rates
	grossRates, err := t.FetchGrossRates(accountNumber)
	if err != nil {
		t.log.ERROR.Println("Failed to fetch gross rates, using default rate: 23.25")
		grossRates = []float64{23.25}
	}

	// Store the gross rates for 24 hours a day
	data := make(api.Rates, 24)
	for i := 0; i < 24; i++ {
		data[i] = api.Rate{
			Start: time.Now().Add(time.Duration(i) * time.Hour),
			End:   time.Now().Add(time.Duration(i+1) * time.Hour),
			Price: grossRates[0], // Assuming the same rate for all hours
		}
	}
	mergeRates(t.data, data)
	return nil
}

// Rates implements the api.Tariff interface
func (t *OctopusDEClient) Rates() (api.Rates, error) {
	var res api.Rates
	err := t.data.GetFunc(func(val api.Rates) {
		res = slices.Clone(val)
	})
	return res, err
}

// Type implements the api.Tariff interface
func (t *OctopusDEClient) Type() api.TariffType {
	return api.TariffTypePriceForecast
}
