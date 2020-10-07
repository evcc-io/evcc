package request

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	"github.com/andig/evcc/util"
)

// Helper provides utility primitives
type Helper struct {
	*http.Client
	log  *log.Logger
	last *http.Response // last response
}

// NewHelper creates http helper for simplified PUT GET logic
func NewHelper(log *util.Logger) *Helper {
	r := &Helper{
		Client: &http.Client{Timeout: 10 * time.Second},
	}

	// add logger
	if log != nil {
		r.log = log.TRACE
	}

	// intercept for logging
	r.Transport(http.DefaultTransport)

	return r
}

// LastResponse returns last http.Response that was read without error
func (r *Helper) LastResponse() *http.Response {
	return r.last
}

type helperTransport struct {
	log          *log.Logger
	detailed     bool
	lastResponse func(*http.Response)
	roundTripper http.RoundTripper
}

func (r *helperTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := r.roundTripper.RoundTrip(req)
	r.lastResponse(resp)

	if r.log != nil {
		msg := fmt.Sprintf("%s %s", req.Method, req.URL)
		if resp != nil {
			if r.detailed {
				if body, err := httputil.DumpResponse(resp, true); err == nil {
					msg += "\n" + strings.TrimSpace(string(body))
				}
			} else {
				msg += "\n" + resp.Status

				if body, _ := ReadBody(resp); len(body) > 0 {
					const max = 2048

					str := string(body)
					if len(str) >= max {
						str = str[:max]
					}

					msg += "\n" + strings.TrimSpace(str)
				}
			}
		}
		r.log.Println(msg)
	}

	return resp, err
}

// Transport wraps the provided transport with logging and sets it as client transport
func (r *Helper) Transport(roundTripper http.RoundTripper) {
	r.Client.Transport = &helperTransport{
		log:          r.log,
		roundTripper: roundTripper,
		lastResponse: func(resp *http.Response) {
			r.last = resp
		},
	}
}

// DoBody executes HTTP request and returns the response body
func (r *Helper) DoBody(req *http.Request) ([]byte, error) {
	resp, err := r.Do(req)
	var body []byte
	if err == nil {
		body, err = ReadBody(resp)
	}
	return body, err
}

// GetBody executes HTTP GET request and returns the response body
func (r *Helper) GetBody(url string) ([]byte, error) {
	resp, err := r.Get(url)
	var body []byte
	if err == nil {
		body, err = ReadBody(resp)
	}
	return body, err
}

// DoJSON executes HTTP request and decodes JSON response
func (r *Helper) DoJSON(req *http.Request, res interface{}) error {
	resp, err := r.Do(req)
	if err == nil {
		err = DecodeJSON(resp, &res)
	}
	return err
}

// GetJSON executes HTTP GET request and decodes JSON response
func (r *Helper) GetJSON(url string, res interface{}) error {
	resp, err := r.Get(url)
	if err == nil {
		err = DecodeJSON(resp, &res)
	}
	return err
}
