package meter

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/meter/tibber"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"github.com/hasura/go-graphql-client"
)

func init() {
	registry.AddCtx("tibber-pulse", NewTibberFromConfig)
}

type Tibber struct {
	data   *util.Monitor[tibber.LiveMeasurement]
	homeID string
}

func NewTibberFromConfig(ctx context.Context, other map[string]interface{}) (api.Meter, error) {
	cc := struct {
		Token   string
		HomeID  string
		Timeout time.Duration
	}{
		Timeout: time.Minute,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.Token == "" {
		return nil, api.ErrMissingToken
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

	ctx2, cancel := context.WithTimeout(ctx, request.Timeout)
	defer cancel()

	if err := qclient.Query(ctx2, &res, nil); err != nil {
		return nil, err
	}

	t := &Tibber{
		data:   util.NewMonitor[tibber.LiveMeasurement](cc.Timeout),
		homeID: cc.HomeID,
	}

	// subscription client
	client := graphql.NewSubscriptionClient(res.Viewer.WebsocketSubscriptionUrl).
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
			"token": cc.Token,
		}).
		WithRetryTimeout(0).
		WithRetryDelay(5 * time.Second).
		WithTimeout(request.Timeout).
		WithLog(log.TRACE.Println).
		OnError(func(_ *graphql.SubscriptionClient, err error) error {
			// exit the subscription client due to unauthorized error
			if strings.Contains(err.Error(), "invalid x-hasura-admin-secret/x-hasura-access-key") {
				return err
			}
			log.ERROR.Println(err)
			return nil
		})

	if err := t.ensureSubscribed(client, cc.Timeout); err != nil {
		return nil, err
	}

	go func() {
		<-ctx.Done()
		if err := client.Close(); err != nil {
			log.ERROR.Println(err)
		}
	}()

	go func() {
		for tick := time.Tick(10 * time.Second); ; {
			err := client.Run()
			if err == nil {
				// The pulse sometimes declines valid(!) subscription requests, and asks the client to disconnect.
				// This invalidates the subscription, and therefore we resubscribe when exiting Run() gracefully
				// upon server request.
				// https://github.com/evcc-io/evcc/issues/17925#issuecomment-2621458890
				err = t.ensureSubscribed(client, cc.Timeout)
			}
			if err != nil {
				log.ERROR.Println(err)
			}

			select {
			case <-tick:
			case <-ctx.Done():
				return
			}
		}
	}()

	return t, nil
}

func (t *Tibber) ensureSubscribed(client *graphql.SubscriptionClient, timeout time.Duration) error {
	done := make(chan error, 1)
	go func(done chan error) {
		done <- t.subscribe(client)
	}(done)

	select {
	case err := <-done:
		return err
	case <-time.After(timeout):
		return api.ErrTimeout
	}
}

func (t *Tibber) subscribe(client *graphql.SubscriptionClient) error {
	var query struct {
		tibber.LiveMeasurement `graphql:"liveMeasurement(homeId: $homeId)"`
	}

	_, err := client.Subscribe(&query, map[string]any{
		"homeId": graphql.ID(t.homeID),
	}, func(data []byte, err error) error {
		if err != nil {
			return err
		}

		var res struct {
			LiveMeasurement tibber.LiveMeasurement
		}

		if err := json.Unmarshal(data, &res); err != nil {
			return err
		}

		t.data.Set(res.LiveMeasurement)

		return nil
	})

	return err
}

func (t *Tibber) CurrentPower() (float64, error) {
	res, err := t.data.Get()
	if err != nil {
		return 0, err
	}

	return res.Power - res.PowerProduction, nil
}

var _ api.PhaseCurrents = (*Tibber)(nil)

// Currents implements the api.PhaseCurrents interface
func (t *Tibber) Currents() (float64, float64, float64, error) {
	res, err := t.data.Get()
	if err != nil {
		return 0, 0, 0, err
	}

	return res.CurrentL1, res.CurrentL2, res.CurrentL3, nil
}
