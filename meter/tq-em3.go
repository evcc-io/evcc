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
	Authentication bool
}

type TqEm struct {
	dataG func() (tqemData, error)
	scale float64
}

// NewTqEmFromConfig creates a new configurable meter
func NewTqEmFromConfig(other map[string]interface{}) (api.Meter, error) {
	cc := struct {
		URI            string
		User, Password string
		Cache          time.Duration
	}{
		Cache: time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	log := util.NewLogger("tq-em3") //.Redact(cc.User, cc.Password)

	client := request.NewHelper(log)
	client.Jar, _ = cookiejar.New(nil)

	base := util.DefaultScheme(strings.TrimRight(cc.URI, "/"), "http")

	dataG := provider.Cached(func() (tqemData, error) {
		var res tqemData
		uri := fmt.Sprintf("%s/mum-webservice/data.php", base)
		err := client.GetJSON(uri, &res)

		if err == nil && !res.Authentication {
			data := url.Values{
				// "login":    {cc.User},
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

		if err == nil && !res.Authentication {
			err = errors.New("authentication failed")
		}

		return res, err
	}, cc.Cache)

	m := &TqEm{
		dataG: dataG,
	}

	return m, nil
}

func (m *TqEm) CurrentPower() (float64, error) {
	res, err := m.dataG()
	_ = res
	return 1, err
}

var _ api.MeterEnergy = (*TqEm)(nil)

func (m *TqEm) TotalEnergy() (float64, error) {
	res, err := m.dataG()
	_ = res
	return 1, err
}
