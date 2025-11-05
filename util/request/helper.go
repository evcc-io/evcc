package request

import (
	"encoding/json"
	"encoding/xml"
	"net/http"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/transport"
)

// Timeout is the default request timeout used by the Helper
var Timeout = 10 * time.Second

// Helper provides utility primitives
type Helper struct {
	*http.Client
}

// NewClient creates http client with default transport
func NewClient(log *util.Logger) *http.Client {
	return &http.Client{
		Timeout:   Timeout,
		Transport: NewTripper(log, transport.Default()),
	}
}

// NewHelper creates http helper for simplified PUT GET logic
func NewHelper(log *util.Logger) *Helper {
	return &Helper{
		Client: NewClient(log),
	}
}

// DoBody executes HTTP request and returns the response body
func (r *Helper) DoBody(req *http.Request) ([]byte, error) {
	resp, err := r.Do(req)
	if err != nil {
		return nil, err
	}

	return ReadBody(resp)
}

// GetBody executes HTTP GET request and returns the response body
func (r *Helper) GetBody(url string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	return r.DoBody(req)
}

// decodeJSON reads HTTP response and decodes JSON body if error is nil
func decodeJSON(resp *http.Response, res any) error {
	if err := ResponseError(resp); err != nil {
		_ = json.NewDecoder(resp.Body).Decode(&res)
		return err
	}

	return json.NewDecoder(resp.Body).Decode(&res)
}

// decodeXML reads HTTP response and decodes XML body if error is nil
func decodeXML(resp *http.Response, res any) error {
	if err := ResponseError(resp); err != nil {
		_ = xml.NewDecoder(resp.Body).Decode(&res)
		return err
	}

	return xml.NewDecoder(resp.Body).Decode(&res)
}

// DoJSON executes HTTP request and decodes JSON response.
// It returns a StatusError on response codes other than HTTP 2xx.
func (r *Helper) DoJSON(req *http.Request, res any) error {
	resp, err := r.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		return nil
	}

	return decodeJSON(resp, &res)
}

// GetJSON executes HTTP GET request and decodes JSON response.
// It returns a StatusError on response codes other than HTTP 2xx.
func (r *Helper) GetJSON(url string, res any) error {
	req, err := New(http.MethodGet, url, nil, AcceptJSON)
	if err != nil {
		return err
	}

	return r.DoJSON(req, &res)
}

// DoXML executes HTTP request and decodes XML response.
// It returns a StatusError on response codes other than HTTP 2xx.
func (r *Helper) DoXML(req *http.Request, res any) error {
	resp, err := r.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		return nil
	}

	return decodeXML(resp, res)
}

// GetXML executes HTTP GET request and decodes XML response.
// It returns a StatusError on response codes other than HTTP 2xx.
func (r *Helper) GetXML(url string, res any) error {
	req, err := New(http.MethodGet, url, nil, AcceptXML)
	if err != nil {
		return err
	}

	return r.DoXML(req, res)
}
