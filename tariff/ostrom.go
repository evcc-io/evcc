package tariff

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/tariff/ostrom"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/oauth"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"github.com/jinzhu/now"
	"golang.org/x/oauth2"
)

type Ostrom struct {
	*embed
	*request.Helper
	log          *util.Logger
	zip          string
	contractType string
	cityId       int // Required for the Fair tariff types
	basic        string
	data         *util.Monitor[api.Rates]
}

var _ api.Tariff = (*Ostrom)(nil)

func init() {
	registry.Add("ostrom", NewOstromFromConfig)
}

// Search for a contract in list of contracts
func ensureContractEx(cid string, contracts []ostrom.Contract) (ostrom.Contract, error) {
	var zero ostrom.Contract

	if cid != "" {
		// cid defined
		for _, contract := range contracts {
			if cid == strconv.FormatInt(contract.Id, 10) {
				return contract, nil
			}
		}
	} else if len(contracts) == 1 {
		// cid empty and exactly one object
		return contracts[0], nil
	}

	return zero, fmt.Errorf("cannot find contract")
}

func NewOstromFromConfig(other map[string]interface{}) (api.Tariff, error) {
	var cc struct {
		ClientId     string
		ClientSecret string
		Contract     string
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.ClientId == "" || cc.ClientSecret == "" {
		return nil, api.ErrMissingCredentials
	}

	basic := transport.BasicAuthHeader(cc.ClientId, cc.ClientSecret)
	log := util.NewLogger("ostrom").Redact(basic)

	t := &Ostrom{
		log:          log,
		basic:        basic,
		contractType: ostrom.PRODUCT_DYNAMIC,
		Helper:       request.NewHelper(log),
		data:         util.NewMonitor[api.Rates](2 * time.Hour),
	}

	t.Client.Transport = &oauth2.Transport{
		Base:   t.Client.Transport,
		Source: oauth.RefreshTokenSource(nil, t),
	}

	contracts, err := t.GetContracts()
	if err != nil {
		return nil, err
	}
	contract, err := ensureContractEx(cc.Contract, contracts)
	if err != nil {
		return nil, err
	}
	done := make(chan error)

	t.contractType = contract.Product
	t.zip = contract.Address.Zip
	if t.Type() == api.TariffTypePriceStatic {
		t.cityId, err = t.getCityId()
		if err != nil {
			return nil, err
		}
		go t.runStatic(done)
	} else {
		go t.run(done)
	}
	err = <-done

	return t, err
}

func rate(entry ostrom.ForecastInfo) api.Rate {
	ts := entry.StartTimestamp.Local()
	return api.Rate{
		Start: ts,
		End:   ts.Add(time.Hour),
		Price: (entry.Marketprice + entry.AdditionalCost) / 100.0, // Both values include VAT
	}
}

func (t *Ostrom) getCityId() (int, error) {
	var city ostrom.CityId

	params := url.Values{
		"zip": {t.zip},
	}

	uri := fmt.Sprintf("%s?%s", ostrom.URI_GET_CITYID, params.Encode())
	if err := t.GetJSON(uri, &city); err != nil {
		return 0, err
	}
	if len(city) < 1 {
		return 0, errors.New("city not found")
	}
	return city[0].Id, nil
}

func (t *Ostrom) getFixedPrice() (float64, error) {
	var tariffs ostrom.Tariffs

	params := url.Values{
		"cityId": {strconv.Itoa(t.cityId)},
		"usage":  {"1000"},
	}

	uri := fmt.Sprintf("%s?%s", ostrom.URI_GET_STATIC_PRICE, params.Encode())
	if err := backoff.Retry(func() error {
		return backoffPermanentError(t.GetJSON(uri, &tariffs))
	}, bo()); err != nil {
		return 0, err
	}

	for _, tariff := range tariffs.Ostrom {
		if tariff.ProductCode == ostrom.PRODUCT_BASIC {
			return tariff.UnitPricePerkWH, nil
		}
	}

	return 0, errors.New("tariff not found")
}

func (t *Ostrom) RefreshToken(_ *oauth2.Token) (*oauth2.Token, error) {
	tokenURL := ostrom.URI_AUTH + "/oauth2/token"
	dataReader := strings.NewReader("grant_type=client_credentials")
	req, _ := request.New(http.MethodPost, tokenURL, dataReader, map[string]string{
		"Authorization": t.basic,
		"Content-Type":  request.FormContent,
		"Accept":        request.JSONContent,
	})

	var res oauth2.Token
	client := request.NewHelper(t.log)
	err := client.DoJSON(req, &res)
	return util.TokenWithExpiry(&res), err
}

func (t *Ostrom) GetContracts() ([]ostrom.Contract, error) {
	var res ostrom.Contracts

	uri := ostrom.URI_API + "/contracts"
	err := t.GetJSON(uri, &res)
	return res.Data, err
}

// This function is used to calculate the prices for the Simplay Fair tarrifs
// using the price given in the configuration
// Unfortunately, the API does not allow to query the price for these yet.
func (t *Ostrom) runStatic(done chan error) {
	var once sync.Once
	var val ostrom.ForecastInfo
	var err error

	tick := time.NewTicker(time.Hour)
	for ; true; <-tick.C {
		val.Marketprice, err = t.getFixedPrice()
		if err == nil {
			val.StartTimestamp = now.BeginningOfDay()
			data := make(api.Rates, 48)
			for i := range data {
				data[i] = rate(val)
				val.StartTimestamp = val.StartTimestamp.Add(time.Hour)
			}
			mergeRates(t.data, data)
		} else {
			t.log.ERROR.Println(err)
		}
		once.Do(func() { close(done) })
	}
}

// This function calls th ostrom API to query the
// dynamic prices
func (t *Ostrom) run(done chan error) {
	var once sync.Once

	tick := time.NewTicker(time.Hour)
	for ; true; <-tick.C {
		var res ostrom.Prices

		start := now.BeginningOfDay()
		end := start.AddDate(0, 0, 2)

		params := url.Values{
			"startDate":  {start.Format(time.RFC3339)},
			"endDate":    {end.Format(time.RFC3339)},
			"resolution": {"HOUR"},
			"zip":        {t.zip},
		}

		uri := fmt.Sprintf("%s/spot-prices?%s", ostrom.URI_API, params.Encode())
		if err := backoff.Retry(func() error {
			return backoffPermanentError(t.GetJSON(uri, &res))
		}, bo()); err != nil {
			once.Do(func() { done <- err })

			t.log.ERROR.Println(err)
			continue
		}

		data := make(api.Rates, 0, 48)
		for _, val := range res.Data {
			data = append(data, rate(val))
		}

		mergeRates(t.data, data)
		once.Do(func() { close(done) })
	}
}

// Rates implements the api.Tariff interface
func (t *Ostrom) Rates() (api.Rates, error) {
	var res api.Rates
	err := t.data.GetFunc(func(val api.Rates) {
		res = slices.Clone(val)
	})
	return res, err
}

// Type implements the api.Tariff interface
func (t *Ostrom) Type() api.TariffType {
	switch t.contractType {
	case ostrom.PRODUCT_DYNAMIC:
		return api.TariffTypePriceForecast
	case ostrom.PRODUCT_FAIR, ostrom.PRODUCT_FAIR_CAP:
		return api.TariffTypePriceStatic
	default:
		return api.TariffTypePriceStatic
	}
}
