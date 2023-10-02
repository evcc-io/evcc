package transport

import "net/http"

// DecorateHeaders wraps the given http.Request with a decorator that adds the given parameters to the request headers.
func DecorateHeaders(headers map[string]string) func(req *http.Request) error {
	return func(req *http.Request) error {
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		return nil
	}
}

// DecorateQuery wraps the given http.Request with a decorator that adds the given parameters to the GET query string.
func DecorateQuery(params map[string]string) func(req *http.Request) error {
	return func(req *http.Request) error {
		q := req.URL.Query()
		for k, v := range params {
			q.Add(k, v)
		}
		req.URL.RawQuery = q.Encode()
		return nil
	}
}
