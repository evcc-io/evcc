package charger

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/meter/ecoflow"
	"github.com/evcc-io/evcc/plugin/mqtt"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

func init() {
	registry.AddCtx("ecoflow-stream-charger", NewEcoFlowStreamChargerFromConfig)
}

// EcoFlowStreamCharger implements api.Charger for EcoFlow Stream devices
// Controls charging/discharging via relay switches over MQTT
type EcoFlowStreamCharger struct {
	log     *util.Logger
	dataG   func() (ecoflow.StreamData, error)
	client  *mqtt.Client
	account string
	sn      string
	relay   int // 1 = AC1 (relay2), 2 = AC2 (relay3)

	mu      sync.Mutex
	enabled bool
}

// NewEcoFlowStreamChargerFromConfig creates EcoFlow Stream charger from config
func NewEcoFlowStreamChargerFromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
	cc := struct {
		// EcoFlow API credentials
		URI       string
		SN        string
		AccessKey string
		SecretKey string
		Cache     time.Duration

		// Relay selection
		Relay int // 1 = AC1, 2 = AC2
	}{
		URI:   "https://api-e.ecoflow.com",
		Cache: 10 * time.Second,
		Relay: 1, // Default to AC1
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.SN == "" || cc.AccessKey == "" || cc.SecretKey == "" {
		return nil, fmt.Errorf("missing sn, accessKey or secretKey")
	}

	log := util.NewLogger("ecoflow-charger").Redact(cc.AccessKey, cc.SecretKey)

	// Create HTTP client with auth transport
	helper := request.NewHelper(log)
	helper.Client.Transport = ecoflow.NewAuthTransport(helper.Client.Transport, cc.AccessKey, cc.SecretKey)

	uri := strings.TrimSuffix(cc.URI, "/")

	// Get MQTT credentials from certification API
	mqttCreds, err := ecoflow.GetMQTTCredentials(helper, uri)
	if err != nil {
		return nil, fmt.Errorf("failed to get mqtt credentials: %w", err)
	}

	log.DEBUG.Printf("MQTT credentials: account=%s, broker=%s", mqttCreds.Account, mqttCreds.BrokerURL())

	// Setup MQTT client
	mqttConfig := mqtt.Config{
		Broker:   mqttCreds.BrokerURL(),
		User:     mqttCreds.Account,
		Password: mqttCreds.Password,
	}

	client, err := mqtt.RegisteredClientOrDefault(log, mqttConfig)
	if err != nil {
		return nil, fmt.Errorf("mqtt: %w", err)
	}

	// Create cached data getter for status reads
	quotaURL := fmt.Sprintf("%s/iot-open/sign/device/quota/all?sn=%s", uri, cc.SN)
	dataG := util.Cached(func() (ecoflow.StreamData, error) {
		var res struct {
			Code    string            `json:"code"`
			Message string            `json:"message"`
			Data    ecoflow.StreamData `json:"data"`
		}
		if err := helper.GetJSON(quotaURL, &res); err != nil {
			return ecoflow.StreamData{}, err
		}
		if res.Code != "0" {
			return ecoflow.StreamData{}, fmt.Errorf("api error: %s - %s", res.Code, res.Message)
		}
		return res.Data, nil
	}, cc.Cache)

	// Read initial state
	data, err := dataG()
	if err != nil {
		log.WARN.Printf("failed to read initial state: %v", err)
	}

	c := &EcoFlowStreamCharger{
		log:     log,
		dataG:   dataG,
		client:  client,
		account: mqttCreds.Account,
		sn:      cc.SN,
		relay:   cc.Relay,
		enabled: relayState(data, cc.Relay),
	}

	return c, nil
}

// relayState returns the relay state from data
func relayState(data ecoflow.StreamData, relay int) bool {
	if relay == 2 {
		return data.Relay3Onoff
	}
	return data.Relay2Onoff
}

// setTopic returns the MQTT topic for setting device parameters
func (c *EcoFlowStreamCharger) setTopic() string {
	return fmt.Sprintf("/open/%s/%s/set", c.account, c.sn)
}

// relayKey returns the relay parameter name
func (c *EcoFlowStreamCharger) relayKey() string {
	if c.relay == 2 {
		return "relay3Onoff" // AC2
	}
	return "relay2Onoff" // AC1
}

// Status implements api.Charger
func (c *EcoFlowStreamCharger) Status() (api.ChargeStatus, error) {
	data, err := c.dataG()
	if err != nil {
		return api.StatusNone, err
	}

	enabled := relayState(data, c.relay)

	// Update cached state
	c.mu.Lock()
	c.enabled = enabled
	c.mu.Unlock()

	if !enabled {
		return api.StatusA, nil // Relay off = not connected
	}

	// Check power flow (positive = charging in evcc convention)
	// EcoFlow: negative = discharge, positive = charge
	if data.PowGetBpCms > 50 { // threshold for noise
		return api.StatusC, nil // Charging
	}

	return api.StatusB, nil // Connected but not charging
}

// Enabled implements api.Charger
func (c *EcoFlowStreamCharger) Enabled() (bool, error) {
	data, err := c.dataG()
	if err != nil {
		// Fall back to cached state
		c.mu.Lock()
		defer c.mu.Unlock()
		return c.enabled, nil
	}

	enabled := relayState(data, c.relay)

	c.mu.Lock()
	c.enabled = enabled
	c.mu.Unlock()

	return enabled, nil
}

// Enable implements api.Charger - controls relay via MQTT
func (c *EcoFlowStreamCharger) Enable(enable bool) error {
	payload := map[string]interface{}{
		"params": map[string]interface{}{
			c.relayKey(): enable,
		},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	c.log.DEBUG.Printf("MQTT publish to %s: %s", c.setTopic(), string(data))

	c.client.Publish(c.setTopic(), false, data)

	c.mu.Lock()
	c.enabled = enable
	c.mu.Unlock()

	return nil
}

// MaxCurrent implements api.Charger - EcoFlow Stream doesn't support current control
func (c *EcoFlowStreamCharger) MaxCurrent(current int64) error {
	// EcoFlow Stream doesn't support current limiting
	return nil
}

// CurrentPower implements api.Meter
func (c *EcoFlowStreamCharger) CurrentPower() (float64, error) {
	data, err := c.dataG()
	if err != nil {
		return 0, err
	}
	// Return battery power (positive = charging)
	return data.PowGetBpCms, nil
}

// Soc implements api.Battery
func (c *EcoFlowStreamCharger) Soc() (float64, error) {
	data, err := c.dataG()
	if err != nil {
		return 0, err
	}
	return data.CmsBattSoc, nil
}

var (
	_ api.Charger = (*EcoFlowStreamCharger)(nil)
	_ api.Meter   = (*EcoFlowStreamCharger)(nil)
	_ api.Battery = (*EcoFlowStreamCharger)(nil)
)
