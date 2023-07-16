package request

import (
	"fmt"
	"net/http"
)

// DontFollow is a redirect policy that does not follow redirects
func DontFollow(req *http.Request, via []*http.Request) error {
	return http.ErrUseLastResponse
}

type InterceptResult = func() (string, error)

// InterceptRedirect captures a redirect url parameter
func InterceptRedirect(param string, stop bool) (func(req *http.Request, via []*http.Request) error, InterceptResult) {
	var val string
	return func(req *http.Request, via []*http.Request) error {
			if val == "" {
				if val = req.URL.Query().Get(param); val != "" && stop {
					return http.ErrUseLastResponse
				}
			}
			return nil
		},
		func() (string, error) {
			var err error
			if val == "" {
				err = fmt.Errorf("%s not found", param)
			}
			return val, err
		}
}
