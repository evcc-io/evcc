package server

import (
	"fmt"
	"maps"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/site"
	"github.com/evcc-io/evcc/util"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/samber/lo"
)

const prometheusPrefix = "evcc_"

// invalidPrometheusChars matches runs of characters that are not valid in a
// Prometheus metric name. Underscores are matched too so that separators from
// snake_case conversion and nested-key flattening collapse into a single
// underscore (e.g. "grid__power" -> "grid_power").
var invalidPrometheusChars = regexp.MustCompile(`[^a-zA-Z0-9]+`)

// promSample is the last known value for a single metric+label combination
type promSample struct {
	value  float64
	labels prometheus.Labels
}

// Prometheus is a Prometheus exporter. It mirrors the internal evcc state
// (the same values published via MQTT/InfluxDB) as an in-memory metric
// snapshot that is served to Prometheus on scrape via /metrics.
type Prometheus struct {
	log *util.Logger

	mu sync.Mutex
	// samples maps metric name -> label signature -> sample
	samples map[string]map[string]promSample
}

// NewPrometheus creates a Prometheus exporter and registers it with reg
func NewPrometheus(reg prometheus.Registerer) *Prometheus {
	p := &Prometheus{
		log:     util.NewLogger("prometheus"),
		samples: make(map[string]map[string]promSample),
	}

	reg.MustRegister(p)

	return p
}

// metricName converts an evcc key such as "loadpoint_chargePower" into a
// valid, idiomatic Prometheus metric name such as "evcc_loadpoint_charge_power"
func metricName(key string) string {
	var b strings.Builder
	b.WriteString(prometheusPrefix)

	for i, r := range key {
		switch {
		case r >= 'A' && r <= 'Z':
			if i > 0 {
				b.WriteByte('_')
			}
			b.WriteRune(r - 'A' + 'a')
		default:
			b.WriteRune(r)
		}
	}

	return invalidPrometheusChars.ReplaceAllString(b.String(), "_")
}

// labelSignature returns a stable string representation of a label set,
// used as a map key to identify a unique metric+label combination
func labelSignature(labels prometheus.Labels) string {
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	for _, k := range keys {
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(labels[k])
		b.WriteByte(';')
	}

	return b.String()
}

// set stores/overwrites the current value for a metric+label combination
func (p *Prometheus) set(name string, value float64, labels prometheus.Labels) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.samples[name] == nil {
		p.samples[name] = make(map[string]promSample)
	}

	p.samples[name][labelSignature(labels)] = promSample{value: value, labels: labels}
}

// record flattens an arbitrary evcc value (as also produced for MQTT/InfluxDB)
// into one or more numeric Prometheus samples. Strings are not exported as
// metrics- their value has no useful numeric representation.
func (p *Prometheus) record(key string, val any, labels prometheus.Labels) {
	if val == nil || lo.IsNil(val) {
		return
	}

	switch v := val.(type) {
	case string:
		return

	case bool:
		p.set(metricName(key), boolToFloat(v), labels)
		return

	case int:
		p.set(metricName(key), float64(v), labels)
		return

	case int64:
		p.set(metricName(key), float64(v), labels)
		return

	case float64:
		p.set(metricName(key), v, labels)
		return

	case time.Time:
		if !v.IsZero() {
			p.set(metricName(key), float64(v.Unix()), labels)
		}
		return

	case time.Duration:
		p.set(metricName(key), v.Seconds(), labels)
		return

	case []float64:
		p.recordPhases(key, v, labels)
		return

	case [3]float64:
		p.recordPhases(key, v[:], labels)
		return
	}

	if mm, ok := val.(api.StructMarshaler); ok {
		if d, err := mm.MarshalStruct(); err == nil {
			p.record(key, d, labels)
		} else {
			p.log.ERROR.Printf("marshal struct: %v", err)
		}
		return
	}

	// BytesMarshaler (raw/binary payloads) has no numeric representation
	if _, ok := val.(api.BytesMarshaler); ok {
		return
	}

	rv := reflect.ValueOf(val)

	switch rv.Kind() {
	case reflect.Pointer:
		if !rv.IsNil() {
			p.record(key, rv.Elem().Interface(), labels)
		}

	case reflect.Struct:
		typ := rv.Type()
		for i := range typ.NumField() {
			if f := typ.Field(i); f.IsExported() {
				p.record(key+"_"+f.Name, rv.Field(i).Interface(), labels)
			}
		}

	case reflect.Slice, reflect.Array:
		for i := range rv.Len() {
			l := maps.Clone(labels)
			l["id"] = strconv.Itoa(i + 1)
			p.record(key, rv.Index(i).Interface(), l)
		}

	case reflect.Map:
		for _, k := range rv.MapKeys() {
			p.record(fmt.Sprintf("%s_%v", key, k.Interface()), rv.MapIndex(k).Interface(), labels)
		}

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		p.set(metricName(key), float64(rv.Int()), labels)

	case reflect.Float32, reflect.Float64:
		p.set(metricName(key), rv.Float(), labels)

	case reflect.Bool:
		p.set(metricName(key), boolToFloat(rv.Bool()), labels)
	}
}

// recordPhases records a 3-phase value both per-phase (with a "phase" label)
// and as an aggregated sum, matching the MQTT/InfluxDB phase handling
func (p *Prometheus) recordPhases(key string, phases []float64, labels prometheus.Labels) {
	if len(phases) != 3 {
		return
	}

	var total float64
	for i, v := range phases {
		total += v

		l := maps.Clone(labels)
		l["phase"] = strconv.Itoa(i + 1)
		p.set(metricName(key), v, l)
	}

	p.set(metricName(key), total, labels)
}

func boolToFloat(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

// Run starts the Prometheus exporter, consuming site state updates from in
// and mirroring them into the in-memory metric snapshot
func (p *Prometheus) Run(site site.API, in <-chan util.Param) {
	for param := range in {
		labels := prometheus.Labels{}

		key := param.Key
		if param.Loadpoint != nil {
			if lps := site.Loadpoints(); *param.Loadpoint < len(lps) {
				lp := lps[*param.Loadpoint]
				labels["loadpoint"] = lp.GetTitle()
				if v := lp.GetVehicle(); v != nil {
					labels["vehicle"] = v.GetTitle()
				}
			}
			key = "loadpoint_" + param.Key
		}

		p.record(key, param.Val, labels)
	}
}

// Describe implements prometheus.Collector. Metrics are dynamic and
// depend on the configured devices, so no fixed descriptors are provided
// upfront- this registers Prometheus as an "unchecked" collector.
func (p *Prometheus) Describe(chan<- *prometheus.Desc) {}

// Collect implements prometheus.Collector
func (p *Prometheus) Collect(ch chan<- prometheus.Metric) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for name, samples := range p.samples {
		for _, s := range samples {
			labelNames := make([]string, 0, len(s.labels))
			labelValues := make([]string, 0, len(s.labels))
			for k, v := range s.labels {
				labelNames = append(labelNames, k)
				labelValues = append(labelValues, v)
			}

			desc := prometheus.NewDesc(name, "evcc: "+name, labelNames, nil)

			m, err := prometheus.NewConstMetric(desc, prometheus.GaugeValue, s.value, labelValues...)
			if err != nil {
				p.log.ERROR.Printf("collect %s: %v", name, err)
				continue
			}

			ch <- m
		}
	}
}
