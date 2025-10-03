package cardata

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
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

	vin, container string

	initial   map[string]TelematicDataPoint
	streaming map[string]StreamingData
}

// NewProvider creates a vehicle api provider
func NewProvider(log *util.Logger, api *API, ts oauth2.TokenSource, vin, container string) *Provider {
	v := &Provider{
		log:       log,
		api:       api,
		vin:       vin,
		container: container,
		streaming: make(map[string]StreamingData),
	}

	go func() {
		bo := backoff.NewExponentialBackOff(backoff.WithMaxInterval(time.Minute))

		token, err := ts.Token()
		if err != nil {
			if !errors.Is(err, ErrLoginRequired) {
				v.log.ERROR.Println(err)
			}

			time.Sleep(bo.NextBackOff())
		}

		bo.Reset()

		if err := v.runMqtt(vin, token); err != nil {
			v.log.ERROR.Println(err)
		}
	}()

	return v
}

func (v *Provider) runMqtt(vin string, token *oauth2.Token) error {
	gcid := tokenExtra(token, "gcid")
	idToken := tokenExtra(token, "id_token")

	o := mqtt.NewClientOptions().
		AddBroker(StreamingURL).
		SetAutoReconnect(true).
		SetUsername(gcid).
		SetPassword(idToken)

	paho := mqtt.NewClient(o)

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

	time.Sleep(time.Until(token.Expiry))

	return nil
}

func (v *Provider) handler(c mqtt.Client, m mqtt.Message) {
	var res StreamingMessage
	if err := json.Unmarshal(m.Payload(), &res); err != nil {
		v.log.ERROR.Println(m.Topic(), string(m.Payload()), err)
		return
	}

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
		res, err := v.api.GetTelematics(v.container)
		if err != nil {
			return nil, fmt.Errorf("get telematics: %w", err)
		}
		v.initial = res.TelematicData
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

// var _ api.ChargeState = (*Provider)(nil)

// // Status implements the api.ChargeState interface
// func (v *Provider) Status() (api.ChargeStatus, error) {
// 	res, err := v.statusG()
// 	if err != nil {
// 		return api.StatusNone, err
// 	}

// 	status := api.StatusA // disconnected
// 	if res.State.ElectricChargingState.IsChargerConnected {
// 		status = api.StatusB
// 	}
// 	if res.State.ElectricChargingState.ChargingStatus == "CHARGING" {
// 		status = api.StatusC
// 	}

// 	return status, nil
// }

// var _ api.VehicleFinishTimer = (*Provider)(nil)

// // FinishTime implements the api.VehicleFinishTimer interface
// func (v *Provider) FinishTime() (time.Time, error) {
// 	res, err := v.statusG()
// err == nil {
// 		ctr := res.VehicleStatus.ChargingTimeRemaining
// 		return time.Now().Add(time.Duration(ctr) * time.Minute), err
// 	}

// 	return time.Time{}, err
// }

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

// var _ api.SocLimiter = (*Provider)(nil)

// // GetLimitSoc implements the api.SocLimiter interface
// func (v *Provider) GetLimitSoc() (int64, error) {
// 	res, err := v.statusG()
// 	if err != nil {
// 		return 0, err
// 	}

// 	return res.State.ElectricChargingState.ChargingTarget, nil
// }

// var _ api.VehicleClimater = (*Provider)(nil)

// // Climater implements the api.VehicleClimater interface
// func (v *Provider) Climater() (bool, error) {
// 	res, err := v.statusG()
// 	return res.State.ClimateControlState.Activity == "HEATING" || res.State.ClimateControlState.Activity == "COOLING", err
// }
