package meter

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/spf13/cast"
	"github.com/tess1o/go-ecoflow"
)

// EcoFlowStreamMqtt is a dedicated meter type for the EcoFlow Stream family
// that uses the public MQTT API for live data.
//
// Unlike other EcoFlow devices (Delta, Powerstream, ...) the Stream series
// only publishes live per-string PV power, per-second battery state etc.
// over MQTT. The legacy `ecoflow` meter (used by the original
// `ecoflow-stream` template) still works but is limited to the stale REST
// subset of parameters exposed by `quota/all`. This type
// adds:
//
//   - subscription to the public MQTT broker for live quota updates;
//   - optional auto-discovery of sibling devices on the same account with
//     aggregated readings (cascade mode);
//   - fixed evcc-usage -> EcoFlow param mapping (grid / pv / battery);
//   - SoC as the cascade average and battery power as inputWatts -
//     outputWatts (then inverted to evcc's sign convention).
type EcoFlowStreamMqtt struct {
	ctx     context.Context
	log     *util.Logger
	usage   string
	primary string   // originally configured serial (used for filter prefix)
	serials []string // every serial this meter aggregates (>= 1)
	cache   time.Duration
	client  *ecoflow.Client
	dataGs  map[string]func() (*ecoflow.GetCmdResponse, error) // per-serial cached REST getter
	mqtt    *ecoflowMqttClient

	// param keys mapped from usage (see usageParams)
	powerKeys    []string // summed into power
	powerNegKeys []string // subtracted from power
	socKey       string   // battery state-of-charge key
}

func init() {
	registry.AddCtx("ecoflow-stream-mqtt", NewEcoFlowStreamMqttFromConfig)
}

// usageParams returns the EcoFlow MQTT parameter keys for the given evcc
// meter usage. These are the keys that Stream devices publish on the
// `/open/<user>/<sn>/quota` topic; the REST `quota/all` endpoint returns a
// largely disjoint (and often stale) subset, so REST is only a fallback.
func usageParams(usage string) (power, powerNeg []string, soc string, err error) {
	switch usage {
	case "grid":
		return []string{"powGetSysGrid"}, nil, "", nil
	case "pv":
		return []string{"powGetPv", "powGetPv2", "powGetPv3", "powGetPv4"}, nil, "", nil
	case "battery":
		return []string{"inputWatts"}, []string{"outputWatts"}, "f32ShowSoc", nil
	default:
		return nil, nil, "", fmt.Errorf("invalid usage: %s", usage)
	}
}

// NewEcoFlowStreamMqttFromConfig creates a Stream meter from generic config.
func NewEcoFlowStreamMqttFromConfig(ctx context.Context, other map[string]any) (api.Meter, error) {
	cc := struct {
		batteryCapacity                      `mapstructure:",squash"`
		batteryPowerLimits                   `mapstructure:",squash"`
		batterySocLimits                     `mapstructure:",squash"`
		Usage                                string
		AccessKey, SecretKey, Serial, Region string
		Serials                              []string
		Discover                             bool
		Cache                                time.Duration
	}{
		Cache:    30 * time.Second,
		Discover: true,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}
	if cc.AccessKey == "" {
		return nil, errors.New("missing access key")
	}
	if cc.SecretKey == "" {
		return nil, errors.New("missing secret key")
	}
	if cc.Serial == "" {
		return nil, errors.New("missing serial")
	}
	if cc.Usage == "" {
		return nil, errors.New("missing usage")
	}

	var uri string
	switch cc.Region {
	case "", "auto":
		uri = "https://api.ecoflow.com"
	case "europe":
		uri = "https://api-e.ecoflow.com"
	case "america":
		uri = "https://api-a.ecoflow.com"
	default:
		return nil, fmt.Errorf("invalid region: %s", cc.Region)
	}

	m, err := NewEcoFlowStreamMqtt(ctx, cc.AccessKey, cc.SecretKey, cc.Serial, cc.Serials, cc.Discover, cc.Usage, uri, cc.Cache)
	if err != nil {
		return nil, err
	}

	if cc.Usage == "battery" {
		return decorateMeterBattery(
			m, nil, m.soc, cc.batteryCapacity.Decorator(),
			cc.batterySocLimits.Decorator(), cc.batteryPowerLimits.Decorator(), nil,
		), nil
	}

	return m, nil
}

