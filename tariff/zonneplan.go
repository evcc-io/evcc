package tariff

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/plugin/auth"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	zonneplan "github.com/jap/go-zonneplan"
	"golang.org/x/oauth2"
)

// Zonneplan is a dynamic tariff provider. Login happens in the ui via evcc's
// provider auth (mailed one-time password); the rotating token is shared with
// every device using the same account.

type Zonneplan struct {
	*embed
	log        *util.Logger
	export     bool
	last       api.Rates // served without waiting for the first update
	ts         oauth2.TokenSource
	client     *zonneplan.Client
	mu         sync.Mutex
	address    string
	connection string
	sunrise    time.Time
	sunset     time.Time
	discovered bool
}

var _ api.Tariff = (*Zonneplan)(nil)

func init() {
	registry.AddCtx("zonneplan", NewZonneplanFromConfig)
}

func NewZonneplanFromConfig(ctx context.Context, other map[string]any) (api.Tariff, error) {
	cc := struct {
		embed           `mapstructure:",squash"`
		Email           string
		Address         string
		Connection      string
		TariffDirection string
	}{}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.Email == "" {
		return nil, api.ErrMissingCredentials
	}

	var export bool
	switch strings.ToLower(cc.TariffDirection) {
	case "", "import":
	case "export":
		export = true
	default:
		return nil, errors.New("invalid tariff direction: " + cc.TariffDirection)
	}

	if err := cc.init(); err != nil {
		return nil, err
	}

	ts, err := auth.NewFromConfig(ctx, "zonneplan", map[string]any{
		"email": cc.Email,
	})
	if err != nil {
		return nil, err
	}

	client := zonneplan.NewClient(zonneplan.WithHTTPClient(&http.Client{
		Timeout: request.Timeout,
		Transport: &oauth2.Transport{
			Source: ts,
			Base:   http.DefaultTransport,
		},
	}))

	t := &Zonneplan{
		embed:      &cc.embed,
		log:        util.NewLogger("zonneplan").Redact(cc.Email),
		export:     export,
		ts:         ts,
		client:     client,
		address:    cc.Address,
		connection: cc.Connection,
	}

	return runOrError(t)
}

func (t *Zonneplan) run(done chan error) {
	var once sync.Once

	// poll every minute while empty (waiting for login or the first update),
	// hourly once rates are flowing
	delay := time.Minute

	for {
		<-time.After(delay)

		// devices do not block on login: start empty, retry until the ui
		// login succeeds
		if _, err := t.ts.Token(); err != nil {
			delay = time.Minute

			if _, ok := errors.AsType[*api.ErrLoginRequired](err); ok {
				t.log.WARN.Println("waiting for login:", err)
				once.Do(func() { close(done) })
				continue
			}

			if reportError(&once, done, err) {
				return
			}
			t.log.ERROR.Println(err)
			continue
		}

		var res api.Rates
		if err := backoff.Retry(func() error {
			var err error
			res, err = t.rates()
			return backoffPermanentError(err)
		}, bo()); err != nil {
			if reportError(&once, done, err) {
				return
			}
			t.log.ERROR.Println(err)
			continue
		}

		t.mu.Lock()
		t.last = res
		t.mu.Unlock()

		delay = time.Hour

		once.Do(func() { close(done) })
	}
}

// rates fetches the quarter-hourly consumer price chart, which covers
// yesterday through tomorrow in a single call.
func (t *Zonneplan) rates() (api.Rates, error) {
	if err := t.discover(); err != nil {
		return nil, err
	}

	prices, err := t.client.ConsumerPrices(context.Background(), zonneplan.ChartElectricityQuarterHourly, t.connection)
	if err != nil {
		return nil, err
	}

	return t.mapRates(prices.Prices), nil
}

// discover resolves the electricity connection uuid and the Zonnebonus sun
// window from the account. It retries until it succeeds, so a temporarily
// failing account does not block the price updates forever.
func (t *Zonneplan) discover() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.discovered {
		return nil
	}

	account, err := t.client.GetAccount(context.Background())
	if err != nil {
		return err
	}

	group, err := resolveGroup(account, t.connection, t.address)
	if err != nil {
		return err
	}

	if t.connection == "" {
		t.connection = group.ElectricityUUIDs()[0]
	}
	t.sunrise, t.sunset = group.Address.Sunrise, group.Address.Sunset
	t.discovered = true

	return nil
}

// resolveGroup picks the address group to use: by connection uuid, by address
// text, or the only group with electricity connections.
func resolveGroup(account *zonneplan.Account, connection, address string) (zonneplan.AddressGroup, error) {
	if connection != "" {
		for _, g := range account.AddressGroups {
			if slices.Contains(g.ElectricityUUIDs(), connection) {
				return g, nil
			}
		}
	}

	if address != "" {
		if g, ok := account.MatchAddress(address); ok {
			return g, nil
		}
		return zonneplan.AddressGroup{}, fmt.Errorf("no address matches %q, available: %s", address, addressOptions(account))
	}

	groups := account.ElectricityGroups()
	switch len(groups) {
	case 1:
		return groups[0], nil
	case 0:
		return zonneplan.AddressGroup{}, errors.New("no electricity connection in the account")
	default:
		return zonneplan.AddressGroup{}, fmt.Errorf("multiple addresses, set the address parameter to one of: %s", addressOptions(account))
	}
}

func addressOptions(account *zonneplan.Account) string {
	var opts []string
	for _, g := range account.ElectricityGroups() {
		opts = append(opts, g.Address.String())
	}
	return strings.Join(opts, "; ")
}

// mapRates converts the price points. Import returns the all-in price
// (energy tax and VAT included). Export returns the feed-in estimate with
// the Zonnebonus and, while the salderingsregeling runs, the energy tax
// refund; use `charges`, `tax` or `formula` to adjust.
func (t *Zonneplan) mapRates(prices []zonneplan.PricePoint) api.Rates {
	res := make(api.Rates, 0, len(prices))

	for _, p := range prices {
		price := p.AllIn()
		if t.export || t.Charges != 0 || len(t.chargesZones) > 0 || t.Tax != 0 || t.Formula != "" {
			if t.export {
				price = t.totalPrice(p.FeedInRateInclusive(t.sunrise, t.sunset), p.StartDate.Local())
			} else {
				price = t.totalPrice(p.PriceTaxExcluded.Euros(), p.StartDate.Local())
			}
		}

		res = append(res, api.Rate{
			Start: p.StartDate.Local(),
			End:   p.EndDate.Local(),
			Value: price,
		})
	}

	res.Sort()
	return res
}

// Rates implements the api.Tariff interface. It must not block the config
// validation, so the rates are empty until the first update after the ui login.
func (t *Zonneplan) Rates() (api.Rates, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return slices.Clone(t.last), nil
}

// Type implements the api.Tariff interface
func (t *Zonneplan) Type() api.TariffType {
	return api.TariffTypePriceForecast
}
