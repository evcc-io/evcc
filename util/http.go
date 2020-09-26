package util

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"time"
)

// HTTPHelper provides utility primitives
type HTTPHelper struct {
	Log    *Logger
	Client *http.Client
	last   *http.Response // last response
}

// NewHTTPHelper creates http helper for simplified PUT GET logic
func NewHTTPHelper(log *Logger) *HTTPHelper {
	r := &HTTPHelper{
		Log:    log,
		Client: &http.Client{Timeout: 10 * time.Second},
	}

	// intercept for logging
	r.Client.Transport = r

	return r
}

// LastResponse returns last http.Response that was read without error
func (r *HTTPHelper) LastResponse() *http.Response {
	return r.last
}

// RoundTrip implements http.Roundtripper
func (r *HTTPHelper) RoundTrip(req *http.Request) (*http.Response, error) {
	r.Log.TRACE.Println(req.RequestURI)

	resp, err := http.DefaultTransport.RoundTrip(req)
	if err == nil {
		if b, err := httputil.DumpResponse(resp, false); err == nil {
			r.Log.TRACE.Println(string(b))
		}
	}

	return resp, err
}

// Response codes other than HTTP 200 or 204 are raised as error
func (r *HTTPHelper) readBody(resp *http.Response, err error) ([]byte, error) {
	r.last = resp
	if err != nil {
		return []byte{}, err
	}
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []byte{}, err
	}

	// maintain body after reading
	resp.Body = ioutil.NopCloser(bytes.NewBuffer(b))

	if r.Log != nil {
		r.Log.TRACE.Printf("%s\n%s", resp.Request.URL.String(), string(b))
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return b, fmt.Errorf("unexpected response %d: %s", resp.StatusCode, string(b))
	}

	return b, nil
}

func (r *HTTPHelper) decodeJSON(resp *http.Response, err error, res interface{}) ([]byte, error) {
	b, err := r.readBody(resp, err)
	if err == nil {
		err = json.Unmarshal(b, &res)
	}

	return b, err
}

// Do executes HTTP request returns the response body
func (r *HTTPHelper) Do(req *http.Request) ([]byte, error) {
	resp, err := r.Client.Do(req)
	return r.readBody(resp, err)
}

// Get executes HTTP GET request returns the response body
func (r *HTTPHelper) Get(url string) ([]byte, error) {
	resp, err := r.Client.Get(url)
	return r.readBody(resp, err)
}

// Put executes HTTP PUT request returns the response body
func (r *HTTPHelper) Put(url string, body []byte) ([]byte, error) {
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(body))
	if err != nil {
		return []byte{}, err
	}

	return r.Do(req)
}

// RequestJSON executes HTTP request and decodes JSON response
func (r *HTTPHelper) RequestJSON(req *http.Request, res interface{}) ([]byte, error) {
	resp, err := r.Client.Do(req)
	return r.decodeJSON(resp, err, res)
}

// GetJSON executes HTTP GET request and decodes JSON response
func (r *HTTPHelper) GetJSON(url string, res interface{}) ([]byte, error) {
	resp, err := r.Client.Get(url)
	return r.decodeJSON(resp, err, res)
}
