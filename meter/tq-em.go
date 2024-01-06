package meter

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/provider"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

func init() {
	registry.Add("tq-em", NewTqEmFromConfig)
}

type tqemData struct {
	Authentication *bool
	Serial         string
	Obis1_4_0      float64  `json:"1-0:1.4.0*255"`
	Obis1_8_0      float64  `json:"1-0:1.8.0*255"`
	Obis2_4_0      float64  `json:"1-0:2.4.0*255"`
	Obis2_8_0      float64  `json:"1-0:2.8.0*255"`
	Obis13_4_0     float64  `json:"1-0:13.4.0*255"`
	Obis14_4_0     float64  `json:"1-0:14.4.0*255"`
	Obis21_4_0     float64  `json:"1-0:21.4.0*255"`
	Obis21_8_0     float64  `json:"1-0:21.8.0*255"`
	Obis22_4_0     float64  `json:"1-0:22.4.0*255"`
	Obis22_8_0     float64  `json:"1-0:22.8.0*255"`
	Obis31_4_0     *float64 `json:"1-0:31.4.0*255"` // optional currents
	Obis32_4_0     float64  `json:"1-0:32.4.0*255"`
	Obis33_4_0     float64  `json:"1-0:33.4.0*255"`
	Obis41_4_0     float64  `json:"1-0:41.4.0*255"`
	Obis41_8_0     float64  `json:"1-0:41.8.0*255"`
	Obis42_4_0     float64  `json:"1-0:42.4.0*255"`
	Obis42_8_0     float64  `json:"1-0:42.8.0*255"`
	Obis51_4_0     *float64 `json:"1-0:51.4.0*255"` // optional currents
	Obis52_4_0     float64  `json:"1-0:52.4.0*255"`
	Obis53_4_0     float64  `json:"1-0:53.4.0*255"`
	Obis61_4_0     float64  `json:"1-0:61.4.0*255"`
	Obis61_8_0     float64  `json:"1-0:61.8.0*255"`
	Obis62_4_0     float64  `json:"1-0:62.4.0*255"`
	Obis62_8_0     float64  `json:"1-0:62.8.0*255"`
	Obis71_4_0     *float64 `json:"1-0:71.4.0*255"` // optional currents
	Obis72_4_0     float64  `json:"1-0:72.4.0*255"`
	Obis73_4_0     float64  `json:"1-0:73.4.0*255"`
}

type TqEm struct {
	dataG func() (tqemData, error)
}

//go:generate go run ../cmd/tools/decorate.go -f decorateTqEm -b api.Meter -t "api.PhaseCurrents,Currents,func() (float64, float64, float64, error)"

// NewTqEmFromConfig creates a new configurable meter
func NewTqEmFromConfig(other map[string]interface{}) (api.Meter, error) {
	cc := struct {
		URI      string
		Password string
		Cache    time.Duration
	}{
		Cache: time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	log := util.NewLogger("tq-em").Redact(cc.Password)

	client := request.NewHelper(log)
	client.Jar, _ = cookiejar.New(nil)

	base := util.DefaultScheme(strings.TrimRight(cc.URI, "/"), "http")

	// get serial number
	var meter tqemData

	uri := fmt.Sprintf("%s/start.php", base)
	err := client.GetJSON(uri, &meter)
	if err != nil {
		return nil, err
	}

	if meter.Serial == "" {
		return nil, errors.New("no serial")
	}

	dataG := provider.Cached(func() (tqemData, error) {
		var res tqemData

		uri := fmt.Sprintf("%s/mum-webservice/data.php", base)
		err := client.GetJSON(uri, &res)

		if err == nil && res.Serial == "" {
			data := url.Values{
				"login":    {meter.Serial},
				"password": {cc.Password},
			}

			var req *http.Request
			req, err = request.New(http.MethodPost, fmt.Sprintf("%s/start.php", base), strings.NewReader(data.Encode()), request.URLEncoding)

			if err == nil {
				_, err = client.DoBody(req)
			}

			if err == nil {
				err = client.GetJSON(uri, &res)
			}
		}

		if err == nil && res.Serial == "" {
			err = errors.New("authentication failed")
		}

		return res, err
	}, cc.Cache)

	m := &TqEm{
		dataG: dataG,
	}

	res, err := dataG()
	if err != nil {
		return nil, err
	}

	if res.Obis31_4_0 != nil {
		return decorateTqEm(m, m.currents), nil
	}

	return m, nil
}

func (m *TqEm) CurrentPower() (float64, error) {
	res, err := m.dataG()
	return res.Obis1_4_0 - res.Obis2_4_0, err
}

var _ api.MeterEnergy = (*TqEm)(nil)

func (m *TqEm) TotalEnergy() (float64, error) {
	res, err := m.dataG()
	return res.Obis1_8_0 / 1e3, err
}

func (m *TqEm) currents() (float64, float64, float64, error) {
	res, err := m.dataG()
	if err != nil {
		return 0, 0, 0, err
	}
	return *res.Obis31_4_0, *res.Obis51_4_0, *res.Obis71_4_0, nil
}