// NewEcoFlowStream constructs the Stream meter. If discover is true the
// EcoFlow device list is queried and every device whose serial shares the
// primary serial's 2-char prefix (e.g. "BK" for the Stream family) is added
// as a sibling so cascade systems appear as one aggregated meter.
func NewEcoFlowStreamMqtt(ctx context.Context, accessKey, secretKey, serial string, serials []string, discover bool,
	usage, uri string, cache time.Duration,
) (*EcoFlowStreamMqtt, error) {
	power, powerNeg, soc, err := usageParams(usage)
	if err != nil {
		return nil, err
	}

	m := &EcoFlowStreamMqtt{
		ctx:          ctx,
		log:          util.NewLogger("ecoflow-stream-mqtt"),
		primary:      serial,
		usage:        usage,
		cache:        cache,
		client:       ecoflow.NewEcoflowClient(accessKey, secretKey, ecoflow.WithBaseUrl(uri)),
		dataGs:       make(map[string]func() (*ecoflow.GetCmdResponse, error)),
		powerKeys:    power,
		powerNegKeys: powerNeg,
		socKey:       soc,
	}

	set := make(map[string]struct{})
	add := func(sn string) {
		if sn == "" {
			return
		}
		if _, ok := set[sn]; ok {
			return
		}
		set[sn] = struct{}{}
		m.serials = append(m.serials, sn)
	}
	add(serial)
	for _, sn := range serials {
		add(sn)
	}

	if discover {
		// Fail soft: if discovery fails we continue with whatever was
		// configured explicitly so a broker or network outage doesn't
		// prevent evcc from starting.
		dctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		resp, dErr := m.client.GetDeviceList(dctx)
		cancel()
		switch {
		case dErr != nil:
			m.log.WARN.Printf("discover: device list failed: %v", dErr)
		case resp == nil:
			m.log.WARN.Printf("discover: empty device list response")
		default:
			// discover adds slaves: every device that shares the
			// primary serial's 2-char family prefix (e.g. "BK" for
			// Stream) is treated as a sibling.
			if len(m.primary) >= 2 {
				prefix := m.primary[:2]
				for _, d := range resp.Devices {
					if !strings.HasPrefix(d.SN, prefix) {
						continue
					}
					add(d.SN)
				}
			}
			m.log.INFO.Printf("discover: tracking %d device(s): %s", len(m.serials), strings.Join(m.serials, ","))
		}
	}

	for _, sn := range m.serials {
		m.dataGs[sn] = util.Cached(func() (*ecoflow.GetCmdResponse, error) {
			return m.fetchData(sn)
		}, cache)
	}

	// Subscribe to live MQTT updates for every serial. Registration runs in
	// a goroutine so that a broker outage doesn't block evcc from starting.
	m.mqtt = ecoflowMqttClientFor(accessKey, secretKey, uri)
	go func() {
		for _, sn := range m.serials {
			if err := m.mqtt.Register(ctx, sn); err != nil {
				m.log.WARN.Printf("mqtt register %s: %v", sn, err)
			}
		}
	}()

	return m, nil
}

// fetchData retrieves the configured parameters for a single serial.
func (m *EcoFlowStreamMqtt) fetchData(sn string) (*ecoflow.GetCmdResponse, error) {
	ctx, cancel := context.WithTimeout(m.ctx, 30*time.Second)
	defer cancel()

	params := append([]string(nil), m.powerKeys...)
	params = append(params, m.powerNegKeys...)
	if m.socKey != "" {
		params = append(params, m.socKey)
	}
	return m.client.GetDeviceParameters(ctx, sn, params)
}

var _ api.Meter = (*EcoFlowStreamMqtt)(nil)

