package tariff

import (
	"errors"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	octopusde "github.com/evcc-io/evcc/tariff/octopus-de"
	"github.com/evcc-io/evcc/util"
	"golang.org/x/exp/slices"
)

type OctopusDE struct {
	*octopusde.OctopusDEGraphQLClient
	data *util.Monitor[api.Rates]
	log  *util.Logger
}

var _ api.Tariff = (*OctopusDE)(nil)

func init() {
	registry.Add("octopus-de", NewOctopusDEFromConfig)
}

// NewOctopusDEFromConfig creates an OctopusDE tariff provider
func NewOctopusDEFromConfig(other map[string]interface{}) (api.Tariff, error) {
	var cc struct {
		Email    string
		Password string
	}

	logger := util.NewLogger("octopus-de")

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.Email == "" {
		return nil, errors.New("missing email")
	}
	if cc.Password == "" {
		return nil, errors.New("missing password")
	}

	client, err := octopusde.NewClient(logger, cc.Email, cc.Password)
	if err != nil {
		return nil, err
	}

	t := &OctopusDE{
		OctopusDEGraphQLClient: client,
		data:                   util.NewMonitor[api.Rates](2 * time.Minute),
		log:                    logger,
	}

	done := make(chan error)
	go t.run(done)
	err = <-done

	return t, err
}

// run starts the rate updater
func (t *OctopusDE) run(done chan error) {
	var once sync.Once

	// Initial fetch
	if err := t.updateRates(); err != nil {
		once.Do(func() { done <- err })
		t.log.ERROR.Println(err)
		return
	}
	once.Do(func() { close(done) })

	// Update every 15 minutes
	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()

	for ; true; <-ticker.C {
		if err := t.updateRates(); err != nil {
			t.log.ERROR.Println(err)
		}
	}
}

// updateRates fetches and processes rate information
func (t *OctopusDE) updateRates() error {
	// Default rate as fallback
	defaultRate := octopusde.Rate{
		Price:     0.2827,
		StartTime: "00:00:00",
		EndTime:   "00:00:00",
		Name:      "DEFAULT",
	}

	// Refresh token
	if err := t.RefreshToken(); err != nil {
		t.log.ERROR.Printf("Failed to refresh token: %v. Using cached rates.", err)
		return nil
	}

	// Get account number
	accountNumber, err := t.AccountNumber()
	if err != nil {
		t.log.ERROR.Printf("Failed to fetch account number: %v. Using cached rates.", err)
		return nil
	}

	// Fetch rates
	rates, err := t.FetchRates(accountNumber)
	if err != nil {
		t.log.ERROR.Printf("Failed to fetch rates: %v. Using default rate.", err)
		rates = []octopusde.Rate{defaultRate}
	}

	// Generate hourly rates for 72 hours
	data := make(api.Rates, 0, 72)
	now := time.Now()
	startHour := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, now.Location())

	for i := 0; i < 72; i++ {
		hour := startHour.Add(time.Duration(i) * time.Hour)
		nextHour := hour.Add(time.Hour)
		hourStr := hour.Format("15:04:05")

		// Find applicable rate
		price, name := defaultRate.Price, defaultRate.Name
		found := false

		for _, r := range rates {
			// Skip invalid rates
			if r.StartTime == "" {
				continue
			}

			// Handle time slots including overnight
			if r.EndTime == "00:00:00" {
				if hourStr >= r.StartTime || hourStr < r.EndTime {
					price, name = r.Price, r.Name
					found = true
					break
				}
			} else if hourStr >= r.StartTime && hourStr < r.EndTime {
				price, name = r.Price, r.Name
				found = true
				break
			}
		}

		// If no match found, use first rate as default
		if !found && len(rates) > 0 {
			price = rates[0].Price
			name = rates[0].Name
		}

		t.log.TRACE.Printf("Hour %s: %.4f €/kWh (%s)", hourStr, price, name)

		data = append(data, api.Rate{
			Start: hour,
			End:   nextHour,
			Price: price,
		})
	}

	// Update stored rates
	if len(data) > 0 {
		mergeRates(t.data, data)
	}

	return nil
}

// Rates implements the api.Tariff interface
func (t *OctopusDE) Rates() (api.Rates, error) {
	var res api.Rates
	err := t.data.GetFunc(func(val api.Rates) {
		res = slices.Clone(val)
	})
	return res, err
}

// Type implements the api.Tariff interface
func (t *OctopusDE) Type() api.TariffType {
	return api.TariffTypePriceForecast
}
