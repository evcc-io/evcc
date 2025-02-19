package tariff

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/hasura/go-graphql-client"
	"golang.org/x/oauth2"
)

const (
	krakenGraphQLURL = "https://api.oeg-kraken.energy/v1/graphql/"
	goPrice          = 0.10
	standardPrice    = 0.30
)

type OctopusGermany struct {
	log      *util.Logger
	email    string
	password string
	data     *util.Monitor[api.Rates]
}

var _ api.Tariff = (*OctopusGermany)(nil)

func init() {
	registry.Add("octopus-germany", NewOctopusGermanyFromConfig)
}

func NewOctopusGermanyFromConfig(other map[string]interface{}) (api.Tariff, error) {
	var cc struct {
		Email    string
		Password string
	}

	logger := util.NewLogger("octopus-germany")

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	t := &OctopusGermany{
		log:      logger,
		email:    cc.Email,
		password: cc.Password,
		data:     util.NewMonitor[api.Rates](2 * time.Hour),
	}

	done := make(chan error)
	go t.run(done)
	err := <-done

	return t, err
}

func (t *OctopusGermany) run(done chan error) {
	var once sync.Once
	client := request.NewHelper(t.log)

	gqlCli, err := NewClientWithEmailPassword(t.log, t.email, t.password)
	if err != nil {
		once.Do(func() { done <- err })
		t.log.ERROR.Println(err)
		return
	}
	tariffCode, fullName, err := gqlCli.TariffCode()
	if err != nil {
		once.Do(func() { done <- err })
		t.log.ERROR.Println(err)
		return
	}
	t.log.INFO.Printf("Tariff Code: %s, Full Name: %s", tariffCode, fullName)

	for tick := time.Tick(time.Hour); ; <-tick {
		var res struct {
			Results []struct {
				ValidityStart     time.Time `graphql:"validityStart"`
				ValidityEnd       time.Time `graphql:"validityEnd"`
				PriceInclusiveTax float64   `graphql:"priceInclusiveTax"`
			} `graphql:"results"`
		}

		if err := backoff.Retry(func() error {
			return backoffPermanentError(client.GetJSON(krakenGraphQLURL, &res))
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
				Price: t.applyPrice(r.ValidityStart, r.ValidityEnd, r.PriceInclusiveTax/1e2),
			}
			data = append(data, ar)
		}

		mergeRates(t.data, data)
		once.Do(func() { close(done) })
	}
}

func (t *OctopusGermany) applyPrice(start, end time.Time, basePrice float64) float64 {
	startHour := start.UTC().Hour()
	endHour := end.UTC().Hour()

	if (startHour >= 0 && startHour < 5) || (endHour > 0 && endHour <= 5) {
		return goPrice
	}

	if t.isPlannedDispatch(start, end) {
		return goPrice
	}

	return standardPrice
}

func (t *OctopusGermany) isPlannedDispatch(start, end time.Time) bool {
	var dispatches []struct {
		Start time.Time `graphql:"start"`
		End   time.Time `graphql:"end"`
	}

	client := request.NewHelper(t.log)
	query := struct {
		PlannedDispatches []struct {
			Start time.Time `graphql:"start"`
			End   time.Time `graphql:"end"`
		} `graphql:"plannedDispatches(accountNumber: $accountNumber)"`
	}{
		PlannedDispatches: dispatches,
	}

	variables := map[string]interface{}{
		"accountNumber": graphql.String("your_account_number"),
	}

	body, err := json.Marshal(map[string]interface{}{
		"query":     query,
		"variables": variables,
	})
	if err != nil {
		t.log.ERROR.Println(err)
		return false
	}

	resp, err := client.Post(krakenGraphQLURL, "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.log.ERROR.Println(err)
		return false
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&query); err != nil {
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

func (t *OctopusGermany) Rates() (api.Rates, error) {
	var res api.Rates
	err := t.data.GetFunc(func(val api.Rates) {
		res = slices.Clone(val)
	})
	return res, err
}

func (t *OctopusGermany) Type() api.TariffType {
	return api.TariffTypePriceForecast
}

// GraphQL Client Implementation

type OctopusGraphQLClient struct {
	Client          *graphql.Client
	email           string
	password        string
	token           string
	refreshToken    string
	tokenExpiration time.Time
	tokenMtx        sync.Mutex
	accountNumber   string
	log             *util.Logger
}

func NewClientWithEmailPassword(log *util.Logger, email, password string) (*OctopusGraphQLClient, error) {
	httpClient := &http.Client{
		Transport: &logTransport{log: log},
	}
	gq := &OctopusGraphQLClient{
		email:    email,
		password: password,
		log:      log,
	}

	token, refreshToken, tokenExpiration, err := gq.getKrakenToken()
	if err != nil {
		return nil, err
	}

	gq.token = token
	gq.refreshToken = refreshToken
	gq.tokenExpiration = tokenExpiration

	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: gq.token},
	)
	httpClient = oauth2.NewClient(context.Background(), src)
	gq.Client = graphql.NewClient(krakenGraphQLURL, httpClient)

	accountNumber, err := gq.fetchAccountNumber()
	if err != nil {
		return nil, err
	}
	gq.accountNumber = accountNumber

	return gq, nil
}

