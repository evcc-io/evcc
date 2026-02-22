package tariff

import (
	"fmt"
	"net/http"
	"slices"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/tariff/rabot"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"golang.org/x/oauth2"
)

type Rabot struct {
	*embed
	*request.Helper
	log        *util.Logger
	gross      bool
	contractId string
	data       *util.Monitor[api.Rates]
}

var _ api.Tariff = (*Rabot)(nil)

func init() {
	registry.Add("rabot", NewRabotFromConfig)
}

func NewRabotFromConfig(other map[string]any) (api.Tariff, error) {
	cc := struct {
		embed    `mapstructure:",squash"`
		Login    string
		Password string
		Gross    bool
	}{}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if err := cc.init(); err != nil {
		return nil, err
	}

	if cc.Gross && (cc.Login == "" || cc.Password == "") {
		return nil, api.ErrMissingCredentials
	}

	log := util.NewLogger("rabot").Redact(cc.Password)

	t := &Rabot{
		embed:  &cc.embed,
		log:    log,
		gross:  cc.Gross,
		Helper: request.NewHelper(log),
		data:   util.NewMonitor[api.Rates](2 * time.Hour),
	}

	if cc.Gross {
		ts, err := rabot.TokenSource(log, cc.Login, cc.Password)
		if err != nil {
			return nil, err
		}

		t.Client.Transport = &oauth2.Transport{
			Source: ts,
			Base:   t.Client.Transport,
		}

		if err := t.fetchContractId(); err != nil {
			return nil, err
		}
	} else {
		t.Client.Transport = transport.BearerAuth(rabot.AppToken, t.Client.Transport)
	}

	return runOrError(t)
}

func (t *Rabot) fetchContractId() error {
	req, err := request.New(http.MethodGet, rabot.BaseURI+"/api/prosumer/v2/contract/list?contractStatus=active", nil, request.AcceptJSON)
	if err != nil {
		return err
	}

	var res rabot.ContractsResponse
	if err := t.DoJSON(req, &res); err != nil {
		return fmt.Errorf("contracts: %w", err)
	}

	if len(res.Contracts) == 0 {
		return fmt.Errorf("no active contracts found")
	}

	t.contractId = res.Contracts[0].ID
	return nil
}

func (t *Rabot) run(done chan error) {
	var once sync.Once

	for tick := time.Tick(time.Hour); ; <-tick {
		var uri string
		if t.gross {
			uri = fmt.Sprintf("%s/api/price-preview/%s", rabot.BaseURI, t.contractId)
		} else {
			uri = rabot.BaseURI + "/api/price-preview"
		}

		var res rabot.PriceResponse
		if err := backoff.Retry(func() error {
			return backoffPermanentError(t.GetJSON(uri, &res))
		}, bo()); err != nil {
			once.Do(func() { done <- err })
			t.log.ERROR.Println(err)
			continue
		}

		data := make(api.Rates, 0, len(res.Records))
		for _, r := range res.Records {
			price := r.PriceNet.Value
			if t.gross {
				price = r.PriceGross.Value
			}

			ar := api.Rate{
				Start: r.Moment,
				End:   r.Moment.Add(15 * time.Minute),
				Value: t.totalPrice(price/100, r.Moment),
			}
			data = append(data, ar)
		}

		mergeRates(t.data, data)
		once.Do(func() { close(done) })
	}
}

func (t *Rabot) Rates() (api.Rates, error) {
	var res api.Rates
	err := t.data.GetFunc(func(val api.Rates) {
		res = slices.Clone(val)
	})
	return res, err
}

func (t *Rabot) Type() api.TariffType {
	return api.TariffTypePriceForecast
}
