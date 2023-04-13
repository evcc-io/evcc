package provider

import (
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/evcc-io/evcc/provider/pipeline"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"github.com/jpfielding/go-http-digest/pkg/digest"
)

// HTTP implements HTTP request provider
type HTTP struct {
	*request.Helper
	url, method string
	headers     map[string]string
	body        string
	scale       float64
	cache       time.Duration
	updated     time.Time
	pipeline    *pipeline.Pipeline
	val         []byte // Cached http response value
	err         error  // Cached http response error
}

func init() {
	registry.Add("http", NewHTTPProviderFromConfig)
}

// Auth is the authorization config
type Auth struct {
	Type, User, Password string
}

// NewHTTPProviderFromConfig creates a HTTP provider
func NewHTTPProviderFromConfig(other map[string]interface{}) (IntProvider, error) {
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
		Scale:   1,
		Timeout: request.Timeout,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	http := NewHTTP(
		util.NewLogger("http"),
		cc.Method,
		cc.URI,
		cc.Insecure,
		cc.Scale,
		cc.Cache,
	).
		WithHeaders(cc.Headers).
		WithBody(cc.Body)

	http.Client.Timeout = cc.Timeout

	var err error
	if cc.Auth.Type != "" {
		_, err = http.WithAuth(cc.Auth.Type, cc.Auth.User, cc.Auth.Password)
	}

	if err == nil {
		var pipe *pipeline.Pipeline
		pipe, err = pipeline.New(cc.Settings)
		http = http.WithPipeline(pipe)
	}

	return http, err
}

// NewHTTP create HTTP provider
func NewHTTP(log *util.Logger, method, uri string, insecure bool, scale float64, cache time.Duration) *HTTP {
	url := util.DefaultScheme(uri, "http")
	if url != uri {
		log.WARN.Printf("missing scheme for %s, assuming http", uri)
	}

	p := &HTTP{
		Helper: request.NewHelper(log),
		url:    url,
		method: method,
		scale:  scale,
		cache:  cache,
	}

	// ignore the self signed certificate
	if insecure {
		p.Client.Transport = request.NewTripper(log, transport.Insecure())
	}

	return p
}

// WithBody adds request body
func (p *HTTP) WithBody(body string) *HTTP {
	p.body = body
	return p
}

// WithHeaders adds request headers
func (p *HTTP) WithHeaders(headers map[string]string) *HTTP {
	p.headers = headers
	return p
}

// WithPipeline adds a processing pipeline
func (p *HTTP) WithPipeline(pipeline *pipeline.Pipeline) *HTTP {
	p.pipeline = pipeline
	return p
}

// WithAuth adds authorized transport
func (p *HTTP) WithAuth(typ, user, password string) (*HTTP, error) {
	switch strings.ToLower(typ) {
	case "basic":
		basicAuth := transport.BasicAuthHeader(user, password)
		log.Redact(basicAuth)

		p.Client.Transport = transport.BasicAuth(user, password, p.Client.Transport)
	case "bearer":
		p.Client.Transport = transport.BearerAuth(password, p.Client.Transport)
	case "digest":
		p.Client.Transport = digest.NewTransport(user, password, p.Client.Transport)
	default:
		return nil, fmt.Errorf("unknown auth type '%s'", typ)
	}

	return p, nil
}

// request executes the configured request or returns the cached value
func (p *HTTP) request(url string, body ...string) ([]byte, error) {
	if time.Since(p.updated) >= p.cache {
		var b io.Reader
		if len(body) == 1 {
			b = strings.NewReader(body[0])
		}

		// empty method becomes GET
		req, err := request.New(strings.ToUpper(p.method), url, b, p.headers)
		if err != nil {
			return []byte{}, err
		}

		p.val, p.err = p.DoBody(req)
		p.updated = time.Now()
	}

	return p.val, p.err
}

var _ StringProvider = (*HTTP)(nil)

// StringGetter sends string request
func (p *HTTP) StringGetter() func() (string, error) {
	return func() (string, error) {
		b, err := p.request(p.url, p.body)

		if err == nil && p.pipeline != nil {
			b, err = p.pipeline.Process(b)
		}

		return string(b), err
	}
}

var _ FloatProvider = (*HTTP)(nil)

// FloatGetter parses float from request
func (p *HTTP) FloatGetter() func() (float64, error) {
	g := p.StringGetter()

	return func() (float64, error) {
		s, err := g()
		if err != nil {
			return 0, err
		}

		f, err := strconv.ParseFloat(s, 64)

		return f * p.scale, err
	}
}

var _ IntProvider = (*HTTP)(nil)

// IntGetter parses int64 from request
func (p *HTTP) IntGetter() func() (int64, error) {
	g := p.FloatGetter()

	return func() (int64, error) {
		f, err := g()
		return int64(math.Round(f)), err
	}
}

var _ BoolProvider = (*HTTP)(nil)

// BoolGetter parses bool from request
func (p *HTTP) BoolGetter() func() (bool, error) {
	g := p.StringGetter()

	return func() (bool, error) {
		s, err := g()
		return util.Truish(s), err
	}
}

func (p *HTTP) set(param string, val interface{}) error {
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

var _ SetIntProvider = (*HTTP)(nil)

// IntSetter sends int request
func (p *HTTP) IntSetter(param string) func(int64) error {
	return func(val int64) error {
		return p.set(param, val)
	}
}

var _ SetStringProvider = (*HTTP)(nil)

// StringSetter sends string request
func (p *HTTP) StringSetter(param string) func(string) error {
	return func(val string) error {
		return p.set(param, val)
	}
}

var _ SetBoolProvider = (*HTTP)(nil)

// BoolSetter sends bool request
func (p *HTTP) BoolSetter(param string) func(bool) error {
	return func(val bool) error {
		return p.set(param, val)
	}
}