func (c *OctopusGraphQLClient) getKrakenToken() (string, string, time.Time, error) {
	var mutation struct {
		ObtainKrakenToken struct {
			Token            string `graphql:"token"`
			RefreshToken     string `graphql:"refreshToken"`
			RefreshExpiresIn int64  `graphql:"refreshExpiresIn"`
		} `graphql:"obtainKrakenToken(input: {email: $email, password: $password})"`
	}

	variables := map[string]interface{}{
		"email":    graphql.String(c.email),
		"password": graphql.String(c.password),
	}

	err := c.Client.Mutate(context.Background(), &mutation, variables)
	if err != nil {
		return "", "", time.Time{}, err
	}

	tokenExpiration := time.Now().Add(time.Duration(mutation.ObtainKrakenToken.RefreshExpiresIn) * time.Second)
	return mutation.ObtainKrakenToken.Token, mutation.ObtainKrakenToken.RefreshToken, tokenExpiration, nil
}

func (c *OctopusGraphQLClient) refreshKrakenToken() error {
	c.tokenMtx.Lock()
	defer c.tokenMtx.Unlock()

	if time.Until(c.tokenExpiration) > 5*time.Minute {
		return nil
	}

	var mutation struct {
		RefreshKrakenToken struct {
			Token            string `graphql:"token"`
			RefreshToken     string `graphql:"refreshToken"`
			RefreshExpiresIn int64  `graphql:"refreshExpiresIn"`
		} `graphql:"refreshKrakenToken(input: {refreshToken: $refreshToken})"`
	}

	variables := map[string]interface{}{
		"refreshToken": graphql.String(c.refreshToken),
	}

	err := c.Client.Mutate(context.Background(), &mutation, variables)
	if err != nil {
		return err
	}

	c.token = mutation.RefreshKrakenToken.Token
	c.refreshToken = mutation.RefreshKrakenToken.RefreshToken
	c.tokenExpiration = time.Now().Add(time.Duration(mutation.RefreshKrakenToken.RefreshExpiresIn) * time.Second)
	return nil
}

func (c *OctopusGraphQLClient) fetchAccountNumber() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	var query struct {
		Viewer struct {
			Accounts []struct {
				Number string `graphql:"number"`
			} `graphql:"accounts"`
		} `graphql:"viewer"`
	}

	err := c.Client.Query(ctx, &query, nil)
	if err != nil {
		return "", err
	}

	if len(query.Viewer.Accounts) == 0 {
		return "", errors.New("no account associated with given email and password")
	}
	if len(query.Viewer.Accounts) > 1 {
		return "", errors.New("more than one account on this email and password not supported")
	}
	return query.Viewer.Accounts[0].Number, nil
}

func (c *OctopusGraphQLClient) AccountNumber() (string, error) {
	if c.accountNumber != "" {
		return c.accountNumber, nil
	}

	if err := c.refreshKrakenToken(); err != nil {
		return "", err
	}

	return c.fetchAccountNumber()
}

func (c *OctopusGraphQLClient) TariffCode() (string, string, error) {
	if err := c.refreshKrakenToken(); err != nil {
		return "", "", err
	}

	acc, err := c.AccountNumber()
	if err != nil {
		return "", "", nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	var query struct {
		Account struct {
			AllProperties []struct {
				ElectricityMalos []struct {
					Agreements []struct {
						IsActive                     bool `graphql:"isActive"`
						UnitRateGrossRateInformation []struct {
							GrossRate string `graphql:"grossRate"`
						} `graphql:"unitRateGrossRateInformation"`
						Product struct {
							Code     string `graphql:"code"`
							FullName string `graphql:"fullName"`
						} `graphql:"product"`
					} `graphql:"agreements"`
				} `graphql:"electricityMalos"`
			} `graphql:"allProperties"`
		} `graphql:"account"`
	}

	variables := map[string]interface{}{
		"accountNumber": graphql.String(acc),
	}

	err = c.Client.Query(ctx, &query, variables)
	if err != nil {
		return "", "", err
	}

	for _, property := range query.Account.AllProperties {
		for _, malo := range property.ElectricityMalos {
			for _, agreement := range malo.Agreements {
				if agreement.IsActive {
					c.log.INFO.Printf("Tariff Code: %s, Full Name: %s", agreement.Product.Code, agreement.Product.FullName)
					return agreement.Product.Code, agreement.Product.FullName, nil
				}
			}
		}
	}

	return "", "", errors.New("no active electricity agreements found")
}

type logTransport struct {
	log *util.Logger
}

func (t *logTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.log.TRACE.Printf("Request: %s %s", req.Method, req.URL)
	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		t.log.ERROR.Printf("Response error: %v", err)
	} else {
		t.log.TRACE.Printf("Response: %s %s", resp.Status, req.URL)
	}
	return resp, err
}
