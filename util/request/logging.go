package request

import (
	"net/http"
	"net/http/httputil"

	"github.com/evcc-io/evcc/util"
)

type LoggingTripper struct {
	log  *util.Logger
	next http.RoundTripper
}

func NewLoggingTripper(log *util.Logger, next http.RoundTripper) *LoggingTripper {
	return &LoggingTripper{
		log:  log,
		next: next,
	}
}

func (t *LoggingTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Dump request
	reqDump, err := httputil.DumpRequestOut(req, true)
	if err == nil {
		t.log.INFO.Printf("HTTP Request:\n%s", string(reqDump))
	}

	// Execute request
	resp, err := t.next.RoundTrip(req)
	if err != nil {
		return resp, err
	}

	// Dump response
	respDump, err := httputil.DumpResponse(resp, true)
	if err == nil {
		t.log.INFO.Printf("HTTP Response:\n%s", string(respDump))
	}

	return resp, nil
}
