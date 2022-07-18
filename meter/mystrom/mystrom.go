package mystrom

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

type Report struct {
	Power float64
	Relay bool
}

type Connection struct {
	*request.Helper
	uri string
}

func NewConnection(uri string) *Connection {
	return &Connection{
		uri:    util.DefaultScheme(strings.TrimSuffix(uri, "/"), "http"),
		Helper: request.NewHelper(util.NewLogger("mystrom")),
	}
}

func (c *Connection) Request(path string) error {
	uri := fmt.Sprintf("%s/%s", c.uri, path)
	req, err := request.New(http.MethodGet, uri, nil, nil)
	if err == nil {
		_, err = c.DoBody(req)
	}
	return err
}

func (c *Connection) Report() (Report, error) {
	var res Report
	uri := fmt.Sprintf("%s/report", c.uri)
	err := c.GetJSON(uri, &res)
	return res, err
}

var _ api.Meter = (*Connection)(nil)

// CurrentPower implements the api.Meter interface
func (c *Connection) CurrentPower() (float64, error) {
	res, err := c.Report()
	return res.Power, err
}
