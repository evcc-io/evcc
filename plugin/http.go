package plugin

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/evcc-io/evcc/plugin/pipeline"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"github.com/sandrolain/httpcache"
)

// HTTP implements HTTP request provider
type HTTP struct {
	*getter
	*request.Helper
	url, method string
	headers     map[string]string
	body        string
	pipeline    *pipeline.Pipeline
	mu          *sync.Mutex
}

func init() {
	registry.AddCtx("http", NewHTTPPluginFromConfig)
}

var mc = httpcache.NewMemoryCache()

// NewHTTPPluginFromConfig creates a HTTP provider
func NewHTTPPluginFromConfig(ctx context.Context, other map[string]any) (Plugin, error) {
	cc := struct {
		URI, Method       string
		Headers           map[string]string
		Body              string
		pipeline.Settings `mapstructure:",squash"`
		Scale             float64
		Insecure          bool
		Auth              Auth
		Timeout           time.Duration
		Cache             time.Duration
	}{
		Headers: make(map[string]string),
		Method:  http.MethodGet,
		Scale:   1,
		Timeout: request.Timeout,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.URI == "" {
		return nil, errors.New("missing uri")
	}

	log := util.ContextLoggerWithDefault(ctx, util.NewLogger("http"))
	p := NewHTTP(
		log,
		strings.ToUpper(cc.Method),
		cc.URI,
		cc.Insecure,
		cc.Cache,
	).
		WithHeaders(cc.Headers).
		WithBody(cc.Body)

	p.Client.Timeout = cc.Timeout

	p.getter = defaultGetters(p, cc.Scale)

	if cc.Auth.Type != "" || cc.Auth.Source != "" {
		transport, err := cc.Auth.Transport(ctx, log, p.Client.Transport)
		if err != nil {
			return nil, err
		}
		p.Client.Transport = transport
	}

	pipe, err := pipeline.New(log, cc.Settings)
	if err != nil {
		return nil, err
	}
	p.pipeline = pipe

	return p, nil
}

// NewHTTP create HTTP provider
func NewHTTP(log *util.Logger, method, uri string, insecure bool, cache time.Duration) *HTTP {
	p := &HTTP{
		Helper: request.NewHelper(log),
		url:    uri,
		method: method,
	}

	// override the transport to accept self-signed certificates
	if insecure {
		p.Client.Transport = request.NewTripper(log, transport.Insecure())
	}

	if cache > 0 {
		// remove no-cache response headers
		p.Client.Transport = &transport.Modifier{
			Modifier: func(resp *http.Response) error {
				dropNoCache(resp, "Cache-Control")
				dropNoCache(resp, "Pragma")
				return nil
			},
			Base: p.Client.Transport,
		}
	}

	// http cache
	p.Client.Transport = &httpcache.Transport{
		Cache:               mc,
		MarkCachedResponses: true,
		Transport:           p.Client.Transport,
	}

	if cache > 0 {
		cacheHeader := fmt.Sprintf("max-age=%d, must-revalidate", int(cache.Seconds()))
		p.Client.Transport = &transport.Decorator{
			Decorator: transport.DecorateHeaders(map[string]string{
				"Cache-Control": cacheHeader,
			}),
			Base: p.Client.Transport,
		}

		// for cached requests enforce single inflight GET
		if method == http.MethodGet {
			p.mu = muForKey(p.url)
		}
	}

	return p
}

func dropNoCache(resp *http.Response, header string) {
	if h := resp.Header.Get(header); h != "" {
		var hh []string

		for h := range strings.SplitSeq(h, ",") {
			if s := strings.TrimSpace(h); strings.ToLower(s) != "no-cache" {
				hh = append(hh, s)
			}
		}

		if len(hh) == 0 {
			resp.Header.Del(header)
		} else {
			resp.Header.Set(header, strings.Join(hh, ", "))
		}
	}
}

// WithBody adds request body
func (p *HTTP) WithBody(body string) *HTTP {
	if body != "" {
		p.body = body
		if p.method == http.MethodGet {
			p.method = http.MethodPost
		}
	}
	return p
}

// WithHeaders adds request headers
func (p *HTTP) WithHeaders(headers map[string]string) *HTTP {
	p.headers = headers
	return p
}

// request executes the configured request or returns the cached value
func (p *HTTP) request(url string, body string) ([]byte, error) {
	var b io.Reader
	if p.method != http.MethodGet {
		b = strings.NewReader(body)
	}

	url = util.DefaultScheme(url, "http")

	// empty method becomes GET
	req, err := request.New(p.method, url, b, p.headers)
	if err != nil {
		return []byte{}, err
	}

	val, err := p.DoBody(req)
	if err != nil {
		if err2 := knownErrors(val); err2 != nil {
			err = err2
		}
	}

	return val, err
}

var _ Getters = (*HTTP)(nil)

// StringGetter sends string request
func (p *HTTP) StringGetter() (func() (string, error), error) {
	return func() (string, error) {
		if p.mu != nil {
			p.mu.Lock()
			defer p.mu.Unlock()
		}

		url, err := setFormattedValue(p.url, "", "")
		if err != nil {
			return "", err
		}

		b, err := p.request(url, p.body)

		if err == nil && p.pipeline != nil {
			b, err = p.pipeline.Process(b)
		}

		return string(b), err
	}, nil
}

func (p *HTTP) set(param string, val any) error {
	url, err := setFormattedValue(p.url, param, val)
	if err != nil {
		return err
	}

	body, err := setFormattedValue(p.body, param, val)
	if err != nil {
		return err
	}

	_, err = p.request(url, body)

	return err
}

var _ IntSetter = (*HTTP)(nil)

// IntSetter sends int request
func (p *HTTP) IntSetter(param string) (func(int64) error, error) {
	return func(val int64) error {
		return p.set(param, val)
	}, nil
}

var _ FloatSetter = (*HTTP)(nil)

// FloatSetter sends int request
func (p *HTTP) FloatSetter(param string) (func(float64) error, error) {
	return func(val float64) error {
		return p.set(param, val)
	}, nil
}

var _ StringSetter = (*HTTP)(nil)

// StringSetter sends string request
func (p *HTTP) StringSetter(param string) (func(string) error, error) {
	return func(val string) error {
		return p.set(param, val)
	}, nil
}

var _ BoolSetter = (*HTTP)(nil)

// BoolSetter sends bool request
func (p *HTTP) BoolSetter(param string) (func(bool) error, error) {
	return func(val bool) error {
		return p.set(param, val)
	}, nil
}
