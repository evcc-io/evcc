package request

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
)

var (
	// URLEncoding specifies application/x-www-form-urlencoded
	URLEncoding = map[string]string{"Content-Type": "application/x-www-form-urlencoded"}

	// JSONEncoding specifies application/json
	JSONEncoding = map[string]string{"Content-Type": "application/json"}
)

// ReadBody reads HTTP response and returns error on response codes other than HTTP 2xx
func ReadBody(resp *http.Response) ([]byte, error) {
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []byte{}, err
	}

	// maintain body after reading
	resp.Body = ioutil.NopCloser(bytes.NewBuffer(b))

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return b, fmt.Errorf("unexpected response %d: %s", resp.StatusCode, string(b))
	}

	return b, nil
}

// DecodeJSON reads HTTP response and decodes JSON body if error is nil
func DecodeJSON(resp *http.Response, res interface{}) error {
	b, err := ReadBody(resp)
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
