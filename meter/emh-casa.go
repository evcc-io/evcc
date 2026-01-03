package meter

import (
	"crypto/tls"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/jpfielding/go-http-digest/pkg/digest"
)

// hostHeaderTransport wraps a RoundTripper and enforces a custom Host header
type hostHeaderTransport struct {
	base http.RoundTripper
	host string
}

func (t *hostHeaderTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.Host = t.host
	req.Header.Set("Host", t.host)
	return t.base.RoundTrip(req)
}

// parseURIHost extracts the host from a URI (without port/path)
func parseURIHost(uri string) (string, error) {
	uri = strings.TrimPrefix(uri, "https://")
	uri = strings.TrimPrefix(uri, "http://")

	if idx := strings.Index(uri, "/"); idx != -1 {
		uri = uri[:idx]
	}
	if idx := strings.Index(uri, ":"); idx != -1 {
		uri = uri[:idx]
	}

	if uri == "" {
		return "", fmt.Errorf("invalid uri")
	}

	return uri, nil
}

func init() {
	registry.Add("emh-casa", NewEMHCasaFromConfig)
}

// EMHCasa meter implementation for EMH CASA 1.1 Smart Meter Gateways
type EMHCasa struct {
	*request.Helper
	uri     string
	meterID string
	valuesG func() (map[string]float64, error)
	log     *util.Logger
}

// NewEMHCasaFromConfig creates meter from evcc config
func NewEMHCasaFromConfig(other map[string]any) (api.Meter, error) {
	cc := struct {
		URI      string
		User     string
		Password string
		MeterID  string
		Host     string // REQUIRED for most CASA gateways (example: 192.168.33.2)
		Cache    time.Duration
	}{
		Cache: 10 * time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.URI == "" {
		return nil, fmt.Errorf("missing uri")
	}

	if cc.User == "" || cc.Password == "" {
		return nil, api.ErrMissingCredentials
	}

	return NewEMHCasa(cc.URI, cc.User, cc.Password, cc.MeterID, cc.Host, cc.Cache)
}

// NewEMHCasa creates an EMH CASA meter
func NewEMHCasa(uri, user, password, meterID, host string, cache time.Duration) (api.Meter, error) {
	log := util.NewLogger("emh-casa")

	if host == "" {
		derived, err := parseURIHost(uri)
		if err != nil {
			return nil, fmt.Errorf("host required and could not be derived: %w", err)
		}

		log.WARN.Printf(
			"no host configured, using host derived from uri (%s). "+
				"EMH CASA usually requires the gateway IP (e.g. 192.168.33.2)",
			derived,
		)

		host = derived
	} else {
		log.DEBUG.Printf("using configured host header: %s", host)
	}

	// CASA gateways require HTTP/1.1 and usually use self-signed certs
	customTransport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
		ForceAttemptHTTP2: false,
	}

	hostTransport := &hostHeaderTransport{
		base: customTransport,
		host: host,
	}

	helper := request.NewHelper(log)
	helper.Client.Transport = digest.NewTransport(user, password, hostTransport)

	m := &EMHCasa{
		Helper:  helper,
		uri:     util.DefaultScheme(uri, "https"),
		meterID: meterID,
		log:     log,
	}

	if m.meterID == "" {
		if err := m.discoverMeterID(); err != nil {
			return nil, fmt.Errorf("failed to discover meter ID: %w", err)
		}
		log.DEBUG.Printf("discovered meter ID: %s", m.meterID)
	} else {
		log.DEBUG.Printf("using configured meter ID: %s", m.meterID)
	}

	// Validate connection
	log.DEBUG.Printf("validating connection...")
	if _, err := m.getMeterValues(); err != nil {
		return nil, fmt.Errorf("failed to validate meter connection: %w", err)
	}
	log.DEBUG.Printf("connection validated successfully")

	m.valuesG = util.Cached(m.getMeterValues, cache)

	return m, nil
}

// ---- API structures ----

type derivedContract struct {
	TafType       string   `json:"taf_type"`
	SensorDomains []string `json:"sensor_domains"`
}

type meterValue struct {
	Value       string `json:"value"`
	Unit        int    `json:"unit"`   // 27 = W, 30 = Wh
	Scaler      int    `json:"scaler"` // power-of-10 multiplier
	LogicalName string `json:"logical_name"`
}

