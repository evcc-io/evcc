package shelly

import (
	"fmt"
	"net/http"
	"slices"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/jpfielding/go-http-digest/pkg/digest"
)

type gen2Conn struct {
	*request.Helper
	uri     string
	channel int
	methods []string
}

type Gen2RpcPost struct {
	Id     int    `json:"id"`
	On     bool   `json:"on"`
	Src    string `json:"src"`
	Method string `json:"method"`
}

type Gen2Methods struct {
	Methods []string
}

// newGen2Conn initializes the connection to the shelly gen2+ api and sets up the cached gen2SwitchStatus, gen2EM1Status and gen2EMStatus
func newGen2Conn(helper *request.Helper, uri string, user, password string, channel int) (*gen2Conn, error) {
	// Shelly GEN 2+ API
	// https://shelly-api-docs.shelly.cloud/gen2/
	c := &gen2Conn{
		Helper:  helper,
		channel: channel,
		uri:     fmt.Sprintf("%s/rpc", util.DefaultScheme(uri, "http")),
	}

	// Shelly gen 2 rfc7616 authentication
	// https://shelly-api-docs.shelly.cloud/gen2/General/Authentication
	if user != "" {
		c.Client.Transport = digest.NewTransport(user, password, c.Client.Transport)
	}

	var res Gen2Methods
	if err := c.execCmd("Shelly.ListMethods", false, &res); err != nil {
		return nil, err
	}

	c.methods = res.Methods

	return c, nil
}

// execCmd executes a shelly api gen2+ command and provides the response
func (c *gen2Conn) execCmd(method string, enable bool, res any) error {
	data := &Gen2RpcPost{
		Id:     c.channel,
		On:     enable,
		Src:    "evcc",
		Method: method,
	}

	req, err := request.New(http.MethodPost, fmt.Sprintf("%s/%s", c.uri, method), request.MarshalJSON(data), request.JSONEncoding)
	if err != nil {
		return err
	}

	return c.DoJSON(req, &res)
}

// https://shelly-api-docs.shelly.cloud/gen2/ComponentsAndServices/Switch#switchgetstatus-example
func (c *gen2Conn) hasSwitchEndpoint() bool {
	return c.hasMethod("Switch.GetStatus")
}

// https://shelly-api-docs.shelly.cloud/gen2/ComponentsAndServices/EM1#em1getstatus-example
func (c *gen2Conn) hasEM1Endpoint() bool {
	return slices.Contains(c.methods, "EM1.GetStatus")
}

func (c *gen2Conn) hasEMEndpoint() bool {
	return slices.Contains(c.methods, "EM.GetStatus")
}

func (c *gen2Conn) hasMethod(method string) bool {
	return slices.Contains(c.methods, method)
}
