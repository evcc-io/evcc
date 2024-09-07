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
	basic        string
	data         *util.Monitor[api.Rates]
}

var _ api.Tariff = (*Ostrom)(nil)

func init() {
	registry.Add("ostrom", NewOstromFromConfig)
}

func NewOstromFromConfig(other map[string]interface{}) (api.Tariff, error) {
	var cc struct {
		embed        `mapstructure:",squash"`
		ClientID     string
		ClientSecret string
		Contract     string
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.ClientID == "" || cc.ClientSecret == "" {
		return nil, errors.New("missing credentials")
	}

	basic := transport.BasicAuthHeader(cc.ClientID, cc.ClientSecret)
	log := util.NewLogger("ostrom").Redact(basic)

	t := &Ostrom{
		embed:        &cc.embed,
		log:          log,
		basic:        basic,
		contractType: ostrom.PRODUCT_DYNAMIC,
		zip:          "",
		Helper:       request.NewHelper(log),
		data:         util.NewMonitor[api.Rates](2 * time.Hour),
	}

	t.Client.Transport = &oauth2.Transport{
		Base:   t.Client.Transport,
		Source: oauth.RefreshTokenSource(new(oauth2.Token), t),
	}

	contract, err := util.EnsureElementEx(cc.Contract, t.GetContracts,
		func(c ostrom.Contract) (string, error) {
			return c.Id, nil
		},
	)
	if err != nil {
		return nil, err
	}
	t.contractType = contract.Type
	t.zip = contract.Address.Zip

	done := make(chan error)
	go t.run(done)
	err = <-done

	return t, err
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
		if count > 0 {
			count--
			r := res.Data[0]
			n := res.Data[1]
			for i := 0; i < count; i++ {
				for ts := r.StartTimestamp.Local(); ts.Before(n.StartTimestamp); ts = ts.Add(time.Hour) {
					ar := api.Rate{
						Start: ts,
						End:   ts.Add(time.Hour),
						Price: r.Marketprice + r.AdditionalCost, // Both values include VAT
					}
					data = append(data, ar)
				}
				r = n
				n = res.Data[i]
			}
			// And now the last one
			ar := api.Rate{
				Start: n.StartTimestamp.Local(),
				End:   n.StartTimestamp.Add(time.Hour).Local(),
				Price: n.Marketprice + n.AdditionalCost, // Both values include VAT
			}
			data = append(data, ar)
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
