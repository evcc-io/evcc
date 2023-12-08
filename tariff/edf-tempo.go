package tariff

import (
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/oauth"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"github.com/fatih/structs"
	"github.com/jinzhu/now"
	"golang.org/x/oauth2"
)

type EdfTempo struct {
	*embed
	*request.Helper
	log    *util.Logger
	basic  string
	data   *util.Monitor[api.Rates]
	prices map[string]float64
}

var _ api.Tariff = (*EdfTempo)(nil)

func init() {
	registry.Add("edf-tempo", NewEdfTempoFromConfig)
}

func NewEdfTempoFromConfig(other map[string]interface{}) (api.Tariff, error) {
	var cc struct {
		embed        `mapstructure:",squash"`
		ClientID     string
		ClientSecret string
		Prices       struct {
			Blue, Red, White float64 `structs:",omitempty"`
		}
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.ClientID == "" && cc.ClientSecret == "" {
		return nil, errors.New("missing credentials")
	}

	basic := transport.BasicAuthHeader(cc.ClientID, cc.ClientSecret)
	log := util.NewLogger("edf-tempo").Redact(basic)

	t := &EdfTempo{
		embed:  &cc.embed,
		log:    log,
		basic:  basic,
		Helper: request.NewHelper(log),
		data:   util.NewMonitor[api.Rates](2 * time.Hour),
	}

	prices := structs.Map(cc.Prices)
	if len(prices) != 3 {
		return nil, errors.New("missing prices for red/blue/white")
	}

	for k, v := range prices {
		t.prices[strings.ToLower(k)] = v.(float64)
	}

	t.Client.Transport = &oauth2.Transport{
		Base:   t.Client.Transport,
		Source: oauth.RefreshTokenSource(new(oauth2.Token), t),
	}

	done := make(chan error)
	go t.run(done)
	err := <-done

	return t, err
}

func (t *EdfTempo) RefreshToken(_ *oauth2.Token) (*oauth2.Token, error) {
	tokenURL := "https://digital.iservices.rte-france.com/token/oauth"
	req, _ := request.New(http.MethodPost, tokenURL, nil, map[string]string{
		"Authorization": t.basic,
		"Content-Type":  request.FormContent,
		"Accept":        request.JSONContent,
	})

	var res oauth.Token
	client := request.NewHelper(t.log)
	err := client.DoJSON(req, &res)

	return (*oauth2.Token)(&res), err
}

func (t *EdfTempo) run(done chan error) {
	var once sync.Once
	bo := newBackoff()

	for ; true; <-time.Tick(time.Hour) {
		var res struct {
			Data struct {
				Values []struct {
					StartDate time.Time `json:"start_date"`
					EndDate   time.Time `json:"end_date"`
					Value     string    `json:"value"`
				} `json:"values"`
			} `json:"tempo_like_calendars"`
		}

		start := now.BeginningOfDay()
		end := start.AddDate(0, 0, 2)

		uri := fmt.Sprintf("https://digital.iservices.rte-france.com/open_api/tempo_like_supply_contract/v1/tempo_like_calendars?start_date=%s&end_date=%s&fallback_status=true",
			strings.ReplaceAll(start.Format(time.RFC3339), "+", "%2B"),
			strings.ReplaceAll(end.Format(time.RFC3339), "+", "%2B"))

		if err := backoff.Retry(func() error {
			return backoffPermanentError(t.GetJSON(uri, &res))
		}, bo); err != nil {
			once.Do(func() { done <- err })

			t.log.ERROR.Println(err)
			continue
		}

		once.Do(func() { close(done) })

		data := make(api.Rates, 0, 24*len(res.Data.Values))
		for _, r := range res.Data.Values {
			for ts := r.StartDate.Local(); ts.Before(r.EndDate); ts = ts.Add(time.Hour) {
				ar := api.Rate{
					Start: ts,
					End:   ts.Add(time.Hour),
					Price: t.totalPrice(t.prices[strings.ToLower(r.Value)]),
				}
				data = append(data, ar)
			}
		}
		data.Sort()

		t.data.Set(data)
	}
}

// Rates implements the api.Tariff interface
func (t *EdfTempo) Rates() (api.Rates, error) {
	var res api.Rates
	err := t.data.GetFunc(func(val api.Rates) {
		res = slices.Clone(val)
	})
	return res, err
}

// Type implements the api.Tariff interface
func (t *EdfTempo) Type() api.TariffType {
	return api.TariffTypePriceForecast
}
