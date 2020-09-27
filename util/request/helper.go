package request

import (
	"net/http"
	"net/http/httputil"
	"time"

	"github.com/andig/evcc/util"
)

// Helper provides utility primitives
type Helper struct {
	Log    *util.Logger
	Client *http.Client
	last   *http.Response // last response
}

// NewHelper creates http helper for simplified PUT GET logic
func NewHelper(log *util.Logger) *Helper {
	r := &Helper{
		Log:    log,
		Client: &http.Client{Timeout: 10 * time.Second},
	}

	// intercept for logging
	r.Client.Transport = r

	return r
}

// LastResponse returns last http.Response that was read without error
func (r *Helper) LastResponse() *http.Response {
	return r.last
}

// RoundTrip implements http.Roundtripper
func (r *Helper) RoundTrip(req *http.Request) (*http.Response, error) {
	println("TRIPPER")

	if r.Log != nil {
		r.Log.TRACE.Println(req.RequestURI)
	}

	resp, err := http.DefaultTransport.RoundTrip(req)
	r.last = resp

	if err == nil {
		if b, err := httputil.DumpResponse(resp, false); err == nil {
			if r.Log != nil {
				r.Log.TRACE.Println(string(b))
			}
		}
	}

	return resp, err
}

// Do executes HTTP request and returns the response body
func (r *Helper) Do(req *http.Request) ([]byte, error) {
	resp, err := r.Client.Do(req)
	return ReadBody(resp, err)
}

// Get executes HTTP GET request and returns the response body
func (r *Helper) Get(url string) ([]byte, error) {
	resp, err := r.Client.Get(url)
	return ReadBody(resp, err)
}

// RequestJSON executes HTTP request and decodes JSON response
func (r *Helper) RequestJSON(req *http.Request, res interface{}) error {
	resp, err := r.Client.Do(req)
	return DecodeJSON(resp, err, res)
}

// GetJSON executes HTTP GET request and decodes JSON response
func (r *Helper) GetJSON(url string, res interface{}) error {
	resp, err := r.Client.Get(url)
	return DecodeJSON(resp, err, res)
}
