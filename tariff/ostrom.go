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
	price        float64 // Required for the Fair tariff types
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
		Price        float64
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.ClientId == "" || cc.ClientSecret == "" {
		return nil, errors.New("missing credentials")
	}

	basic := transport.BasicAuthHeader(cc.ClientId, cc.ClientSecret)
	log := util.NewLogger("ostrom").Redact(basic)

	t := &Ostrom{
		log:          log,
		basic:        basic,
		contractType: ostrom.PRODUCT_DYNAMIC,
		zip:          "",
		price:        0.0,
		Helper:       request.NewHelper(log),
		data:         util.NewMonitor[api.Rates](2 * time.Hour),
	}

	t.Client.Transport = &oauth2.Transport{
		Base:   t.Client.Transport,
		Source: oauth.RefreshTokenSource(new(oauth2.Token), t),
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
		log.DEBUG.Printf("Static Price is %f\n", cc.Price)
		if cc.Price == 0.0 {
			return nil, errors.New("You have to define a price for SIMPLY FAIR tariffs")
		}
		t.price = cc.Price * 100
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
	t.log.DEBUG.Println(res.Data)
	return res.Data, err
}

// This function is used to calculate the prices for the Simplay Fair tarrifs
// using the price given in the configuration
// Unfortunately, the API does not allow to query the price for these yet.
func (t *Ostrom) runStatic(done chan error) {
	var once sync.Once
	var val ostrom.ForecastInfo
	val.AdditionalCost = 0
	val.Marketprice = t.price

	tick := time.NewTicker(time.Hour)
	for ; true; <-tick.C {
		val.StartTimestamp = now.BeginningOfDay()
		data := make(api.Rates, 0, 48)
		for i := 0; i < 48; i++ {
			data = addPrice(val, data)
			val.StartTimestamp = val.StartTimestamp.Add(time.Hour)
		}
		mergeRates(t.data, data)
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
		count := len(res.Data)
		for i := 0; i < count; i++ {
			data = addPrice(res.Data[i], data)
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
	if t.contractType == ostrom.PRODUCT_DYNAMIC {
		return api.TariffTypePriceForecast
	} else if t.contractType == ostrom.PRODUCT_FAIR || t.contractType == ostrom.PRODUCT_FAIR_CAP {
		return api.TariffTypePriceStatic
	} else {
		t.log.ERROR.Printf("Unknown tariff type %s\n", t.contractType)
		return api.TariffTypePriceStatic
	}
}