// CurrentPower implements the api.Meter interface. Per-serial it sums the
// powerKeys and subtracts the powerNegKeys; the per-serial nets are then
// summed across all tracked devices to produce the total. Battery power is
// finally inverted to match evcc's convention (negative = charging).
//
// A REST failure on one device (e.g. a transient cloud hiccup for a single
// cascade unit) is logged and skipped so the aggregate meter keeps
// reporting from the healthy devices. Only when every device is
// unreachable is the last error surfaced.
func (m *EcoFlowStreamMqtt) CurrentPower() (float64, error) {
	var (
		total    float64
		anyFound bool
		lastErr  error
	)

	for _, sn := range m.serials {
		pos, neg, found, err := m.devicePower(sn)
		if err != nil {
			m.log.WARN.Printf("%s: %v", sn, err)
			lastErr = err
			continue
		}
		total += pos - neg
		if found {
			anyFound = true
		}
	}

	if !anyFound {
		if lastErr != nil {
			return 0, lastErr
		}
		// For a single positive key on a single device (simple grid) we
		// preserve strict ErrNotAvailable propagation. With multi-key
		// sums (per-string PV, input/output split) or multiple devices,
		// an empty response means 0 W.
		totalKeys := len(m.powerKeys) + len(m.powerNegKeys)
		if totalKeys == 1 && len(m.serials) == 1 {
			return 0, api.ErrNotAvailable
		}
	}

	if m.usage == "battery" {
		total = -total // ecoflow reports charging as positive
	}

	return total, nil
}

// devicePower resolves the net power for a single serial. Both key lists
// share one REST round-trip (via an inner closure) to minimise API load.
func (m *EcoFlowStreamMqtt) devicePower(sn string) (pos, neg float64, found bool, err error) {
	var restData map[string]any
	getRest := func() (map[string]any, error) {
		if restData != nil {
			return restData, nil
		}
		response, err := m.dataGs[sn]()
		if err != nil {
			return nil, err
		}
		restData = response.Data
		if restData == nil {
			restData = map[string]any{}
		}
		return restData, nil
	}

	pos, posFound, err := m.sumKeys(sn, m.powerKeys, getRest)
	if err != nil {
		return 0, 0, false, err
	}
	neg, negFound, err := m.sumKeys(sn, m.powerNegKeys, getRest)
	if err != nil {
		return 0, 0, false, err
	}
	return pos, neg, posFound || negFound, nil
}

// sumKeys resolves each key via MQTT (preferred) or REST for the given
// serial and returns the running sum. Keys missing from both sources are
// skipped silently so a device that doesn't expose every optional key
// doesn't abort the whole reading.
func (m *EcoFlowStreamMqtt) sumKeys(sn string, keys []string, getRest func() (map[string]any, error)) (float64, bool, error) {
	var sum float64
	var found bool
	for _, key := range keys {
		if v, ok := m.mqttValue(sn, key); ok {
			sum += v
			found = true
			continue
		}

		data, err := getRest()
		if err != nil {
			return 0, false, err
		}

		v, err := ecoflowValue(data, key)
		if errors.Is(err, api.ErrNotAvailable) {
			continue
		}
		if err != nil {
			return 0, false, err
		}
		sum += v
		found = true
	}
	return sum, found, nil
}

// mqttValue returns the live MQTT-cached value for the given serial/key,
// if available.
func (m *EcoFlowStreamMqtt) mqttValue(sn, key string) (float64, bool) {
	if m.mqtt == nil {
		return 0, false
	}
	v, ok := m.mqtt.Lookup(sn, key)
	if !ok {
		return 0, false
	}
	f, err := cast.ToFloat64E(v)
	if err != nil {
		return 0, false
	}
	return f, true
}

// soc returns the battery state of charge, averaged across all tracked
// serials (each contributing serial weighted equally, matching the EcoFlow
// app's cascade view). Per-device REST failures are logged and skipped so
// one failing unit doesn't hide the SoC of the remaining devices.
func (m *EcoFlowStreamMqtt) soc() (float64, error) {
	if m.socKey == "" {
		return 0, api.ErrNotAvailable
	}

	var (
		sum     float64
		count   int
		lastErr error
	)
	for _, sn := range m.serials {
		if v, ok := m.mqttValue(sn, m.socKey); ok {
			sum += v
			count++
			continue
		}

		response, err := m.dataGs[sn]()
		if err != nil {
			m.log.WARN.Printf("%s: %v", sn, err)
			lastErr = err
			continue
		}
		v, err := ecoflowValue(response.Data, m.socKey)
		if errors.Is(err, api.ErrNotAvailable) {
			continue
		}
		if err != nil {
			m.log.WARN.Printf("%s: %v", sn, err)
			lastErr = err
			continue
		}
		sum += v
		count++
	}

	if count == 0 {
		if lastErr != nil {
			return 0, lastErr
		}
		return 0, api.ErrNotAvailable
	}
	return sum / float64(count), nil
}
