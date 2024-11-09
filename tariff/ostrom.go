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

	contract, err := util.EnsureElementEx(cc.Contract, t.GetContracts,
		func(c ostrom.Contract) (string, error) {
			return strconv.FormatInt(c.Id, 10), nil
		},
	)
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

func addPrice(entry ostrom.ForecastInfo, rates api.Rates) api.Rates {
	ts := entry.StartTimestamp.Local()
	ar := api.Rate{
		Start: ts,
		End:   ts.Add(time.Hour),
		Price: (entry.Marketprice + entry.AdditionalCost) / 100.0, // Both values include VAT
	}
	return append(rates, ar)
}

func (t *Ostrom) getCityId() (int, error) {
	var city ostrom.CityId

	params := url.Values{
		"zip": {t.zip},
	}

	uri := fmt.Sprintf("%s?%s", ostrom.URI_GET_CITYID, params.Encode())
	if err := backoff.Retry(func() error {
		return backoffPermanentError(t.GetJSON(uri, &city))
	}, bo()); err != nil {
		t.log.ERROR.Println(err)
		return 0, err
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

	return 0, errors.New("Could not find basic tariff in tariff response")
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

	if err != nil {
		t.log.DEBUG.Printf("Requesting token failed with Error: %s\n", err.Error())
	}

	return util.TokenWithExpiry(&res), err
}

func (t *Ostrom) GetContracts() ([]ostrom.Contract, error) {
	var res ostrom.Contracts

	contractsURL := ostrom.URI_API + "/contracts"
	err := t.GetJSON(contractsURL, &res)
	return res.Data, err
}

// This function is used to calculate the prices for the Simplay Fair tarrifs
// using the price given in the configuration
// Unfortunately, the API does not allow to query the price for these yet.
func (t *Ostrom) runStatic(done chan error) {
	var once sync.Once
	var val ostrom.ForecastInfo
	var err error
	val.AdditionalCost = 0

	tick := time.NewTicker(time.Hour)
	for ; true; <-tick.C {
		val.Marketprice, err = t.getFixedPrice()
		if err == nil {
			val.StartTimestamp = now.BeginningOfDay()
			data := make(api.Rates, 0, 48)
			for i := 0; i < 48; i++ {
				data = addPrice(val, data)
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
			data = addPrice(val, data)
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
		t.log.ERROR.Printf("Unknown tariff type %s\n", t.contractType)
		return api.TariffTypePriceStatic
	}
}
