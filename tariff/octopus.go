package tariff

import (
	"encoding/json"
	"errors"
	"io"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/evcc-io/evcc/api"
	octoGql "github.com/evcc-io/evcc/tariff/octopus/graphql"
	octoRest "github.com/evcc-io/evcc/tariff/octopus/rest"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

type Octopus struct {
	log           *util.Logger
	region        string
	productCode   string
	apikey        string
	token         string
	goPrice       float64
	standardPrice float64
	data          *util.Monitor[api.Rates]
}

var _ api.Tariff = (*Octopus)(nil)

func init() {
	// Register the Octopus tariff
	api.RegisterTariff("octopusenergy", NewOctopusFromConfig)
}

func NewOctopusFromConfig(other map[string]interface{}) (api.Tariff, error) {
	var cc struct {
		Region        string
		Tariff        string // DEPRECATED: use ProductCode
		ProductCode   string
		ApiKey        string
		Email         string
		Password      string
		RegionType    string
		GoPrice       float64
		StandardPrice float64
	}

	logger := util.NewLogger("octopus")

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	// Allow ApiKey or Token to be missing only if Region and Tariff are not.
	if cc.ApiKey == "" && (cc.Email == "" || cc.Password == "") {
		if cc.Region == "" {
			return nil, errors.New("missing region")
		}
		if cc.Tariff != "" {
			// deprecated - copy to correct slot and WARN
			logger.WARN.Print("'tariff' is deprecated and will break in a future version - use 'productCode' instead")
			cc.ProductCode = cc.Tariff
		}
		if cc.ProductCode == "" {
			return nil, errors.New("missing product code")
		}
	} else if cc.ApiKey != "" {
		// ApiKey validators
		if cc.Region != "" || cc.Tariff != "" {
			return nil, errors.New("cannot use apikey at same time as product code")
		}
		if len(cc.ApiKey) != 32 || !strings.HasPrefix(cc.ApiKey, "sk_live_") {
			return nil, errors.New("invalid apikey format")
		}
	}

	t := &Octopus{
		log:           logger,
		region:        cc.Region,
		productCode:   cc.ProductCode,
		apikey:        cc.ApiKey,
		goPrice:       cc.GoPrice,
		standardPrice: cc.StandardPrice,
		data:          util.NewMonitor[api.Rates](2 * time.Hour),
	}

	// Get token if Email and Password are provided
	if cc.Email != "" && cc.Password != "" {
		token, err := octoGql.GetKrakenToken(cc.Email, cc.Password)
		if err != nil {
			return nil, err
		}
		t.token = token
	}

	done := make(chan error)
	go t.run(cc.RegionType, done)
	err := <-done

	return t, err
}

func (t *Octopus) run(regionType string, done chan error) {
	var once sync.Once
	client := request.NewHelper(t.log)

	var restQueryUri string

	// If ApiKey is available, use GraphQL to get appropriate tariff code before entering execution loop.
	if t.apikey != "" {
		gqlCli, err := octoGql.NewClient(t.log, t.apikey)
		if err != nil {
			once.Do(func() { done <- err })
			t.log.ERROR.Println(err)
			return
		}
		tariffCode, err := gqlCli.TariffCode()
		if err != nil {
			once.Do(func() { done <- err })
			t.log.ERROR.Println(err)
			return
		}
		restQueryUri = octoRest.ConstructRatesAPIFromTariffCode(tariffCode)
	} else if t.token != "" && regionType == "DE" {
		// Use GraphQL with token to get appropriate tariff code before entering execution loop.
		gqlCli, err := octoGql.NewClientWithEmailPassword(t.log, t.token)
		if err != nil {
			once.Do(func() { done <- err })
			t.log.ERROR.Println(err)
			return
		}
		tariffCode, err := gqlCli.GermanTariffCode()
		if err != nil {
			once.Do(func() { done <- err })
			t.log.ERROR.Println(err)
			return
		}
		restQueryUri = octoRest.ConstructRatesAPIFromTariffCode(tariffCode)
	} else {
		// Construct Rest Query URI using tariff and region codes.
		restQueryUri = octoRest.ConstructRatesAPIFromProductAndRegionCode(t.productCode, t.region)
	}

	// TODO tick every 15 minutes if GraphQL is available to poll for Intelligent slots.
	for tick := time.Tick(time.Hour); ; <-tick {
		var res octoRest.UnitRates

		if err := backoff.Retry(func() error {
			return backoffPermanentError(client.GetJSON(restQueryUri, &res))
		}, bo()); err != nil {
			once.Do(func() { done <- err })

			t.log.ERROR.Println(err)
			continue
		}

		data := make(api.Rates, 0, len(res.Results))
		for _, r := range res.Results {
			ar := api.Rate{
				Start: r.ValidityStart,
				End:   r.ValidityEnd,
				// Apply GoPrice from 0-5 am UTC or during planned dispatch periods, else use StandardPrice
				Price: t.applyPrice(r.ValidityStart, r.ValidityEnd, r.PriceInclusiveTax/1e2),
			}
			data = append(data, ar)
		}

		mergeRates(t.data, data)
		once.Do(func() { close(done) })
	}
}

// applyPrice applies the GoPrice or StandardPrice based on the time of day and planned dispatch periods
func (t *Octopus) applyPrice(start, end time.Time, basePrice float64) float64 {
	startHour := start.UTC().Hour()
	endHour := end.UTC().Hour()

	// Check if the rate falls within the GoPrice period (0-5 am UTC)
	if (startHour >= 0 && startHour < 5) || (endHour > 0 && endHour <= 5) {
		return t.goPrice
	}

	// Check if the rate falls within a planned dispatch period
	if t.isPlannedDispatch(start, end) {
		return t.goPrice
	}

	// Otherwise, use the StandardPrice
	return t.standardPrice
}

// isPlannedDispatch checks if the given time period overlaps with a planned dispatch period
func (t *Octopus) isPlannedDispatch(start, end time.Time) bool {
	// Implement the logic to check for planned dispatch periods
	// Example implementation
	var dispatches []struct {
		Start time.Time `json:"start"`
		End   time.Time `json:"end"`
	}

	// Fetch planned dispatch periods from the API
	client := request.NewHelper(t.log)
	body, err := json.Marshal(struct {
		Query     string `json:"query"`
		Variables struct {
			AccountNumber string `json:"accountNumber"`
		} `json:"variables"`
	}{
		Query: `query getPlannedDispatches($accountNumber: String!) {
					plannedDispatches(accountNumber: $accountNumber) {
						start
						end
					}
				}`,
		Variables: struct {
			AccountNumber string `json:"accountNumber"`
		}{
			AccountNumber: "your_account_number", // Replace with actual account number
		},
	})
	if err != nil {
		t.log.ERROR.Println(err)
		return false
	}

	resp, err := client.Post(octoGql.GermanURI, "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.log.ERROR.Println(err)
		return false
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&dispatches); err != nil {
		t.log.ERROR.Println(err)
		return false
	}

	for _, d := range dispatches {
		if start.Before(d.End) && end.After(d.Start) {
			return true
		}
	}

	return false
}

// Rates implements the api.Tariff interface
func (t *Octopus) Rates() (api.Rates, error) {
	var res api.Rates
	err := t.data.GetFunc(func(val api.Rates) {
		res = slices.Clone(val)
	})
	return res, err
}

// Type implements the api.Tariff interface
func (t *Octopus) Type() api.TariffType {
	return api.TariffTypePriceForecast
}
