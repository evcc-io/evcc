package request

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httputil"
	"strconv"
	"strings"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sandrolain/httpcache"
)

type roundTripper struct {
	log  *util.Logger
	base http.RoundTripper
}

var (
	LogHeaders bool
	LogMaxLen  = 1024 * 8
	reqMetric  *prometheus.SummaryVec
	resMetric  *prometheus.CounterVec
)

func init() {
	labels := []string{"host"}

	reqMetric = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Namespace: "evcc",
		Subsystem: "http",
		Name:      "request_duration_seconds",
		Help:      "A summary of HTTP request durations",
		Objectives: map[float64]float64{
			0.5:  0.05,  // 50th percentile with a max. absolute error of 0.05
			0.9:  0.01,  // 90th percentile with a max. absolute error of 0.01
			0.99: 0.001, // 99th percentile with a max. absolute error of 0.001
		},
	}, labels)

	resMetric = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "evcc",
		Subsystem: "http",
		Name:      "request_total",
		Help:      "Total count of HTTP requests",
	}, append(labels, "status"))

	prometheus.MustRegister(reqMetric, resMetric)
}

// NewTripper creates a logging roundtrip handler
func NewTripper(log *util.Logger, base http.RoundTripper) http.RoundTripper {
	tripper := &roundTripper{
		log:  log,
		base: base,
	}

	return tripper
}

func isWebSocket(req *http.Request) bool {
	// WebSocket handshake must be GET
	if req.Method != http.MethodGet {
		return false
	}

	// Must contain: Connection: Upgrade
	if !headerContainsToken(req.Header, "Connection", "Upgrade") {
		return false
	}

	// Must contain: Upgrade: websocket
	if !headerContainsToken(req.Header, "Upgrade", "websocket") {
		return false
	}

	// Must contain WebSocket-specific headers
	if req.Header.Get("Sec-WebSocket-Key") == "" {
		return false
	}

	if req.Header.Get("Sec-WebSocket-Version") == "" {
		return false
	}

	return true
}

func headerContainsToken(h http.Header, key, token string) bool {
	for _, v := range h.Values(key) {
		for part := range strings.SplitSeq(v, ",") {
			if strings.EqualFold(strings.TrimSpace(part), token) {
				return true
			}
		}
	}
	return false
}

// copy of http.drainBody
func drainBody(b io.ReadCloser) (r1, r2 io.ReadCloser, err error) {
	if b == nil || b == http.NoBody {
		// No copying needed. Preserve the magic sentinel meaning of NoBody.
		return http.NoBody, http.NoBody, nil
	}
	var buf bytes.Buffer
	if _, err = buf.ReadFrom(b); err != nil {
		return nil, b, err
	}
	if err = b.Close(); err != nil {
		return nil, b, err
	}
	return io.NopCloser(&buf), io.NopCloser(bytes.NewReader(buf.Bytes())), nil
}

// dump http request/response body
func dump(r io.ReadCloser, w *strings.Builder) error {
	body, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	if w.Len() > 0 && len(body) > 0 {
		w.WriteString("\n--\n")
	}
	_, err = w.Write(bytes.TrimSpace(body[:min(LogMaxLen, len(body))]))
	return err
}

func (r *roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// add evcc user agent
	if req.Header.Get("User-Agent") == "" {
		req = req.Clone(req.Context())
		req.Header.Set("User-Agent", "evcc/"+util.FormattedVersion())
	}

	// dump without headers
	var err error
	var save io.ReadCloser

	bld := new(strings.Builder)

	isWebSocketReq := isWebSocket(req)
	if !isWebSocketReq {
		if LogHeaders {
			if body, err := httputil.DumpRequestOut(req, true); err == nil {
				bld.WriteString("\n")
				bld.Write(bytes.TrimSpace(body[:min(LogMaxLen, len(body))]))
			}
		} else {
			if save, req.Body, err = drainBody(req.Body); err == nil {
				err = dump(save, bld)
			}
			if err != nil {
				return nil, err
			}
		}
	}

	startTime := time.Now()
	resp, err := r.base.RoundTrip(req)

	reqMetric.WithLabelValues(req.URL.Hostname()).Observe(time.Since(startTime).Seconds())

	var cached string
	if err == nil && resp.Header.Get(httpcache.XFromCache) != "" {
		cached = " CACHED"
	}
	r.log.TRACE.Printf("%s%s %s", req.Method, cached, req.URL.String())

	if err == nil {
		resMetric.WithLabelValues(req.URL.Hostname(), strconv.Itoa(resp.StatusCode)).Add(1)

		if !isWebSocketReq {
			if LogHeaders {
				if body, err := httputil.DumpResponse(resp, true); err == nil {
					bld.WriteString("\n\n")
					bld.Write(bytes.TrimSpace(body[:min(LogMaxLen, len(body))]))
				}
			} else {
				if save, resp.Body, err = drainBody(resp.Body); err == nil {
					err = dump(save, bld)
				}
				if err != nil {
					return nil, err
				}
			}
		}
	} else {
		resMetric.WithLabelValues(req.URL.Hostname(), "999").Add(1)
	}

	if bld.Len() > 0 {
		r.log.TRACE.Println(bld.String())
	}

	return resp, err
}
