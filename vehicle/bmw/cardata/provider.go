package cardata

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/samber/lo"
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

	vin       string
	container string

	rest      map[string]TelematicData
	streaming map[string]StreamingData
	updated   time.Time
	cache     time.Duration
}

// NewProvider creates a vehicle api provider
func NewProvider(ctx context.Context, log *util.Logger, api *API, ts oauth2.TokenSource, clientID, vin string, cache time.Duration) *Provider {
	v := &Provider{
		log:       log,
		api:       api,
		ts:        ts,
		vin:       vin,
		cache:     cache,
		rest:      make(map[string]TelematicData),
		streaming: make(map[string]StreamingData),
	}

	mqtt := NewMqttConnector(context.Background(), log, clientID, ts)
	recvC := mqtt.Subscribe(vin)

	go func() {
		<-ctx.Done()
		mqtt.Unsubscribe(vin)
	}()

	go func() {
		for msg := range recvC {
			v.mu.Lock()
			maps.Copy(v.streaming, msg.Data)
			v.updated = time.Now()
			v.mu.Unlock()
		}
	}()

	return v
}

func (v *Provider) findOrCreateContainer() (string, error) {
	containers, err := v.api.GetContainers()
	if err != nil {
		return "", err
	}

	if cc := lo.Filter(containers, func(c Container, _ int) bool {
		return c.Name == "evcc.io" && c.Purpose == requiredVersion
	}); len(cc) > 0 {
		return cc[0].ContainerId, nil
	}

	res, err := v.api.CreateContainer(CreateContainer{
		Name:                 "evcc.io",
		Purpose:              requiredVersion,
		TechnicalDescriptors: requiredKeys,
	})

	return res.ContainerId, err
}

func (v *Provider) setupContainer() error {
	container, err := v.findOrCreateContainer()
	if err != nil {
		return fmt.Errorf("get container: %v", err)
	}

	v.container = container

	return nil
}

func (v *Provider) updateContainerData() error {
	res, err := v.api.GetTelematics(v.vin, v.container)
	if err != nil {
		return fmt.Errorf("get telematics: %v", err)
	}

	v.rest = res.TelematicData
	v.streaming = make(map[string]StreamingData) // reset streaming

	return nil
}

func (v *Provider) any(key string) (any, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	_, tokenErr := v.ts.Token()

	switch {
	case tokenErr == nil && v.updated.IsZero():
		// this will only happen once
		if err := v.setupContainer(); err != nil {
			v.log.WARN.Println(err)
		}
		fallthrough

	case tokenErr == nil && time.Since(v.updated) > v.cache && v.container != "":
		if err := v.updateContainerData(); err != nil {
			v.log.WARN.Println(err)
		}
		v.updated = time.Now()
	}

	if a, ok := v.streaming[key]; ok {
		return a.Value, nil
	}

	if el, ok := v.rest[key]; ok {
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
	return v.Float("vehicle.drivetrain.batteryManagement.header")
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
	if err != nil || hv == "" || hv == "INVALID" {
		hv, err = v.String("vehicle.drivetrain.electricEngine.charging.status")
	}

	if slices.Contains([]string{
		"CHARGING",       // vehicle.drivetrain.electricEngine.charging.hvStatus
		"CHARGINGACTIVE", // vehicle.drivetrain.electricEngine.charging.status
	}, hv) {
		return api.StatusC, nil
	}

	return status, err
}

var _ api.VehicleFinishTimer = (*Provider)(nil)

// FinishTime implements the api.VehicleFinishTimer interface
func (v *Provider) FinishTime() (time.Time, error) {
	res, err := v.Int("vehicle.drivetrain.electricEngine.charging.timeRemaining")
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
	if err == nil && res != "" {
		return slices.Contains([]string{"COMFORT_HEATING", "COMFORT_COOLING", "COMFORT_VENTILATION", "DEFROST"}, res), nil
	}

	if res, err = v.String("vehicle.vehicle.preConditioning.activity"); err == nil {
		return slices.Contains([]string{"HEATING", "COOLING", "VENTILATION"}, res), nil
	}

	return false, err
}
