package plugin

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
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
	log         *util.Logger
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
		log:    log,
	}

	// build the cache stack without logging so the logging tripper
	// can sit outside the cache and see cached responses too
	var base http.RoundTripper = transport.Default()
	if insecure {
		base = transport.Insecure()
	}

	if cache > 0 {
		// remove cache-busting response headers
		base = &transport.Modifier{
			Modifier: func(resp *http.Response) error {
				dropCacheBusting(resp, "Cache-Control")
				dropCacheBusting(resp, "Pragma")
				// httpcache derives freshness from the response Date; stamp one
				// for devices that omit it, else every read is treated as stale
				if resp.Header.Get("Date") == "" {
					resp.Header.Set("Date", time.Now().UTC().Format(http.TimeFormat))
				}
				return nil
			},
			Base: base,
		}
	}

	// http cache
	base = &httpcache.Transport{
		Cache:               mc,
		MarkCachedResponses: true,
		Transport:           base,
	}

	if cache > 0 {
		cacheHeader := fmt.Sprintf("max-age=%d, must-revalidate", int(cache.Seconds()))
		base = &transport.Decorator{
			Decorator: transport.DecorateHeaders(map[string]string{
				"Cache-Control": cacheHeader,
			}),
			Base: base,
		}

		// for cached requests enforce single inflight GET
		if method == http.MethodGet {
			p.mu = muForKey(p.url)
		}
	}

	// logging is outermost so cache hits are visible in the trace log
	p.Client.Transport = request.NewTripper(log, base)

	return p
}

// dropCacheBusting removes response directives that defeat the cache layer
// (no-cache, no-store and max-age=0) so a configured cache duration takes effect.
func dropCacheBusting(resp *http.Response, header string) {
	h := resp.Header.Get(header)
	if h == "" {
		return
	}

	var hh []string

	for token := range strings.SplitSeq(h, ",") {
		s := strings.TrimSpace(token)

		name, value, _ := strings.Cut(s, "=")
		switch strings.ToLower(strings.TrimSpace(name)) {
		case "no-cache", "no-store":
			continue
		case "max-age":
			if v, err := strconv.Atoi(strings.TrimSpace(value)); err == nil && v <= 0 {
				continue
			}
		}

		hh = append(hh, s)
	}

	if len(hh) == 0 {
		resp.Header.Del(header)
	} else {
		resp.Header.Set(header, strings.Join(hh, ", "))
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

	resp, err := p.Do(req)
	if err != nil {
		return nil, err
	}

	// warn on uncached GET polling: a repeated roundtrip means neither a configured
	// cache nor the device's own response headers spared it. cache hits are exempt.
	if p.method == http.MethodGet && p.mu == nil && resp.Header.Get(httpcache.XFromCache) == "" {
		if key := stripQuery(url); repeatedGet(key, time.Now()) {
			p.log.WARN.Printf("uncached request repeated within 1s, please report at https://github.com/evcc-io/evcc/issues: %s", key)
		}
	}

	val, err := request.ReadBody(resp)
	if err != nil {
		if err2 := knownErrors(val); err2 != nil {
			err = err2
		}
	}

	return val, err
}

type httpAccess struct {
	last   time.Time
	warned bool
}

var (
	httpSeenMu sync.Mutex
	httpSeen   = make(map[string]httpAccess)
)

// stripQuery drops the query and fragment so cache-busting params do not make
// each poll look like a distinct url.
func stripQuery(url string) string {
	if i := strings.IndexAny(url, "?#"); i >= 0 {
		return url[:i]
	}
	return url
}

// repeatedGet reports the first time url is fetched again within a second, a sign
// the response should be cached. It fires once per url to avoid log spam.
func repeatedGet(url string, now time.Time) bool {
	httpSeenMu.Lock()
	defer httpSeenMu.Unlock()

	a, seen := httpSeen[url]
	warn := seen && !a.warned && now.Sub(a.last) < time.Second
	httpSeen[url] = httpAccess{last: now, warned: a.warned || warn}
	return warn
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
