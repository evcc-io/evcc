package meter

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/evcc-io/evcc/plugin/mqtt"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

// ecoflowMqttClient maintains a shared MQTT subscription to EcoFlow's public
// IoT broker. The REST `quota/all` endpoint returns only a static subset of
// parameters for Stream devices; live per-string PV power, per-second state
// etc. are delivered exclusively via MQTT. Each configured EcoFlow meter
// registers its device serial here and reads from the in-memory cache.
//
// Transport is delegated to evcc's shared `plugin/mqtt.Client` (which handles
// the Paho options, TLS, reconnect, and automatic re-subscription of every
// `Listen`ed topic on reconnect). The EcoFlow-specific parts that remain
// here are the HMAC-signed HTTPS certification call, the stable
// per-host client ID (to respect EcoFlow's 10-IDs/account/day quota) and
// the flat-vs-envelope quota-payload parser.
type ecoflowMqttClient struct {
	log *util.Logger

	accessKey string
	secretKey string
	uri       string // REST base, e.g. https://api-e.ecoflow.com
	http      *http.Client

	startMu  sync.Mutex // serializes the initial certification/connect
	mu       sync.Mutex
	state    map[string]map[string]any // sn -> key -> latest value
	devices  map[string]struct{}       // serials for which Listen has been called
	client   *mqtt.Client
	username string // certificateAccount, used in quota topic
}

// ecoflowMqttClients caches one client per (accessKey, uri) pair so the HTTPS
// certification call is made once per account, and so the resulting MQTT
// connection (and its 1-of-10 daily client-ID slot) is shared across every
// ecoflow meter for that account.
var (
	ecoflowMqttClientsMu sync.Mutex
	ecoflowMqttClients   = map[string]*ecoflowMqttClient{}
)

func ecoflowMqttClientFor(accessKey, secretKey, uri string) *ecoflowMqttClient {
	ecoflowMqttClientsMu.Lock()
	defer ecoflowMqttClientsMu.Unlock()

	key := accessKey + "|" + uri
	if c, ok := ecoflowMqttClients[key]; ok {
		return c
	}
	log := util.NewLogger("ecoflow-mqtt")
	c := &ecoflowMqttClient{
		log:       log,
		accessKey: accessKey,
		secretKey: secretKey,
		uri:       uri,
		http:      request.NewClient(log),
		state:     make(map[string]map[string]any),
		devices:   make(map[string]struct{}),
	}
	ecoflowMqttClients[key] = c
	return c
}

// ecoflowCertResponse is the payload from GET /iot-open/sign/certification.
type ecoflowCertResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    struct {
		CertificateAccount  string `json:"certificateAccount"`
		CertificatePassword string `json:"certificatePassword"`
		URL                 string `json:"url"`
		Port                string `json:"port"`
		Protocol            string `json:"protocol"`
	} `json:"data"`
}

// ecoflowSign builds the HMAC-SHA256 signature for a signed EcoFlow public
// API request as documented in EcoFlow's developer portal.
func ecoflowSign(accessKey, secretKey, nonce, timestamp, query string) string {
	target := "accessKey=" + accessKey + "&nonce=" + nonce + "&timestamp=" + timestamp
	if query != "" {
		target = query + "&" + target
	}
	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write([]byte(target))
	return hex.EncodeToString(h.Sum(nil))
}

// certify fetches MQTT credentials from the public API.
func (c *ecoflowMqttClient) certify(ctx context.Context) (*ecoflowCertResponse, error) {
	nonce := strconv.Itoa(int(time.Now().UnixNano() % 1_000_000))
	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
	sign := ecoflowSign(c.accessKey, c.secretKey, nonce, timestamp, "")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.uri+"/iot-open/sign/certification", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("accessKey", c.accessKey)
	req.Header.Set("nonce", nonce)
	req.Header.Set("timestamp", timestamp)
	req.Header.Set("sign", sign)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("certification: http %d: %s", resp.StatusCode, string(body))
	}

	var cert ecoflowCertResponse
	if err := json.Unmarshal(body, &cert); err != nil {
		return nil, fmt.Errorf("certification: %w (body=%s)", err, string(body))
	}
	if cert.Code != "0" {
		return nil, fmt.Errorf("certification: code=%s message=%s", cert.Code, cert.Message)
	}
	return &cert, nil
}

