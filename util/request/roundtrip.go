package request

import (
	"bytes"
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

// NewTripper creates a logging roundtrip handler
func NewTripper(log *util.Logger, transport http.RoundTripper) http.RoundTripper {
	tripper := &roundTripper{
		log:       log,
		transport: transport,
	}

	return tripper
}

var bld = strings.Builder{}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (r *roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	bld.Reset()

	if body, err := httputil.DumpRequestOut(req, true); err == nil {
		bld.WriteString("\n")
		bld.Write(bytes.TrimSpace(body[:min(max, len(body))]))
	}

	resp, err := r.transport.RoundTrip(req)

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
