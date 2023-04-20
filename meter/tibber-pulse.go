package meter

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/meter/tibber"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"github.com/hasura/go-graphql-client"
)

func init() {
	registry.Add("tibber-pulse", NewTibberFromConfig)
}

var timeout = time.Minute

type Tibber struct {
	mu            sync.Mutex
	log           *util.Logger
	updated       time.Time
	live          tibber.LiveMeasurement
	url           string
	token, homeID string
	client        *graphql.SubscriptionClient
}

func NewTibberFromConfig(other map[string]interface{}) (api.Meter, error) {
	var cc struct {
		Token  string
		HomeID string
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.Token == "" {
		return nil, errors.New("missing token")
	}

	log := util.NewLogger("pulse").Redact(cc.Token, cc.HomeID)

	// query client
	qclient := tibber.NewClient(log, cc.Token)

	if cc.HomeID == "" {
		home, err := qclient.DefaultHome("")
		if err != nil {
			return nil, err
		}
		cc.HomeID = home.ID
	}

	var res struct {
		Viewer struct {
			WebsocketSubscriptionUrl string
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), request.Timeout)
	defer cancel()

	if err := qclient.Query(ctx, &res, nil); err != nil {
		return nil, err
	}

	t := &Tibber{
		log:    log,
		url:    res.Viewer.WebsocketSubscriptionUrl,
		token:  cc.Token,
		homeID: cc.HomeID,
	}

	// run the client
	done := make(chan error)
	t.newSubscriptionClient()
	go t.subscribe(done)
	err := <-done

	return t, err
}

// newSubscriptionClient creates graphql subscription client
func (t *Tibber) newSubscriptionClient() {
	t.client = graphql.NewSubscriptionClient(t.url).
		WithProtocol(graphql.GraphQLWS).
		WithWebSocketOptions(graphql.WebsocketOptions{
			HTTPClient: &http.Client{
				Transport: &transport.Decorator{
					Base: http.DefaultTransport,
					Decorator: transport.DecorateHeaders(map[string]string{
						"User-Agent": "go-graphql-client/0.9.0",
					}),
				},
			},
		}).
		WithConnectionParams(map[string]any{
			"token": t.token,
		}).
		WithRetryTimeout(0).
		WithLog(t.log.TRACE.Println)
}

func (t *Tibber) subscribe(done chan error) {
	var query struct {
		tibber.LiveMeasurement `graphql:"liveMeasurement(homeId: $homeId)"`
	}

	var once sync.Once

	_, err := t.client.Subscribe(&query, map[string]any{
		"homeId": graphql.ID(t.homeID),
	}, func(data []byte, err error) error {
		if err != nil {
			once.Do(func() { done <- err })
		}

		var res struct {
			LiveMeasurement tibber.LiveMeasurement
		}

		if err := json.Unmarshal(data, &res); err != nil {
			once.Do(func() { done <- err })

			t.log.ERROR.Println(err)
			return nil
		}

		t.mu.Lock()
		t.live = res.LiveMeasurement
		t.updated = time.Now()
		t.mu.Unlock()

		once.Do(func() { close(done) })

		return nil
	})

	if err != nil {
		once.Do(func() { done <- err })
	}

	go func() {
		if err := t.client.Run(); err != nil {
			once.Do(func() { done <- err })
		}
	}()
}

func (t *Tibber) restart() error {
	_ = t.client.Close()

	done := make(chan error)
	t.newSubscriptionClient()
	go t.subscribe(done)
	return <-done
}

func (t *Tibber) CurrentPower() (float64, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// if time.Since(t.updated) > timeout && t.restart() != nil {
	if t.restart() != nil {
		return 0, api.ErrTimeout
	}

	return t.live.Power - t.live.PowerProduction, nil
}

var _ api.PhaseCurrents = (*Tibber)(nil)

// Currents implements the api.PhaseCurrents interface
func (t *Tibber) Currents() (float64, float64, float64, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if time.Since(t.updated) > timeout && t.restart() != nil {
		return 0, 0, 0, api.ErrTimeout
	}

	return t.live.CurrentL1, t.live.CurrentL2, t.live.CurrentL3, nil
}
