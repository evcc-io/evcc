package util

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type HTTPHelper struct {
	Log    *Logger
	Client *http.Client
}

// NewHTTPHelper creates http helper for simplified PUT GET logic
func NewHTTPHelper(log *Logger) *HTTPHelper {
	r := &HTTPHelper{
		Log:    log,
		Client: &http.Client{},
	}
	return r
}

// Response codes other than HTTP 200 or 204 are raised as error
func (r *HTTPHelper) readBody(resp *http.Response, err error) ([]byte, error) {
	if err != nil {
		return []byte{}, err
	}
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []byte{}, err
	}

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

// Request executes HTTP request returns the response body
func (r *HTTPHelper) Request(req *http.Request) ([]byte, error) {
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

	resp, err := r.Client.Do(req)
	return r.readBody(resp, err)
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

// PutJSON executes HTTP PUT request and returns the response body
func (r *HTTPHelper) PutJSON(url string, data interface{}) ([]byte, error) {
	body, err := json.Marshal(data)
	if err != nil {
		return []byte{}, err
	}

	return r.Put(url, body)
}
