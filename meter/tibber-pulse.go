package meter

import (
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/meter/tibber"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/hasura/go-graphql-client"
)

func init() {
	registry.Add("tibber-pulse", NewTibberFromConfig)
}

type Tibber struct {
	mu      sync.Mutex
	log     *util.Logger
	updated time.Time
	live    tibber.LiveMeasurement
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

	t := &Tibber{
		log: util.NewLogger("pulse").Redact(cc.Token, cc.HomeID),
	}

	if cc.HomeID == "" {
		// query client
		qclient := tibber.NewClient(t.log, cc.Token)

		var err error
		if cc.HomeID, err = qclient.DefaultHomeID(); err != nil {
			return nil, err
		}
	}

	// subscription client
	client := graphql.NewSubscriptionClient(tibber.SubscriptionURI).
		WithConnectionParams(map[string]any{
			"token": cc.Token,
		}).
		WithLog(t.log.TRACE.Println)

	// run the client
	go func() {
		if err := client.Run(); err != nil {
			t.log.ERROR.Println(err)
		}
	}()

	err := t.subscribe(client, cc.HomeID)

	return t, err
}

// subscribe to the websocket query
func (t *Tibber) subscribe(client *graphql.SubscriptionClient, homeID string) error {
	var query struct {
		tibber.LiveMeasurement `graphql:"liveMeasurement(homeId: $homeId)"`
	}

	var once sync.Once
	recv := make(chan struct{})
	errC := make(chan error)

	_, err := client.Subscribe(&query, map[string]any{
		"homeId": graphql.ID(homeID),
	}, func(data []byte, err error) error {
		if err != nil {
			select {
			case errC <- err:
			default:
			}
			return nil
		}

		var res struct {
			LiveMeasurement tibber.LiveMeasurement
		}

		if err := json.Unmarshal(data, &res); err != nil {
			t.log.ERROR.Println(err)
			return nil
		}

		once.Do(func() {
			close(recv)
		})

		t.mu.Lock()
		t.live = res.LiveMeasurement
		t.updated = time.Now()
		t.mu.Unlock()

		return nil
	})

	// wait for connection
	if err == nil {
		select {
		case <-recv:
		case <-time.After(request.Timeout):
			err = api.ErrTimeout
		case err = <-errC:
		}
	}

	return err
}

// CurrentPower implements the api.Meter interface
func (t *Tibber) CurrentPower() (float64, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if time.Since(t.updated) > request.Timeout {
		return 0, api.ErrTimeout
	}

	return t.live.Power, nil
}

var _ api.MeterCurrent = (*Tibber)(nil)

// Currents implements the api.MeterCurrent interface
func (t *Tibber) Currents() (float64, float64, float64, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if time.Since(t.updated) > request.Timeout {
		return 0, 0, 0, api.ErrTimeout
	}

	return t.live.CurrentL1, t.live.CurrentL2, t.live.CurrentL3, nil
}
