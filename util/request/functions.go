package request

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

var (
	// URLEncoding specifies application/x-www-form-urlencoded
	URLEncoding = map[string]string{"Content-Type": "application/x-www-form-urlencoded"}

	// JSONEncoding specifies application/json
	JSONEncoding = map[string]string{
		"Content-Type": "application/json",
		"Accept":       "application/json",
	}
)

// StatusError indicates unsuccessful http response
type StatusError struct {
	resp *http.Response
}

// NewStatusError create new StatusError for given response
func NewStatusError(resp *http.Response) StatusError {
	return StatusError{resp: resp}
}

func (e StatusError) Error() string {
	return fmt.Sprintf("unexpected status: %d", e.resp.StatusCode)
}

// Response returns the respose with the unexpected error
func (e StatusError) Response() *http.Response {
	return e.resp
}

// StatusCode returns the respose's status code
func (e StatusError) StatusCode() int {
	return e.resp.StatusCode
}

// HasStatus returns true if the respose's status code matches any of the given codes
func (e StatusError) HasStatus(codes ...int) bool {
	for _, code := range codes {
		if e.resp.StatusCode == code {
			return true
		}
	}
	return false
}

// ReadBody reads HTTP response and returns error on response codes other than HTTP 2xx. It closes the request body after reading.
func ReadBody(resp *http.Response) ([]byte, error) {
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return []byte{}, err
	}

	// maintain body after reading
	resp.Body = io.NopCloser(bytes.NewBuffer(b))

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return b, StatusError{resp: resp}
	}

	return b, nil
}

// DecodeJSON reads HTTP response and decodes JSON body if error is nil
func DecodeJSON(resp *http.Response, res interface{}) error {
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return StatusError{resp: resp}
	}

	return json.NewDecoder(resp.Body).Decode(&res)
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

	return req, err
}
