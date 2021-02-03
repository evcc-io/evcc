package tesla

import (
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/andig/evcc/util"
)

type roundTripper struct {
	log       *util.Logger
	transport http.RoundTripper
}

const max = 2048

func (r *roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if body, err := httputil.DumpRequest(req, true); err == nil {
		s := strings.TrimSpace(string(body))
		if len(s) > max {
			s = s[:max]
		}
		r.log.TRACE.Println(s)
	}

	resp, err := r.transport.RoundTrip(req)

	if resp != nil {
		if body, err := httputil.DumpResponse(resp, true); err == nil {
			s := strings.TrimSpace(string(body))
			if len(s) > max {
				s = s[:max]
			}
			r.log.TRACE.Println(s)
		}
	}

	return resp, err
}