// ensureStarted lazily performs the HTTPS certification and opens a shared
// `plugin/mqtt.Client` on the first device registration. Subsequent calls
// are no-ops.
func (c *ecoflowMqttClient) ensureStarted(ctx context.Context) error {
	c.startMu.Lock()
	defer c.startMu.Unlock()

	c.mu.Lock()
	started := c.client != nil
	c.mu.Unlock()
	if started {
		return nil
	}

	cert, err := c.certify(ctx)
	if err != nil {
		return err
	}

	port, err := strconv.Atoi(cert.Data.Port)
	if err != nil {
		return fmt.Errorf("invalid mqtt port %q: %w", cert.Data.Port, err)
	}

	// Stable client id that is also unique per host. EcoFlow's broker
	// permits only a single concurrent connection per client id and imposes
	// a 10 unique-ID/day limit per account. Hashing (accessKey + hostname)
	// keeps the id stable across restarts while allowing multiple evcc
	// instances (e.g. prod + dev) to coexist on the same access key.
	hostname, _ := os.Hostname()
	h := sha256.Sum256([]byte(c.accessKey + "|" + hostname))
	clientID := "evcc-" + hex.EncodeToString(h[:6])

	// EcoFlow's certification response uses the `ssl` scheme which is
	// paho's alias for TLS. plugin/mqtt specifically recognises the
	// `tls://` prefix and preserves it through its DefaultPort helper;
	// normalise to that form here.
	broker := fmt.Sprintf("tls://%s:%d", cert.Data.URL, port)

	client, err := mqtt.NewClient(c.log, broker,
		cert.Data.CertificateAccount, cert.Data.CertificatePassword,
		clientID, 1, false, "", "", "")
	if err != nil {
		return fmt.Errorf("mqtt connect: %w", err)
	}

	c.mu.Lock()
	c.client = client
	c.username = cert.Data.CertificateAccount
	c.mu.Unlock()

	return nil
}

// Register subscribes the client to live updates for the given device serial.
// Safe to call multiple times; extra calls are no-ops.
//
// Re-subscription on reconnect is handled automatically by `plugin/mqtt`'s
// ConnectionHandler, which replays every `Listen`ed topic.
func (c *ecoflowMqttClient) Register(ctx context.Context, sn string) error {
	if err := c.ensureStarted(ctx); err != nil {
		return err
	}

	c.mu.Lock()
	_, existed := c.devices[sn]
	if !existed {
		c.devices[sn] = struct{}{}
	}
	client := c.client
	username := c.username
	c.mu.Unlock()

	if existed {
		return nil
	}

	topic := fmt.Sprintf("/open/%s/%s/quota", username, sn)
	return client.Listen(topic, func(payload string) {
		c.handleMessage(sn, []byte(payload))
	})
}

// handleMessage parses a quota payload and updates the cache. Payload shapes
// observed on EcoFlow public MQTT:
//
//  1. Stream devices emit a flat JSON object with parameters at the top
//     level, e.g. {"powGetPv2":214.14,"gridConnectionPower":283.99}.
//  2. Other devices wrap updates in a {"params": {...}} (or "param") envelope
//     and may include metadata like timestamp/typeCode at the top level.
//
// We accept both shapes: if "params"/"param" is present we take it, otherwise
// we merge the top-level object as-is (ignoring a few well-known metadata
// fields that are not quota parameters).
func (c *ecoflowMqttClient) handleMessage(sn string, payload []byte) {
	var raw map[string]any
	if err := json.Unmarshal(payload, &raw); err != nil {
		c.log.TRACE.Printf("ignoring non-JSON payload on %s: %v", sn, err)
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	dev := c.state[sn]
	if dev == nil {
		dev = make(map[string]any)
		c.state[sn] = dev
	}

	var merged bool
	if p, ok := raw["params"].(map[string]any); ok {
		maps.Copy(dev, p)
		merged = true
	}
	if p, ok := raw["param"].(map[string]any); ok {
		maps.Copy(dev, p)
		merged = true
	}

	if !merged {
		// flat payload: treat top-level keys as parameters
		for k, v := range raw {
			switch k {
			case "id", "version", "timestamp", "typeCode", "cmdFunc", "cmdId":
				// skip message envelope fields
				continue
			}
			dev[k] = v
		}
	}
}

// Lookup returns the most recently cached MQTT value for the given serial and
// parameter name. The boolean is false if either the device has not been seen
// yet or the key has not been reported.
func (c *ecoflowMqttClient) Lookup(sn, key string) (any, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	dev, ok := c.state[sn]
	if !ok {
		return nil, false
	}
	v, ok := dev[key]
	return v, ok
}