type meterReading struct {
	Values []meterValue `json:"values"`
}

// discoverMeterID finds the first TAF-1 contract
func (m *EMHCasa) discoverMeterID() error {
	var contracts []string
	uri := fmt.Sprintf("%s/json/metering/derived", m.uri)

	m.log.DEBUG.Printf("discovering meter ID from: %s", uri)

	if err := m.GetJSON(uri, &contracts); err != nil {
		return fmt.Errorf("failed to get contracts: %w", err)
	}

	m.log.DEBUG.Printf("found %d contract(s)", len(contracts))

	for _, id := range contracts {
		m.log.DEBUG.Printf("checking contract: %s", id)

		var c derivedContract
		uri := fmt.Sprintf("%s/json/metering/derived/%s", m.uri, id)

		if err := m.GetJSON(uri, &c); err != nil {
			m.log.DEBUG.Printf("failed to get contract details: %v", err)
			continue
		}

		m.log.DEBUG.Printf("contract %s: taf_type=%s, sensor_domains=%v", id, c.TafType, c.SensorDomains)

		if c.TafType == "TAF-1" && len(c.SensorDomains) > 0 {
			m.meterID = c.SensorDomains[0]
			return nil
		}
	}

	return fmt.Errorf("no TAF-1 contract found among %d contracts", len(contracts))
}

// convertToOBIS converts CASA logical name to OBIS C.D.E
func convertToOBIS(logicalName string) (string, error) {
	hex := strings.SplitN(logicalName, ".", 2)[0]

	if len(hex) != 12 {
		return "", fmt.Errorf("unexpected logical name: %s", logicalName)
	}

	c, err := strconv.ParseInt(hex[4:6], 16, 64)
	if err != nil {
		return "", err
	}
	d, err := strconv.ParseInt(hex[6:8], 16, 64)
	if err != nil {
		return "", err
	}
	e, err := strconv.ParseInt(hex[8:10], 16, 64)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%d.%d.%d", c, d, e), nil
}

// getMeterValues fetches and parses meter readings
func (m *EMHCasa) getMeterValues() (map[string]float64, error) {
	var reading meterReading
	uri := fmt.Sprintf("%s/json/metering/origin/%s/extended", m.uri, m.meterID)

	if err := m.GetJSON(uri, &reading); err != nil {
		return nil, err
	}

	values := make(map[string]float64)

	for _, item := range reading.Values {
		obis, err := convertToOBIS(item.LogicalName)
		if err != nil {
			continue
		}

		raw, err := strconv.ParseFloat(item.Value, 64)
		if err != nil {
			continue
		}

		val := raw * math.Pow(10, float64(item.Scaler))

		switch item.Unit {
		case 27: // W
			values[obis] = val
		case 30: // Wh → kWh
			values[obis] = val / 1000
		}
	}

	if len(values) == 0 {
		return nil, fmt.Errorf("no valid meter values found")
	}

	return values, nil
}

// ---- evcc interfaces ----

// CurrentPower implements api.Meter (OBIS 16.7.0)
func (m *EMHCasa) CurrentPower() (float64, error) {
	values, err := m.valuesG()
	if err != nil {
		return 0, err
	}

	v, ok := values["16.7.0"]
	if !ok {
		return 0, fmt.Errorf("power value (16.7.0) not found")
	}

	return v, nil
}

// TotalEnergy implements api.MeterEnergy (OBIS 1.8.0)
func (m *EMHCasa) TotalEnergy() (float64, error) {
	values, err := m.valuesG()
	if err != nil {
		return 0, err
	}

	v, ok := values["1.8.0"]
	if !ok {
		return 0, fmt.Errorf("energy value (1.8.0) not found")
	}

	return v, nil
}

// GridProduction returns grid feed-in (OBIS 2.8.0)
func (m *EMHCasa) GridProduction() (float64, error) {
	values, err := m.valuesG()
	if err != nil {
		return 0, err
	}

	v, ok := values["2.8.0"]
	if !ok {
		return 0, fmt.Errorf("production value (2.8.0) not found")
	}

	return v, nil
}

var _ api.Meter = (*EMHCasa)(nil)
var _ api.MeterEnergy = (*EMHCasa)(nil)
