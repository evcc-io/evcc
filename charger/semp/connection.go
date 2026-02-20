package semp

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

// Connection represents a SEMP HTTP connection with helper methods
type Connection struct {
	*request.Helper
	uri     string
	mu      sync.Mutex
	updated time.Time
}

// NewConnection creates a new SEMP client
func NewConnection(log *util.Logger, uri string) *Connection {
	// Ensure URI ends with exactly one trailing slash
	uri = strings.TrimRight(uri, "/") + "/"

	return &Connection{
		Helper: request.NewHelper(log),
		uri:    uri,
	}
}

// GetDeviceXML retrieves the complete SEMP document from the base URL
func (c *Connection) GetDeviceXML() (Device2EM, error) {
	var response Device2EM
	if err := c.GetXML(c.uri, &response); err != nil {
		return Device2EM{}, err
	}
	return response, nil
}

// GetParametersXML retrieves device parameters from the /Parameters endpoint
func (c *Connection) GetParametersXML() ([]Parameter, error) {
	var response Device2EM
	uri := c.uri + "Parameters"
	if err := c.GetXML(uri, &response); err != nil {
		return nil, err
	}

	if response.Parameters == nil {
		return []Parameter{}, nil
	}

	return response.Parameters.Parameter, nil
}

// SendDeviceControl sends a control message to the SEMP device
// power is optional - if nil, RecommendedPowerConsumption will be omitted
func (c *Connection) SendDeviceControl(deviceId string, power int) error {
	control := DeviceControl{
		DeviceID:  deviceId,
		On:        power > 0,
		Timestamp: int(time.Now().Unix()),
	}

	if power > 0 {
		control.RecommendedPowerConsumption = &power
	}

	message := EM2Device{
		Xmlns:         "http://www.sma.de/communication/schema/SEMP/v1",
		DeviceControl: []DeviceControl{control},
	}

	req, err := request.New(http.MethodPost, c.uri, request.MarshalXML(message), request.XMLEncoding)
	if err != nil {
		return err
	}

	_, err = c.DoBody(req)
	if err == nil {
		c.mu.Lock()
		c.updated = time.Now()
		c.mu.Unlock()
	}
	return err
}

// Updated returns the last successful device control update time
func (c *Connection) Updated() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.updated
}
