package request

import (
	"bytes"
	"net/http"
	"net/http/httputil"
	"strconv"
	"strings"
	"time"

	"github.com/evcc-io/evcc/util/logx"
	"github.com/prometheus/client_golang/prometheus"
)

type roundTripper struct {
	log  logx.Logger
	base http.RoundTripper
}

const max = 1024 * 64

var (
	reqMetric *prometheus.SummaryVec
	resMetric *prometheus.CounterVec
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
func NewTripper(log logx.Logger, base http.RoundTripper) http.RoundTripper {
	tripper := &roundTripper{
		log:  logx.TraceLevel(log),
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
	_ = r.log.Log("method", req.Method, "uri", req.URL.String())

	var bld strings.Builder
	if body, err := httputil.DumpRequestOut(req, true); err == nil {
		bld.WriteString("\n")
		bld.Write(bytes.TrimSpace(body[:min(max, len(body))]))
	}

	startTime := time.Now()
	resp, err := r.base.RoundTrip(req)

	reqMetric.WithLabelValues(req.URL.Hostname()).Observe(time.Since(startTime).Seconds())

	if err == nil {
		resMetric.WithLabelValues(req.URL.Hostname(), strconv.Itoa(resp.StatusCode)).Add(1)

		if body, err := httputil.DumpResponse(resp, true); err == nil {
			bld.WriteString("\n\n")
			bld.Write(bytes.TrimSpace(body[:min(max, len(body))]))
		}
	} else {
		resMetric.WithLabelValues(req.URL.Hostname(), "999").Add(1)
	}

	if bld.Len() > 0 {
		_ = r.log.Log("payload", bld.String())
	}

	return resp, err
}
