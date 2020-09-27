package request

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"time"

	"github.com/andig/evcc/util"
)

var (
	// URLEncoding specifies application/x-www-form-urlencoded
	URLEncoding = map[string]string{"Content-Type": "application/x-www-form-urlencoded"}

	// JSONEncoding specifies application/json
	JSONEncoding = map[string]string{"Content-Type": "application/json"}
)

// ReadBody reads HTTP response and returns error on response codes other than HTTP 200 or 204
func ReadBody(resp *http.Response, err error) ([]byte, error) {
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

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return b, fmt.Errorf("unexpected response %d: %s", resp.StatusCode, string(b))
	}

	return b, nil
}

// DecodeJSON reads HTTP response and decodes JSON body
func DecodeJSON(resp *http.Response, err error, res interface{}) error {
	b, err := ReadBody(resp, err)
	if err == nil {
		err = json.Unmarshal(b, &res)
	}

	return err
}

// New builds and executes HTTP request and returns the response
func New(method, uri string, data io.Reader, headers ...map[string]string) (*http.Request, error) {
	req, err := http.NewRequest(method, uri, data)
	if err == nil {
		for _, headers := range headers {
			for k, v := range headers {
				req.Header.Add(k, v)
			}
		}
	}

	return req, nil
}

type errorReader struct {
	err error
}

func (r *errorReader) Read(p []byte) (int, error) {
	return 0, r.err
}

// MarshalJSON marshals JSON into an io.Reader
func MarshalJSON(data interface{}) io.Reader {
	body, err := json.Marshal(data)
	if err != nil {
		return &errorReader{err: err}
	}

	return bytes.NewReader(body)
}

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
