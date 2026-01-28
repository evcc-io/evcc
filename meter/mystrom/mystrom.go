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
	uri   string
	token string
}

func NewConnection(uri, token string) *Connection {
	return &Connection{
		uri:    util.DefaultScheme(strings.TrimSuffix(uri, "/"), "http"),
		token:  token,
		Helper: request.NewHelper(util.NewLogger("mystrom")),
	}
}

func (c *Connection) Request(path string) error {
	uri := fmt.Sprintf("%s/%s", c.uri, path)
	req, _ := request.New(http.MethodGet, uri, nil, nil)
	if c.token != "" {
		req.Header.Set("Token", c.token)
	}
	_, err := c.DoBody(req)
	return err
}

func (c *Connection) Report() (Report, error) {
	var res Report
	uri := fmt.Sprintf("%s/report", c.uri)
	req, _ := request.New(http.MethodGet, uri, nil, request.AcceptJSON)
	if c.token != "" {
		req.Header.Set("Token", c.token)
	}
	err := c.DoJSON(req, &res)
	return res, err
}

var _ api.Meter = (*Connection)(nil)

// CurrentPower implements the api.Meter interface
func (c *Connection) CurrentPower() (float64, error) {
	res, err := c.Report()
	return res.Power, err
}
