package tariff

import (
	"errors"
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
	log         *util.Logger
	region      string
	productCode string
	apikey      string
	data        *util.Monitor[api.Rates]
}

var _ api.Tariff = (*Octopus)(nil)

func init() {
	registry.Add("octopusenergy", NewOctopusFromConfig)
}

func NewOctopusFromConfig(other map[string]interface{}) (api.Tariff, error) {
	var cc struct {
		Region      string
		Tariff      string // DEPRECATED: use ProductCode
		ProductCode string
		ApiKey      string
	}

	logger := util.NewLogger("octopus")

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	// Allow ApiKey to be missing only if Region and Tariff are not.
	if cc.ApiKey == "" {
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
	} else {
		// ApiKey validators
		if cc.Region != "" || cc.Tariff != "" {
			return nil, errors.New("cannot use apikey at same time as product code")
		}
		if len(cc.ApiKey) != 32 || !strings.HasPrefix(cc.ApiKey, "sk_live_") {
			return nil, errors.New("invalid apikey format")
		}
	}

	t := &Octopus{
		log:         logger,
		region:      cc.Region,
		productCode: cc.ProductCode,
		apikey:      cc.ApiKey,
		data:        util.NewMonitor[api.Rates](2 * time.Hour),
	}

	done := make(chan error)
	go t.run(done)
	err := <-done

	return t, err
}

func (t *Octopus) run(done chan error) {
	var once sync.Once
	client := request.NewHelper(t.log)
	bo := newBackoff()

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
	} else {
		// Construct Rest Query URI using tariff and region codes.
		restQueryUri = octoRest.ConstructRatesAPIFromProductAndRegionCode(t.productCode, t.region)
	}

	// TODO tick every 15 minutes if GraphQL is available to poll for Intelligent slots.
	tick := time.NewTicker(time.Hour)
	for ; true; <-tick.C {
		var res octoRest.UnitRates

		if err := backoff.Retry(func() error {
			return backoffPermanentError(client.GetJSON(restQueryUri, &res))
		}, bo); err != nil {
			once.Do(func() { done <- err })

			t.log.ERROR.Println(err)
			continue
		}

		data := make(api.Rates, 0, len(res.Results))
		for _, r := range res.Results {
			ar := api.Rate{
				Start: r.ValidityStart,
				End:   r.ValidityEnd,
				// UnitRates are supplied inclusive of tax, though this could be flipped easily with a config flag.
				Price: r.PriceInclusiveTax / 1e2,
			}
			data = append(data, ar)
		}
		data.Sort()

		t.data.Set(data)
		once.Do(func() { close(done) })
	}
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
