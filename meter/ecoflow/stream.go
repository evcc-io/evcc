package ecoflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/plugin/mqtt"
	"github.com/evcc-io/evcc/util"
)

type Stream struct {
	*Device
	log   *util.Logger
	dataG util.Cacheable[StreamData]

	// MQTT for control (optional)
	client  *mqtt.Client
	account string

	// Battery limits (configurable)
	minSoc   float64
	maxSoc   float64
	capacity float64

	mu          sync.RWMutex
	mqttData    StreamData
	lastMqtt    time.Time
	batteryMode api.BatteryMode
}

var _ api.Meter = (*Stream)(nil)

func NewStreamFromConfig(ctx context.Context, other map[string]any) (api.Meter, error) {
	device, err := NewDevice(other, "ecoflow-stream")
	if err != nil {
		return nil, err
	}

	log := util.NewLogger("ecoflow-stream")

	// Get config values for battery limits
	var cc config
	if err := cc.decode(other); err != nil {
		return nil, err
	}

	s := &Stream{
		Device:      device,
		log:         log,
		minSoc:      cc.MinSoc,
		maxSoc:      cc.MaxSoc,
		capacity:    cc.Capacity,
		batteryMode: api.BatteryNormal,
		dataG: util.ResettableCached(func() (StreamData, error) {
			return FetchQuota[StreamData](device)
		}, device.cache),
	}

	// Setup MQTT for battery control (only for battery usage)
	if device.usage == "battery" {
		if err := s.setupMQTT(); err != nil {
			log.WARN.Printf("MQTT setup failed: %v (battery control disabled)", err)
		}
	}

	if device.usage == "battery" {
		return &StreamBattery{s}, nil
	}
	return s, nil
}

// setupMQTT initializes MQTT connection for control and live updates
func (s *Stream) setupMQTT() error {
	// Get MQTT credentials from certification API
	creds, err := GetMQTTCredentials(s.Helper, s.uri)
	if err != nil {
		return fmt.Errorf("mqtt credentials: %w", err)
	}

	s.account = creds.Account

	// Setup MQTT client
	clientID := fmt.Sprintf("evcc_%s_%d", s.sn, time.Now().UnixNano()%100000)
	mqttConfig := mqtt.Config{
		Broker:   creds.BrokerURL(),
		User:     creds.Account,
		Password: creds.Password,
		ClientID: clientID,
	}

	client, err := mqtt.RegisteredClientOrDefault(s.log, mqttConfig)
	if err != nil {
		return fmt.Errorf("mqtt client: %w", err)
	}

	s.client = client

	// Subscribe to quota topic for live updates
	quotaTopic := fmt.Sprintf("/open/%s/%s/quota", s.account, s.sn)
	if err := client.Listen(quotaTopic, s.handleQuotaMessage); err != nil {
		s.log.WARN.Printf("MQTT subscribe failed: %v", err)
	} else {
		s.log.DEBUG.Printf("subscribed to %s", quotaTopic)
	}

	return nil
}

// handleQuotaMessage processes incoming MQTT messages
func (s *Stream) handleQuotaMessage(payload string) {
	var msg struct {
		Params StreamData `json:"params"`
	}
	if err := json.Unmarshal([]byte(payload), &msg); err != nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// EcoFlow MQTT sends complete data snapshots, not partial updates.
	// Replace mqttData wholesale to correctly handle zero values (e.g., 0 W PV at night).
	s.mqttData = msg.Params
	s.lastMqtt = time.Now()
}

// getData returns current data, preferring MQTT if available
func (s *Stream) getData() (StreamData, error) {
	s.mu.RLock()
	lastMqtt := s.lastMqtt
	mqttData := s.mqttData
	s.mu.RUnlock()

	// Use MQTT data if recent
	if time.Since(lastMqtt) < 60*time.Second {
		return mqttData, nil
	}

	// Fallback to REST API
	return s.dataG.Get()
}

func (s *Stream) CurrentPower() (float64, error) {
	data, err := s.getData()
	if err != nil {
		return 0, err
	}

	switch s.usage {
	case "pv":
		return data.PowGetPvSum, nil
	case "grid":
		return data.PowGetSysGrid, nil
	case "battery":
		return -data.PowGetBpCms, nil
	default:
		return 0, fmt.Errorf("invalid usage: %s", s.usage)
	}
}

// setTopic returns MQTT topic for control
func (s *Stream) setTopic() string {
	return fmt.Sprintf("/open/%s/%s/set", s.account, s.sn)
}

// setRelay controls the AC relay via MQTT
func (s *Stream) setRelay(relay int, state bool) error {
	if s.client == nil {
		return errors.New("mqtt not available")
	}

	key := "relay2Onoff" // AC1
	if relay == 2 {
		key = "relay3Onoff" // AC2
	}

	payload := map[string]any{
		"params": map[string]any{
			key: state,
		},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	s.log.DEBUG.Printf("MQTT publish to %s: %s", s.setTopic(), string(data))
	s.client.Publish(s.setTopic(), false, data)

	return nil
}

// StreamBattery wraps Stream with Battery and BatteryController interfaces
type StreamBattery struct{ *Stream }

var (
	_ api.Battery           = (*StreamBattery)(nil)
	_ api.BatteryController = (*StreamBattery)(nil)
	_ api.BatterySocLimiter = (*StreamBattery)(nil)
)

// Soc implements api.Battery
func (s *StreamBattery) Soc() (float64, error) {
	data, err := s.getData()
	if err != nil {
		return 0, err
	}
	return data.CmsBattSoc, nil
}

// GetSocLimits implements api.BatterySocLimiter
func (s *StreamBattery) GetSocLimits() (float64, float64) {
	minSoc := s.minSoc
	maxSoc := s.maxSoc

	// Use configured values, or try to get from API
	if minSoc == 0 && maxSoc == 0 {
		if data, err := s.getData(); err == nil {
			minSoc = float64(data.CmsMinDsgSoc)
			maxSoc = float64(data.CmsMaxChgSoc)
		}
	}

	// Default fallbacks
	if minSoc == 0 {
		minSoc = 10
	}
	if maxSoc == 0 {
		maxSoc = 100
	}

	return minSoc, maxSoc
}

// Capacity implements api.BatteryCapacity (if configured)
func (s *StreamBattery) Capacity() float64 {
	return s.capacity
}

// SetBatteryMode implements api.BatteryController
// Controls battery discharge via relay
func (s *StreamBattery) SetBatteryMode(mode api.BatteryMode) error {
	if s.client == nil {
		return errors.New("mqtt not available for battery control")
	}

	s.mu.Lock()
	s.batteryMode = mode
	s.mu.Unlock()

	switch mode {
	case api.BatteryNormal:
		// Normal operation - enable both relays
		s.log.DEBUG.Printf("battery mode: normal (relays enabled)")
		if err := s.setRelay(1, true); err != nil {
			return err
		}
		return s.setRelay(2, true)

	case api.BatteryHold:
		// Hold - disable discharge by turning off relays
		s.log.DEBUG.Printf("battery mode: hold (discharge disabled)")
		if err := s.setRelay(1, false); err != nil {
			return err
		}
		return s.setRelay(2, false)

	case api.BatteryCharge:
		// Charge from grid - EcoFlow doesn't support direct grid charging
		// Best effort: enable relays to allow any available charging
		s.log.DEBUG.Printf("battery mode: charge (relays enabled, grid charging not directly supported)")
		if err := s.setRelay(1, true); err != nil {
			return err
		}
		return s.setRelay(2, true)

	default:
		return fmt.Errorf("unsupported battery mode: %v", mode)
	}
}
