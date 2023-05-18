package provider

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

// Prometheus provider
type Prometheus struct {
	log     *util.Logger
	api     v1.API
	query   string
	timeout time.Duration
}

func init() {
	registry.Add("prometheus", NewPrometheusFromConfig)
}

func NewPrometheusFromConfig(other map[string]interface{}) (Provider, error) {
	cc := struct {
		Uri, Query string
		Timeout    time.Duration
	}{
		Timeout: request.Timeout,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	log := util.NewLogger("prometheus")

	config := api.Config{
		Address:      cc.Uri,
		RoundTripper: transport.Default(),
	}

	client, err := api.NewClient(config)
	if err != nil {
		return nil, err
	}

	p := NewPrometheus(log, client, cc.Query, cc.Timeout)

	return p, nil
}

func NewPrometheus(log *util.Logger, client api.Client, query string, timeout time.Duration) *Prometheus {
	p := &Prometheus{
		log:     log,
		api:     v1.NewAPI(client),
		query:   query,
		timeout: timeout,
	}

	return p
}

func (p *Prometheus) Query() (model.Value, error) {
	ctx, cancel := context.WithTimeout(context.Background(), p.timeout)
	defer cancel()

	res, warn, err := p.api.Query(ctx, p.query, time.Now())
	if err != nil {
		return nil, err
	}

	p.log.TRACE.Printf("query %q: %+v", p.query, res)
	if len(warn) > 0 {
		p.log.WARN.Printf("query %q returned warnings: %v", p.query, warn)
	}

	return res, nil
}

var _ FloatProvider = (*Prometheus)(nil)

// FloatGetter expects scalar value from query response as float
func (p *Prometheus) FloatGetter() func() (float64, error) {
	return func() (float64, error) {
		res, err := p.Query()
		if err != nil {
			return 0, err
		}

		if res.Type() != model.ValScalar {
			return 0, fmt.Errorf("query returned value of type %q, expected %q, consider wrapping query in scalar()", res.Type().String(), model.ValScalar.String())
		}

		scalarVal := res.(*model.Scalar)
		return float64(scalarVal.Value), nil
	}
}

var _ IntProvider = (*Prometheus)(nil)

// IntGetter expects scalar value from query response as int
func (p *Prometheus) IntGetter() func() (int64, error) {
	floatGetter := p.FloatGetter()
	return func() (int64, error) {
		float, err := floatGetter()
		return int64(math.Round(float)), err
	}
}
