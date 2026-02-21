package tariff

import (
	"bytes"
	"encoding/json"
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
)

type Rabot struct {
	*embed
	*request.Helper
	log        *util.Logger
	login      string
	password   string
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
		embed `mapstructure:",squash"`
		Login    string
		Password string
		Gross    bool
	}{
		Gross: true,
	}

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
		embed:    &cc.embed,
		log:      log,
		login:    cc.Login,
		password: cc.Password,
		gross:    cc.Gross,
		Helper:   request.NewHelper(log),
		data:     util.NewMonitor[api.Rates](2 * time.Hour),
	}

	if cc.Login != "" && cc.Password != "" {
		sessionToken, err := t.authenticate()
		if err != nil {
			return nil, err
		}

		t.Client.Transport = transport.BearerAuth(sessionToken, t.Client.Transport)
	} else {
		t.Client.Transport = transport.BearerAuth(rabot.AppToken, t.Client.Transport)
	}

	return runOrError(t)
}

func (t *Rabot) authenticate() (string, error) {
	body, err := json.Marshal(struct {
		Login    string `json:"login"`
		Password string `json:"password"`
	}{
		Login:    t.login,
		Password: t.password,
	})
	if err != nil {
		return "", err
	}

	req, err := request.New(http.MethodPost, rabot.BaseURI+"/api/prosumer/session/login", bytes.NewReader(body), map[string]string{
		"Authorization": "Bearer " + rabot.AppToken,
		"Content-Type":  request.JSONContent,
		"Accept":        request.JSONContent,
	})
	if err != nil {
		return "", err
	}

	client := request.NewHelper(t.log)
	var loginRes rabot.LoginResponse
	if err := client.DoJSON(req, &loginRes); err != nil {
		return "", fmt.Errorf("login: %w", err)
	}

	t.log.Redact(loginRes.SessionToken)

	// fetch contract ID
	req, err = request.New(http.MethodGet, rabot.BaseURI+"/api/prosumer/v2/contract/list?contractStatus=active", nil, map[string]string{
		"Authorization": "Bearer " + loginRes.SessionToken,
		"Accept":        request.JSONContent,
	})
	if err != nil {
		return "", err
	}

	var contractsRes rabot.ContractsResponse
	if err := client.DoJSON(req, &contractsRes); err != nil {
		return "", fmt.Errorf("contracts: %w", err)
	}

	if len(contractsRes.Contracts) == 0 {
		return "", fmt.Errorf("no active contracts found")
	}

	t.contractId = contractsRes.Contracts[0].ID

	return loginRes.SessionToken, nil
}

func (t *Rabot) run(done chan error) {
	var once sync.Once

	for tick := time.Tick(time.Hour); ; <-tick {
		var uri string
		if t.gross {
			sessionToken, err := t.authenticate()
			if err != nil {
				once.Do(func() { done <- err })
				t.log.ERROR.Println(err)
				continue
			}

			t.Client.Transport = transport.BearerAuth(sessionToken, http.DefaultTransport)
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
