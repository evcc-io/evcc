package cardata

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/spf13/cast"
	"golang.org/x/oauth2"
)

const StreamingURL = "tls://customer.streaming-cardata.bmwgroup.com:9000"

// Provider implements the vehicle api
type Provider struct {
	mu  sync.Mutex
	log *util.Logger
	api *API
	ts  oauth2.TokenSource

	vin string

	initial   map[string]TelematicDataPoint
	streaming map[string]StreamingData
}

// NewProvider creates a vehicle api provider
func NewProvider(ctx context.Context, log *util.Logger, api *API, ts oauth2.TokenSource, vin string) *Provider {
	v := &Provider{
		log:       log,
		api:       api,
		ts:        ts,
		vin:       vin,
		streaming: make(map[string]StreamingData),
	}

	go func() {
		bo := backoff.NewExponentialBackOff(backoff.WithMaxInterval(time.Minute))

		for ctx.Err() == nil {
			time.Sleep(bo.NextBackOff())

			token, err := ts.Token()
			if err != nil {
				if !tokenError(err) {
					v.log.ERROR.Println(err)
				}

				continue
			}

			bo.Reset()

			if err := v.runMqtt(ctx, vin, token); err != nil {
				v.log.ERROR.Println(err)
			}
		}
	}()

	return v
}

func (v *Provider) runMqtt(ctx context.Context, vin string, token *oauth2.Token) error {
	gcid := TokenExtra(token, "gcid")
	idToken := TokenExtra(token, "id_token")

	paho := mqtt.NewClient(
		mqtt.NewClientOptions().
			AddBroker(StreamingURL).
			SetAutoReconnect(true).
			SetUsername(gcid).
			SetPassword(idToken))

	timeout := 30 * time.Second
	if t := paho.Connect(); !t.WaitTimeout(timeout) {
		return errors.New("connect timeout")
	} else if err := t.Error(); err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer paho.Disconnect(0)

	topic := fmt.Sprintf("%s/%s", gcid, vin)

	if t := paho.Subscribe(topic, 0, v.handler); !t.WaitTimeout(timeout) {
		return errors.New("subcribe timeout")
	} else if err := t.Error(); err != nil {
		return fmt.Errorf("subscribe: %w", err)
	}

	ctx, cancel := context.WithDeadline(ctx, token.Expiry)
	defer cancel()

	<-ctx.Done()

	return nil
}

func (v *Provider) handler(c mqtt.Client, m mqtt.Message) {
	var res StreamingMessage
	if err := json.Unmarshal(m.Payload(), &res); err != nil {
		v.log.ERROR.Println(m.Topic(), string(m.Payload()), err)
		return
	}

	v.log.TRACE.Println("recv: " + string(m.Payload()))

	v.mu.Lock()
	defer v.mu.Unlock()

	maps.Copy(v.streaming, res.Data)
}

func (v *Provider) any(key string) (any, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if a, ok := v.streaming[key]; ok {
		return a, nil
	}

	if v.initial == nil {
		// don't try as long as there's no token
		if _, err := v.ts.Token(); err != nil {
			return nil, api.ErrNotAvailable
		}

		defer func() {
			if v.initial == nil {
				v.initial = make(map[string]TelematicDataPoint)
			}
		}()

		container, err := v.api.EnsureContainer()
		if err != nil {
			v.log.ERROR.Printf("get container: %v", err)
			return nil, api.ErrNotAvailable
		}

		if res, err := v.api.GetTelematics(v.vin, container); err == nil {
			v.initial = res.TelematicData
		} else {
			v.log.ERROR.Printf("get telematics: %v", err)
			return nil, api.ErrNotAvailable
		}
	}

	if el, ok := v.initial[key]; ok {
		return el.Value, nil
	}

	return nil, api.ErrNotAvailable
}

func (v *Provider) String(key string) (string, error) {
	res, err := v.any(key)
	if err != nil {
		return "", err
	}

	return cast.ToStringE(res)
}

func (v *Provider) Int(key string) (int64, error) {
	res, err := v.any(key)
	if err != nil {
		return 0, err
	}

	return cast.ToInt64E(res)
}

func (v *Provider) Float(key string) (float64, error) {
	res, err := v.any(key)
	if err != nil {
		return 0, err
	}

	return cast.ToFloat64E(res)
}

var _ api.Battery = (*Provider)(nil)

// Soc implements the api.Vehicle interface
func (v *Provider) Soc() (float64, error) {
	return v.Float("vehicle.drivetrain.electricEngine.charging.level")
}

var _ api.ChargeState = (*Provider)(nil)

// Status implements the api.ChargeState interface
func (v *Provider) Status() (api.ChargeStatus, error) {
	port, err := v.String("vehicle.body.chargingPort.status")
	if err != nil {
		return api.StatusNone, err
	}

	status := api.StatusA // disconnected
	if port == "CONNECTED" {
		status = api.StatusB
	}

	hv, err := v.String("vehicle.drivetrain.electricEngine.charging.hvStatus")
	if hv == "CHARGING" {
		status = api.StatusC
	}

	return status, err
}

var _ api.VehicleFinishTimer = (*Provider)(nil)

// FinishTime implements the api.VehicleFinishTimer interface
func (v *Provider) FinishTime() (time.Time, error) {
	res, err := v.Int("vehicle.drivetrain.electricEngine.charging.timeToFullyCharged")
	return time.Now().Add(time.Duration(res) * time.Minute), err
}

var _ api.VehicleRange = (*Provider)(nil)

// Range implements the api.VehicleRange interface
func (v *Provider) Range() (int64, error) {
	return v.Int("vehicle.drivetrain.electricEngine.kombiRemainingElectricRange")
}

var _ api.VehicleOdometer = (*Provider)(nil)

// Odometer implements the api.VehicleOdometer interface
func (v *Provider) Odometer() (float64, error) {
	return v.Float("vehicle.vehicle.travelledDistance")
}

var _ api.SocLimiter = (*Provider)(nil)

// GetLimitSoc implements the api.SocLimiter interface
func (v *Provider) GetLimitSoc() (int64, error) {
	return v.Int("vehicle.powertrain.electric.battery.stateOfCharge.target")
}

var _ api.VehicleClimater = (*Provider)(nil)

// Climater implements the api.VehicleClimater interface
func (v *Provider) Climater() (bool, error) {
	res, err := v.String("vehicle.cabin.hvac.preconditioning.status.comfortState")
	return slices.Contains([]string{"COMFORT_HEATING", "COMFORT_COOLING", "COMFORT_VENTILATION", "DEFROST"}, res), err
}
