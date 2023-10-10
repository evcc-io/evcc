package autonomic

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"golang.org/x/oauth2"
)

const ApiURI = "https://api.autonomic.ai/v1beta/telemetry/sources/fordpass"

var (
	ApplicationID = "667D773E-1BDC-4139-8AD0-2B16474E8DC7"
	Dyna          = "MT_3_30_2352378557_3-0_" + "uuidv4()" + "_0_789_87"
)

// API is the Ford api client
type API struct {
	*request.Helper
}

// NewAPI creates a new api client
func NewAPI(log *util.Logger, ts oauth2.TokenSource) *API {
	v := &API{
		Helper: request.NewHelper(log),
	}

	v.Client.Transport = &transport.Decorator{
		Decorator: func(req *http.Request) error {
			token, err := ts.Token()
			if err != nil {
				return err
			}

			for k, v := range map[string]string{
				"Content-type":   request.JSONContent,
				"Application-Id": ApplicationID,
				"x-dynatrace":    Dyna,
				"Authorization":  "Bearer " + token.AccessToken,
			} {
				req.Header.Set(k, v)
			}

			return nil
		},
		Base: v.Client.Transport,
	}

	return v
}

// RefreshResult retrieves a refresh result using :query
func (v *API) Status(vin string) (MetricsResponse, error) {
	var res MetricsResponse

	uri := fmt.Sprintf("%s/vehicles/%s:query", ApiURI, vin)
	req, err := request.New(http.MethodPost, uri, strings.NewReader("{}"), request.JSONEncoding)
	if err == nil {
		err = v.DoJSON(req, &res)
		// b, _ := v.DoBody(req)
		// fmt.Println(string(b))
	}

	return res, err
}
