package meter

import (
	"fmt"
	"math"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

func init() {
	registry.Add("quatt", NewQuattFromConfig)
}

type quattMeter struct {
	*request.Helper
	uri  string
	unit string
	data func() (quattFeed, error)
}

const (
	quattUnitAuto  = ""
	quattUnitHP1   = "hp1"
	quattUnitHP2   = "hp2"
	quattUnitTotal = "total"
)

type quattFeed struct {
	Boiler struct {
		FlameOn *bool `json:"otFbFlameOn"`
	} `json:"boiler"`
	FlowMeter struct {
		WaterSupplyTemperature *float64 `json:"waterSupplyTemperature"`
	} `json:"flowMeter"`
	HP1    *quattHeatPump `json:"hp1"`
	HP2    *quattHeatPump `json:"hp2"`
	QC     quattQC        `json:"qc"`
	System struct {
		MaxWaterTemperature *float64 `json:"chMaxWaterTemperature"`
	} `json:"system"`
}

type quattHeatPump struct {
	PowerInput          float64  `json:"powerInput"`
	TemperatureOutside  *float64 `json:"temperatureOutside"`
	TemperatureWaterIn  *float64 `json:"temperatureWaterIn"`
	TemperatureWaterOut *float64 `json:"temperatureWaterOut"`
}

type quattQC struct {
	SupervisoryControlMode *int `json:"supervisoryControlMode"`
}

func NewQuattFromConfig(other map[string]any) (api.Meter, error) {
	cc := struct {
		URI   string
		Host  string
		Port  int
		Unit  string
		Cache time.Duration
	}{
		Port:  8080,
		Cache: time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.Unit != quattUnitAuto && cc.Unit != quattUnitHP1 && cc.Unit != quattUnitHP2 && cc.Unit != quattUnitTotal {
		return nil, fmt.Errorf("invalid unit: %s", cc.Unit)
	}

	uri := cc.URI
	if uri == "" {
		if cc.Host == "" {
			return nil, fmt.Errorf("missing host")
		}
		uri = fmt.Sprintf("http://%s:%d/beta/feed/data.json", cc.Host, cc.Port)
	}

	m := &quattMeter{
		Helper: request.NewHelper(util.NewLogger("quatt")),
		uri:    uri,
		unit:   cc.Unit,
	}
	m.data = util.Cached(m.feed, cc.Cache)

	return m, nil
}

var _ api.Meter = (*quattMeter)(nil)

func (m *quattMeter) feed() (quattFeed, error) {
	var res quattFeed
	err := m.GetJSON(m.uri, &res)
	return res, err
}

func (m *quattMeter) CurrentPower() (float64, error) {
	feed, err := m.data()
	if err != nil {
		return 0, err
	}

	switch m.unit {
	case quattUnitHP1:
		return heatPumpPower(feed.HP1), nil
	case quattUnitHP2:
		return heatPumpPower(feed.HP2), nil
	case quattUnitTotal:
		return heatPumpPower(feed.HP1) + heatPumpPower(feed.HP2), nil
	default:
		if feed.HP2 != nil {
			return heatPumpPower(feed.HP1) + heatPumpPower(feed.HP2), nil
		}
		return heatPumpPower(feed.HP1), nil
	}
}

var _ api.MeterDetails = (*quattMeter)(nil)

func (m *quattMeter) Details() ([]string, error) {
	feed, err := m.data()
	if err != nil {
		return nil, err
	}

	hp := feed.HP1
	if m.unit == quattUnitHP2 {
		hp = feed.HP2
	}

	details := []string{
		"Mode: " + supervisoryMode(feed.QC.SupervisoryControlMode),
	}

	if feed.System.MaxWaterTemperature != nil {
		details = append(details, fmt.Sprintf("Max water temp: %.0f °C", *feed.System.MaxWaterTemperature))
	}
	if feed.Boiler.FlameOn != nil {
		details = append(details, "Flame: "+flameStatus(feed.Boiler.FlameOn))
	}

	if m.unit == quattUnitTotal || feed.HP2 != nil {
		details = appendHeatPumpDetails(details, "HP1", feed.HP1)
		details = appendHeatPumpDetails(details, "HP2", feed.HP2)
		return details, nil
	}

	details = appendHeatPumpDetails(details, "", hp)
	return details, nil
}

func heatPumpPower(hp *quattHeatPump) float64 {
	if hp == nil {
		return 0
	}
	return hp.PowerInput * 1000
}

func appendHeatPumpDetails(details []string, label string, hp *quattHeatPump) []string {
	prefix := ""
	if label != "" {
		prefix = label + " "
	}
	if hp == nil {
		return append(details, prefix+"not available")
	}
	if hp.TemperatureWaterIn != nil {
		details = append(details, fmt.Sprintf("%swater in: %.1f °C", prefix, round1(*hp.TemperatureWaterIn)))
	}
	if hp.TemperatureWaterOut != nil {
		details = append(details, fmt.Sprintf("%swater out: %.1f °C", prefix, round1(*hp.TemperatureWaterOut)))
	}
	if hp.TemperatureOutside != nil {
		details = append(details, fmt.Sprintf("%soutside: %.1f °C", prefix, round1(*hp.TemperatureOutside)))
	}
	return details
}

func supervisoryMode(mode *int) string {
	if mode == nil {
		return "unknown"
	}

	switch *mode {
	case 0:
		return "standby"
	default:
		return fmt.Sprintf("mode %d", *mode)
	}
}

func flameStatus(flame *bool) string {
	if flame == nil {
		return "unknown"
	}
	if *flame {
		return "on"
	}
	return "off"
}

func round1(v float64) float64 {
	return math.Round(v*10) / 10
}
