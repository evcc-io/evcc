package transport

import "net/http"

func DecorateHeaders(headers map[string]string) func(req *http.Request) error {
	return func(req *http.Request) error {
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		return nil
	}
}
