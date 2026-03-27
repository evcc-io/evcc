package meter

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

func init() {
	registry.Add("stromleser", NewStromleserFromConfig)
}

type stromleserResponse map[string]string

// Stromleser meter implementation
type Stromleser struct {
	*request.Helper
	uri   string
	usage string
	dataG util.Cacheable[stromleserResponse]
}

// NewStromleserFromConfig creates a Stromleser meter from generic config
func NewStromleserFromConfig(other map[string]any) (api.Meter, error) {
	cc := struct {
		URI   string
		Usage string
		Cache time.Duration
	}{
		Cache: 5 * time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.URI == "" {
		return nil, errors.New("missing uri")
	}

	uri := util.DefaultScheme(strings.TrimRight(cc.URI, "/"), "http")

	log := util.NewLogger("stromleser")

	m := &Stromleser{
		Helper: request.NewHelper(log),
		uri:    uri,
		usage:  strings.ToLower(cc.Usage),
	}

	m.dataG = util.ResettableCached(func() (stromleserResponse, error) {
		var res stromleserResponse
		err := m.GetJSON(m.uri+"/v1/data", &res)
		return res, err
	}, cc.Cache)

	return m, nil
}

// parseOBIS parses an OBIS value string of the form "<float> <unit>" and returns the float.
func parseOBIS(s string) (float64, error) {
	parts := strings.SplitN(strings.TrimSpace(s), " ", 2)
	if len(parts) == 0 || parts[0] == "" {
		return 0, fmt.Errorf("stromleser: cannot parse OBIS value %q", s)
	}
	return strconv.ParseFloat(parts[0], 64)
}

var _ api.Meter = (*Stromleser)(nil)

// CurrentPower implements the api.Meter interface.
// Prefers OBIS 16.7.0 (net active power); falls back to 1.7.0 - 2.7.0.
// For pv usage the sign is negated so that export is positive.
func (m *Stromleser) CurrentPower() (float64, error) {
	res, err := m.dataG.Get()
	if err != nil {
		return 0, err
	}

	var power float64

	if v, ok := res["16.7.0"]; ok {
		power, err = parseOBIS(v)
		if err != nil {
			return 0, err
		}
	} else {
		imp, err1 := parseOBIS(res["1.7.0"])
		exp, err2 := parseOBIS(res["2.7.0"])
		if err1 != nil || err2 != nil {
			return 0, fmt.Errorf("stromleser: no usable power register in response")
		}
		power = imp - exp
	}

	if m.usage == "pv" {
		return -power, nil
	}
	return power, nil // grid and charge: positive = consuming
}

var _ api.MeterEnergy = (*Stromleser)(nil)

// TotalEnergy implements the api.MeterEnergy interface.
// grid: OBIS 1.8.0 (total import kWh); pv: OBIS 2.8.0 (total export kWh).
func (m *Stromleser) TotalEnergy() (float64, error) {
	res, err := m.dataG.Get()
	if err != nil {
		return 0, err
	}

	key := "1.8.0" // grid and charge: total import kWh
	if m.usage == "pv" {
		key = "2.8.0" // pv: total export kWh
	}

	v, ok := res[key]
	if !ok {
		return 0, fmt.Errorf("stromleser: OBIS key %q not found in response", key)
	}

	return parseOBIS(v)
}
