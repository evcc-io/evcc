package request

import (
	"fmt"
	"io"
	"net/http"
	"slices"

	"github.com/cenkalti/backoff/v4"
)

var (
	FormContent  = "application/x-www-form-urlencoded"
	JSONContent  = "application/json"
	PlainContent = "text/plain"
	XMLContent   = "application/xml"

	// URLEncoding specifies application/x-www-form-urlencoded
	URLEncoding = map[string]string{"Content-Type": FormContent}

	// JSONEncoding specifies application/json
	JSONEncoding = map[string]string{
		"Content-Type": JSONContent,
		"Accept":       JSONContent,
	}

	// AcceptJSON accepting application/json
	AcceptJSON = map[string]string{
		"Accept": JSONContent,
	}

	// XMLEncoding specifies application/xml
	XMLEncoding = map[string]string{
		"Content-Type": XMLContent,
		"Accept":       XMLContent,
	}

	// AcceptXML accepting application/xml
	AcceptXML = map[string]string{
		"Accept": XMLContent,
	}
)

// StatusError indicates unsuccessful http response
type StatusError struct {
	resp *http.Response
}

func NewStatusError(resp *http.Response) *StatusError {
	return &StatusError{resp: resp}
}

func (e *StatusError) Error() string {
	req := e.resp.Request
	return fmt.Sprintf("unexpected status: %d (%s) %s %s", e.resp.StatusCode, http.StatusText(e.resp.StatusCode), req.Method, req.URL)
}

// Response returns the response with the unexpected error
func (e *StatusError) Response() *http.Response {
	return e.resp
}

// StatusCode returns the response's status code
func (e *StatusError) StatusCode() int {
	return e.resp.StatusCode
}

// HasStatus returns true if the response's status code matches any of the given codes
func (e *StatusError) HasStatus(codes ...int) bool {
	return slices.Contains(codes, e.resp.StatusCode)
}

// ResponseError turns an HTTP status code into an error
func ResponseError(resp *http.Response) error {
	if c := resp.StatusCode; c < 200 || c >= 300 {
		return backoff.Permanent(&StatusError{resp: resp})
	}
	return nil
}

// ReadBody reads HTTP response and returns error on response codes other than HTTP 2xx. It closes the request body after reading.
func ReadBody(resp *http.Response) ([]byte, error) {
	defer resp.Body.Close()

	if err := ResponseError(resp); err != nil {
		b, _ := io.ReadAll(resp.Body)
		return b, err
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return []byte{}, err
	}

	return b, nil
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
