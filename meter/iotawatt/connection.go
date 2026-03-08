package iotawatt

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

var errMissingURI = errors.New("missing uri")

// Connection is the IoTaWatt connection
type Connection struct {
	*request.Helper
	uri   string
	cache time.Duration

	mu          sync.Mutex
	lastEnergy  time.Time
	totalEnergy float64
}

// NewConnection creates an IoTaWatt connection
func NewConnection(uri string, cache time.Duration) (*Connection, error) {
	if uri == "" {
		return nil, errMissingURI
	}

	log := util.NewLogger("iotawatt")

	uri = util.DefaultScheme(strings.TrimRight(uri, "/"), "http")

	c := &Connection{
		Helper: request.NewHelper(log),
		uri:    uri,
		cache:  cache,
	}

	return c, nil
}

// ShowSeries returns the available IoTaWatt series (inputs and outputs).
func (c *Connection) ShowSeries() ([]Series, error) {
	var res ShowSeriesResponse
	err := c.GetJSON(fmt.Sprintf("%s/query?show=series", c.uri), &res)
	return res.Series, err
}

// DeviceConfig returns the IoTaWatt device configuration from /config.txt.
func (c *Connection) DeviceConfig() (*DeviceConfig, error) {
	var res DeviceConfig
	err := c.GetJSON(fmt.Sprintf("%s/config.txt", c.uri), &res)
	return &res, err
}

// ValidateSeries checks that all given series names exist on the device
// and have one of the expected units. It returns the actual unit of the
// first series (all must share the same unit).
func (c *Connection) ValidateSeries(names []string, allowedUnits ...string) (string, error) {
	if len(names) == 0 {
		return "", nil
	}

	series, err := c.ShowSeries()
	if err != nil {
		return "", fmt.Errorf("fetching series: %w", err)
	}

	available := make(map[string]string, len(series))
	for _, s := range series {
		available[s.Name] = s.Unit
	}

	var unit string
	for _, name := range names {
		got, ok := available[name]
		if !ok {
			return "", fmt.Errorf("unknown iotawatt series: %s", name)
		}

		// check allowed units
		allowed := false
		for _, u := range allowedUnits {
			if strings.EqualFold(got, u) {
				allowed = true
				break
			}
		}
		if !allowed {
			return "", fmt.Errorf("iotawatt series %s has unit %q, expected one of %v", name, got, allowedUnits)
		}

		// all series must share the same unit
		if unit == "" {
			unit = got
		} else if !strings.EqualFold(unit, got) {
			return "", fmt.Errorf("iotawatt phase series have mixed units: %s is %q, but expected %q", name, got, unit)
		}
	}

	return unit, nil
}

// queryValues executes a query for the given channels and unit modifier.
// It returns one float64 per channel.
func (c *Connection) queryValues(unit string, channels ...string) ([]float64, error) {
	if len(channels) == 0 {
		return nil, errors.New("no channels specified")
	}

	// build select parameter: [ch1.unit,ch2.unit,...]
	parts := make([]string, len(channels))
	for i, ch := range channels {
		parts[i] = ch + "." + unit
	}
	sel := "[" + strings.Join(parts, ",") + "]"

	uri := fmt.Sprintf("%s/query?select=%s&begin=s-10s&end=s&group=all",
		c.uri, url.QueryEscape(sel))

	var res [][]float64
	if err := c.GetJSON(uri, &res); err != nil {
		return nil, err
	}

	if len(res) != 1 || len(res[0]) != len(channels) {
		return nil, fmt.Errorf("unexpected query response: expected 1 row with %d values, got %d rows", len(channels), len(res))
	}

	return res[0], nil
}

// QueryPower returns the current power in Watts for each channel.
func (c *Connection) QueryPower(channels ...string) ([]float64, error) {
	return c.queryValues("watts", channels...)
}

// QueryCurrents returns the current in Amps for each channel.
func (c *Connection) QueryCurrents(channels ...string) ([]float64, error) {
	return c.queryValues("amps", channels...)
}

// QueryVoltages returns the voltage in Volts for each channel.
func (c *Connection) QueryVoltages(channels ...string) ([]float64, error) {
	return c.queryValues("volts", channels...)
}

// TotalEnergy returns the accumulated energy in kWh for the given channels.
// It uses delta-based accumulation: each call queries Wh since the last call
// and adds it to a running counter. Multiple channels are summed.
func (c *Connection) TotalEnergy(channels ...string) (float64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()

	// first call: seed the timestamp, return 0
	if c.lastEnergy.IsZero() {
		c.lastEnergy = now
		return 0, nil
	}

	begin := fmt.Sprintf("%d", c.lastEnergy.Unix())

	parts := make([]string, len(channels))
	for i, ch := range channels {
		parts[i] = ch + ".wh"
	}
	sel := url.QueryEscape("[" + strings.Join(parts, ",") + "]")

	uri := fmt.Sprintf("%s/query?select=%s&begin=%s&end=s&group=all",
		c.uri, sel, begin)

	var res [][]float64
	if err := c.GetJSON(uri, &res); err != nil {
		return c.totalEnergy, err
	}

	if len(res) == 1 && len(res[0]) == len(channels) {
		for _, wh := range res[0] {
			c.totalEnergy += wh / 1000 // Wh to kWh
		}
		c.lastEnergy = now
	}

	return c.totalEnergy, nil
}
