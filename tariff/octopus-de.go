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

// Implementation of intelligent dispatch times is WIP
// NewOctopusDEFromConfig creates a new OctopusDE instance from the given configuration.
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

func (t *OctopusDE) run(done chan error) {
	var once sync.Once

	// Initial fetch
	if err := t.updateRates(); err != nil {
		once.Do(func() { done <- err })
		t.log.ERROR.Println(err)
		return
	}
	once.Do(func() { close(done) })

	// Ticker to update rates every 15 minutes
	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()
	for ; true; <-ticker.C {
		if err := t.updateRates(); err != nil {
			t.log.ERROR.Println(err)
		}
	}
}

func (t *OctopusDE) updateRates() error {
	// Refresh the token
	if err := t.RefreshToken(); err != nil {
		t.log.ERROR.Printf("Failed to refresh token: %v. Using cached rates.", err)
		return nil // Don't fail completely, just use cached rates
	}

	// Fetch the account number
	accountNumber, err := t.AccountNumber()
	if err != nil {
		t.log.ERROR.Printf("Failed to fetch account number: %v. Using cached rates.", err)
		return nil // Don't fail completely, just use cached rates
	}

	// Fetch detailed rates
	rates, err := t.FetchRates(accountNumber)
	if err != nil {
		t.log.ERROR.Printf("Failed to fetch rates: %v. Using default rate.", err)
		rates = []octopusde.Rate{
			{
				Price:     0.2827, // Default fallback rate
				StartTime: "00:00:00",
				EndTime:   "00:00:00",
				Name:      "DEFAULT",
			},
		}
	}

	// Generate hourly rates for the next 72 hours
	data := make(api.Rates, 0, 72)
	now := time.Now()

	// Round to the nearest hour
	startHour := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, now.Location())

	for i := 0; i < 72; i++ {
		hour := startHour.Add(time.Duration(i) * time.Hour)
		nextHour := hour.Add(time.Hour)

		hourStr := hour.Format("15:04:05")

		// Find applicable rate for this hour
		var price float64 = 0.2827 // Default price
		var rateName string = "DEFAULT"

		foundRate := false
		for _, r := range rates {
			// Skip invalid rates
			if r.StartTime == "" {
				continue
			}

			// Handle special case for rates spanning midnight
			if r.EndTime == "00:00:00" {
				if hourStr >= r.StartTime || hourStr < r.EndTime {
					price = r.Price
					rateName = r.Name
					foundRate = true
					break
				}
			} else if hourStr >= r.StartTime && hourStr < r.EndTime {
				price = r.Price
				rateName = r.Name
				foundRate = true
				break
			}
		}

		if !foundRate && len(rates) > 0 {
			// If no specific time slot matches, use the first rate as default
			price = rates[0].Price
			rateName = rates[0].Name
		}

		t.log.TRACE.Printf("Hour %s: %.4f €/kWh (%s)", hourStr, price, rateName)

		data = append(data, api.Rate{
			Start: hour,
			End:   nextHour,
			Price: price,
		})
	}

	// Only update if we have valid data
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
