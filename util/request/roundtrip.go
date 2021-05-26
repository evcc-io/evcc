package request

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/andig/evcc/util"
)

type roundTripper struct {
	log  *util.Logger
	base http.RoundTripper
}

const max = 2048 * 2

// NewTripper creates a logging roundtrip handler
func NewTripper(log *util.Logger, base http.RoundTripper) http.RoundTripper {
	tripper := &roundTripper{
		log:  log,
		base: base,
	}

	return tripper
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (r *roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	var bld strings.Builder
	bld.WriteString(fmt.Sprintf("%s %s", req.Method, req.URL.String()))

	if body, err := httputil.DumpRequestOut(req, true); err == nil {
		bld.WriteString("\n")
		bld.Write(bytes.TrimSpace(body[:min(max, len(body))]))
	}

	resp, err := r.base.RoundTrip(req)

	if resp != nil {
		if body, err := httputil.DumpResponse(resp, true); err == nil {
			bld.WriteString("\n\n")
			bld.Write(bytes.TrimSpace(body[:min(max, len(body))]))
		}
	}

	if bld.Len() > 0 {
		r.log.TRACE.Println(bld.String())
	}

	return resp, err
}
