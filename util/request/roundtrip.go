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
	r.log.TRACE.Printf("%s %s", req.Method, req.URL.String())

	// dump without headers
	var err error
	var save io.ReadCloser

	bld := new(strings.Builder)
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

	startTime := time.Now()
	resp, err := r.base.RoundTrip(req)

	reqMetric.WithLabelValues(req.URL.Hostname()).Observe(time.Since(startTime).Seconds())

	if err == nil {
		resMetric.WithLabelValues(req.URL.Hostname(), strconv.Itoa(resp.StatusCode)).Add(1)

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
	} else {
		resMetric.WithLabelValues(req.URL.Hostname(), "999").Add(1)
	}

	if bld.Len() > 0 {
		r.log.TRACE.Println(bld.String())
	}

	return resp, err
}
