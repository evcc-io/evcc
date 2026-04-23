package plugwise

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
)

// Connection holds the HTTP client and cached fetch function for a Plugwise Smile P1.
type Connection struct {
	*request.Helper
	uri   string
	dataG func() (DomainObjects, error)
}

// NewConnection creates a new Plugwise Smile P1 connection with BasicAuth transport and a cached fetch.
func NewConnection(uri, password string, cache time.Duration) (*Connection, error) {
	if uri == "" {
		return nil, errors.New("missing uri")
	}

	log := util.NewLogger("plugwise")
	basicAuth := transport.BasicAuthHeader("smile", password)
	log.Redact(password, basicAuth)

	c := &Connection{
		Helper: request.NewHelper(log),
		uri:    util.DefaultScheme(strings.TrimRight(uri, "/"), "http"),
	}
	c.Client.Transport = transport.BasicAuth("smile", password, c.Client.Transport)

	c.dataG = util.Cached(func() (DomainObjects, error) {
		var res DomainObjects
		req, err := request.New(http.MethodGet, c.uri+"/core/domain_objects", nil)
		if err != nil {
			return res, err
		}
		err = c.DoXML(req, &res)
		return res, err
	}, cache)

	return c, nil
}

var _ api.Meter = (*Connection)(nil)

// CurrentPower implements api.Meter. Returns signed net grid power in watts.
// Positive = grid import (consuming more than producing), negative = grid export.
func (c *Connection) CurrentPower() (float64, error) {
	res, err := c.dataG()
	if err != nil {
		return 0, err
	}
	consumed := res.Location.Logs.PowerWatts("electricity_consumed")
	produced := res.Location.Logs.PowerWatts("electricity_produced")
	return consumed - produced, nil
}
