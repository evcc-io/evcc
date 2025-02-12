package tariff

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
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
func ensureContractEx(cid int64, contracts []ostrom.Contract) (ostrom.Contract, error) {
	var zero ostrom.Contract

	if cid != -1 {
		// cid defined
		for _, contract := range contracts {
			if cid == contract.Id {
				return contract, nil
			}
		}
	} else if len(contracts) == 1 {
		// cid empty and exactly one object
		return contracts[0], nil
	}

	return zero, errors.New("cannot find contract")
}

func NewOstromFromConfig(other map[string]interface{}) (api.Tariff, error) {
	var cc struct {
		ClientId     string
		ClientSecret string
		Contract     int64
	}

	cc.Contract = -1
	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.ClientId == "" || cc.ClientSecret == "" {
		return nil, api.ErrMissingCredentials
	}

	basic := transport.BasicAuthHeader(cc.ClientId, cc.ClientSecret)
	log := util.NewLogger("ostrom").Redact(basic)

	t := &Ostrom{
		log:    log,
		basic:  basic,
		Helper: request.NewHelper(log),
		data:   util.NewMonitor[api.Rates](2 * time.Hour),
	}

	t.Client.Transport = &oauth2.Transport{
		Base:   t.Client.Transport,
		Source: oauth.RefreshTokenSource(nil, t),
	}

	contracts, err := t.getContracts()
	if err != nil {
		return nil, err
	}
	contract, err := ensureContractEx(cc.Contract, contracts)
	if err != nil {
		return nil, err
	}

	t.contractType = contract.Product
	t.zip = contract.Address.Zip

	done := make(chan error)
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

func (t *Ostrom) getContracts() ([]ostrom.Contract, error) {
	var res ostrom.Contracts

	uri := ostrom.URI_API + "/contracts"
	err := t.GetJSON(uri, &res)
	return res.Data, err
}

func (t *Ostrom) getCityId() (int, error) {
	var city ostrom.CityId

	uri := fmt.Sprintf("%s?zip=%s", ostrom.URI_GET_CITYID, t.zip)
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

	uri := fmt.Sprintf("%s?usage=1000&cityId=%d", ostrom.URI_GET_STATIC_PRICE, t.cityId)
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
	uri := ostrom.URI_AUTH + "/oauth2/token"
	data := url.Values{"grant_type": {"client_credentials"}}
	req, _ := request.New(http.MethodPost, uri, strings.NewReader(data.Encode()), map[string]string{
		"Authorization": t.basic,
		"Content-Type":  request.FormContent,
		"Accept":        request.JSONContent,
	})

	var res oauth2.Token
	client := request.NewHelper(t.log)
	err := client.DoJSON(req, &res)
	return util.TokenWithExpiry(&res), err
}

// This function is used to calculate the prices for the Simplay Fair tarrifs
// using the price given in the configuration
// Unfortunately, the API does not allow to query the price for these yet.
func (t *Ostrom) runStatic(done chan error) {
	var once sync.Once

	for tick := time.Tick(time.Hour); ; <-tick {
		price, err := t.getFixedPrice()
		if err != nil {
			once.Do(func() { done <- err })
			t.log.ERROR.Println(err)
			continue
		}

		data := make(api.Rates, 48)
		for i := range data {
			ts := now.BeginningOfDay().Add(time.Duration(i) * time.Hour)
			data[i] = api.Rate{
				Start: ts,
				End:   ts.Add(time.Hour),
				Price: price / 100.0,
			}
		}

		mergeRates(t.data, data)
		once.Do(func() { close(done) })
	}
}

// This function calls th ostrom API to query the
// dynamic prices
func (t *Ostrom) run(done chan error) {
	var once sync.Once

	for tick := time.Tick(time.Hour); ; <-tick {
		var res ostrom.Prices

		start := now.BeginningOfDay()
		end := start.AddDate(0, 0, 2)

		params := url.Values{
			"startDate":  {start.Format("2006-01-02T15:04:05.000Z07:00")},
			"endDate":    {end.Format("2006-01-02T15:04:05.000Z07:00")},
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
			ts := val.StartTimestamp.Local()
			data = append(data, api.Rate{
				Start: ts,
				End:   ts.Add(time.Hour),
				Price: (val.Marketprice + val.AdditionalCost) / 100.0, // Both values include VAT
			})
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
		panic("invalid contract type: " + t.contractType)
	}
}
