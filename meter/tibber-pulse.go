package meter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime/debug"
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

func getUserAgent() string {
	graphqlClientVersion := "unknown"

	if info, ok := debug.ReadBuildInfo(); ok {
		for _, dep := range info.Deps {
			if dep.Path == "github.com/hasura/go-graphql-client" {
				graphqlClientVersion = baseVersion(dep.Version)
			}
		}
	}

	return fmt.Sprintf("evcc/%s hasura/go-graphql-client/%s", util.FormattedVersion(), graphqlClientVersion)
}

func baseVersion(v string) string {
	if i := strings.IndexAny(v, "-+"); i != -1 {
		return v[:i]
	}
	return v
}

type Tibber struct {
	data *util.Monitor[tibber.LiveMeasurement]
}

func NewTibberFromConfig(ctx context.Context, other map[string]interface{}) (api.Meter, error) {
	cc := struct {
		Token   string
		HomeID  string
		Timeout time.Duration
	}{
		Timeout: 2 * time.Minute,
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
		data: util.NewMonitor[tibber.LiveMeasurement](cc.Timeout),
	}

	// subscription client
	client := graphql.NewSubscriptionClient(res.Viewer.WebsocketSubscriptionUrl).
		WithProtocol(graphql.GraphQLWS).
		WithWebSocketOptions(graphql.WebsocketOptions{
			HTTPClient: &http.Client{
				Transport: &transport.Decorator{
					Base: http.DefaultTransport,
					Decorator: transport.DecorateHeaders(map[string]string{
						"User-Agent": getUserAgent(),
					}),
				},
			},
		}).
		WithConnectionParams(map[string]any{
			"token": cc.Token,
		}).
		WithRetryTimeout(15 * time.Second). // Retry 15 seconds (3 tries), then exit to outer retry loop that has backoff
		WithRetryDelay(5 * time.Second).
		WithWriteTimeout(request.Timeout).
		WithReadTimeout(90 * time.Second).
		WithLog(log.TRACE.Println).
		OnConnected(func() {
			log.INFO.Println("Tibber pulse: websocket connected")
		}).
		OnDisconnected(func() {
			log.WARN.Println("Tibber pulse: websocket disconnected")
		}).
		OnSubscriptionComplete(func(_ graphql.Subscription) {
			log.WARN.Println("Tibber pulse: websocket subscription completed by server")
		}).
		OnError(func(sc *graphql.SubscriptionClient, err error) error {
			// Don't let Hasura go graphql client reconnect when authorization fails
			if sc.IsUnauthorized(err) {
				log.ERROR.Printf("Tibber pulse: Unauthorized: %v", err)
				return err
			}
			// Don't let Hasura	go graphql client reconnect when too many initialisation requests
			// Reconnection will be attempted in the loop later
			if sc.IsTooManyInitialisationRequests(err) {
				log.ERROR.Printf("Tibber pulse: Too many initialisation requests: %v", err)
				return err
			}

			log.ERROR.Printf("Tibber pulse: error occurred: %v", err)
			return nil
		})

	done := make(chan error, 1)
	go func(done chan error) {
		done <- t.subscribe(client, cc.HomeID, log)
	}(done)

	select {
	case err := <-done:
		if err != nil {
			return nil, err
		}
	case <-time.After(cc.Timeout):
		return nil, api.ErrTimeout
	}

	go func() {
		<-ctx.Done()
		log.INFO.Println("context canceled, closing Tibber Pulse client")
		if err := client.Close(); err != nil {
			log.ERROR.Printf("error closing Tibber Pulse client: %v", err)
		}
	}()

	reconnectCount := 0
	go func() {
		// The pulse sometimes declines valid(!) subscription requests, and asks the client to disconnect.
		// Therefore we need to restart the client when exiting gracefully upon server request
		// https://github.com/evcc-io/evcc/issues/17925#issuecomment-2621458890

		// Note that there are several reconnection strategies in play:
		// 1. Mechanism built into Hasura go graphql client
		// 2. This loop, which is triggered server when Hasura exits on error or gracefully
		// 3. evcc itself restarts if the client exits with an error

		// Exponential backoff parameters
		baseDelay := 5 * time.Second
		maxDelay := 5 * time.Hour
		delay := baseDelay

		for {
			reconnectCount++
			log.INFO.Printf("Tibber pulse: Hasura go graphql client connection attempt #%d", reconnectCount)

			startTime := time.Now()
			err := client.Run()
			duration := time.Since(startTime).Round(time.Second)
			if err != nil {
				log.ERROR.Printf("Tibber pulse: Hasura go graphql client exited with error at %s: %v", duration, err)
				// Do not retry if unauthorized
				if client.IsUnauthorized(err) {
					log.ERROR.Println("Tibber pulse: Not retrying due to unauthorized error.")
					return
				}
				// Exponential backoff: double the delay, up to maxDelay
				delay *= 2
				if delay > maxDelay {
					delay = maxDelay
				}
			} else {
				log.INFO.Printf("Tibber pulse: Hasura go graphql client exited gracefully at %s", duration)
				// Reset delay after successful connection
				delay = baseDelay
			}

			select {
			case <-time.After(delay):
				log.INFO.Printf("Tibber pulse: Reconnection timer triggered after %s, attempting reconnect", delay)
			case <-ctx.Done():
				log.INFO.Println("Tibber pulse: Context canceled, exit reconnection loop")
				return
			}
		}
	}()

	log.INFO.Printf("!! User-Agent set to %s", getUserAgent())

	return t, nil
}

func (t *Tibber) subscribe(client *graphql.SubscriptionClient, homeID string, log *util.Logger) error {
	var query struct {
		tibber.LiveMeasurement `graphql:"liveMeasurement(homeId: $homeId)"`
	}

	_, err := client.Subscribe(&query, map[string]any{
		"homeId": graphql.ID(homeID),
	}, func(data []byte, err error) error {
		if err != nil {
			log.ERROR.Printf("Tibber pulse: Error during subscription: %v", err)
			return err
		}

		var res struct {
			LiveMeasurement tibber.LiveMeasurement
		}

		if err := json.Unmarshal(data, &res); err != nil {
			log.ERROR.Printf("Tibber pulse: Error unmarshaling data: %v", err)
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
