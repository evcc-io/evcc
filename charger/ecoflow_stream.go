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
	client  *mqtt.Client
	account string
	sn      string
	relay   int // 1 = AC1 (relay2), 2 = AC2 (relay3)

	mu   sync.RWMutex
	data ecoflow.StreamData // live data from MQTT subscription

	// Fallback REST API getter (used if MQTT data is stale)
	restDataG func() (ecoflow.StreamData, error)
	lastMqtt  time.Time
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

	// Setup MQTT client with unique client ID
	clientID := fmt.Sprintf("evcc_%s_%d", cc.SN, time.Now().UnixNano()%100000)
	mqttConfig := mqtt.Config{
		Broker:   mqttCreds.BrokerURL(),
		User:     mqttCreds.Account,
		Password: mqttCreds.Password,
		ClientID: clientID,
	}

	client, err := mqtt.RegisteredClientOrDefault(log, mqttConfig)
	if err != nil {
		return nil, fmt.Errorf("mqtt: %w", err)
	}

	// Create REST API fallback getter
	quotaURL := fmt.Sprintf("%s/iot-open/sign/device/quota/all?sn=%s", uri, cc.SN)
	restDataG := util.Cached(func() (ecoflow.StreamData, error) {
		var res struct {
			Code    string             `json:"code"`
			Message string             `json:"message"`
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

	c := &EcoFlowStreamCharger{
		log:       log,
		client:    client,
		account:   mqttCreds.Account,
		sn:        cc.SN,
		relay:     cc.Relay,
		restDataG: restDataG,
	}

	// Subscribe to MQTT topics for live updates
	if err := c.subscribe(); err != nil {
		log.WARN.Printf("MQTT subscribe failed: %v (falling back to REST API)", err)
	}

	// Read initial state via REST API
	if data, err := restDataG(); err == nil {
		c.mu.Lock()
		c.data = data
		c.mu.Unlock()
	} else {
		log.WARN.Printf("failed to read initial state: %v", err)
	}

	return c, nil
}

// subscribe sets up MQTT subscriptions for live data
func (c *EcoFlowStreamCharger) subscribe() error {
	// Subscribe to quota topic for live data updates
	quotaTopic := fmt.Sprintf("/open/%s/%s/quota", c.account, c.sn)
	if err := c.client.Listen(quotaTopic, c.handleQuotaMessage); err != nil {
		return fmt.Errorf("subscribe quota: %w", err)
	}
	c.log.DEBUG.Printf("subscribed to %s", quotaTopic)

	// Subscribe to status topic for online/offline notifications
	statusTopic := fmt.Sprintf("/open/%s/%s/status", c.account, c.sn)
	if err := c.client.Listen(statusTopic, c.handleStatusMessage); err != nil {
		c.log.WARN.Printf("subscribe status failed: %v", err)
		// Non-fatal, quota topic is more important
	} else {
		c.log.DEBUG.Printf("subscribed to %s", statusTopic)
	}

	return nil
}

// mqttMessage wraps incoming MQTT data
type mqttMessage struct {
	Params ecoflow.StreamData `json:"params"`
}

// handleQuotaMessage processes incoming quota messages from MQTT
func (c *EcoFlowStreamCharger) handleQuotaMessage(payload string) {
	var msg mqttMessage
	if err := json.Unmarshal([]byte(payload), &msg); err != nil {
		c.log.TRACE.Printf("quota parse error: %v", err)
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Merge incoming data (MQTT sends partial updates)
	c.mergeData(&msg.Params)
	c.lastMqtt = time.Now()

	c.log.TRACE.Printf("MQTT update: SOC=%.1f%%, Power=%.1fW, Relay1=%v, Relay2=%v",
		c.data.CmsBattSoc, c.data.PowGetBpCms, c.data.Relay2Onoff, c.data.Relay3Onoff)
}

// mergeData merges partial MQTT updates into current data
func (c *EcoFlowStreamCharger) mergeData(update *ecoflow.StreamData) {
	// Only update non-zero values (MQTT sends partial updates)
	if update.CmsBattSoc != 0 {
		c.data.CmsBattSoc = update.CmsBattSoc
	}
	if update.PowGetBpCms != 0 {
		c.data.PowGetBpCms = update.PowGetBpCms
	}
	if update.PowGetPvSum != 0 {
		c.data.PowGetPvSum = update.PowGetPvSum
	}
	if update.PowGetSysGrid != 0 {
		c.data.PowGetSysGrid = update.PowGetSysGrid
	}
	if update.PowGetSysLoad != 0 {
		c.data.PowGetSysLoad = update.PowGetSysLoad
	}
	// Relay states are boolean, always update
	c.data.Relay2Onoff = update.Relay2Onoff
	c.data.Relay3Onoff = update.Relay3Onoff
}

// handleStatusMessage processes incoming status messages
func (c *EcoFlowStreamCharger) handleStatusMessage(payload string) {
	c.log.DEBUG.Printf("status message: %s", payload)
}

// getData returns current device data, preferring MQTT but falling back to REST
func (c *EcoFlowStreamCharger) getData() (ecoflow.StreamData, error) {
	c.mu.RLock()
	lastMqtt := c.lastMqtt
	data := c.data
	c.mu.RUnlock()

	// Use MQTT data if recent (within 60 seconds)
	if time.Since(lastMqtt) < 60*time.Second {
		return data, nil
	}

	// Fallback to REST API
	restData, err := c.restDataG()
	if err != nil {
		// Return cached data even if stale
		return data, nil
	}

	// Update cache with REST data
	c.mu.Lock()
	c.data = restData
	c.mu.Unlock()

	return restData, nil
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
	data, err := c.getData()
	if err != nil {
		return api.StatusNone, err
	}

	enabled := relayState(data, c.relay)

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
	data, err := c.getData()
	if err != nil {
		return false, err
	}
	return relayState(data, c.relay), nil
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

	// Optimistically update local state
	c.mu.Lock()
	if c.relay == 2 {
		c.data.Relay3Onoff = enable
	} else {
		c.data.Relay2Onoff = enable
	}
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
	data, err := c.getData()
	if err != nil {
		return 0, err
	}
	// Return battery power (positive = charging)
	return data.PowGetBpCms, nil
}

// Soc implements api.Battery
func (c *EcoFlowStreamCharger) Soc() (float64, error) {
	data, err := c.getData()
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
