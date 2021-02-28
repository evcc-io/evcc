package provider

import (
	"encoding/base64"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"

	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/jq"
	"github.com/andig/evcc/util/request"
	"github.com/itchyny/gojq"
)

// HTTP implements HTTP request provider
type HTTP struct {
	*request.Helper
	url, method string
	headers     map[string]string
	body        string
	scale       float64
	jq          *gojq.Query
}

func init() {
	registry.Add("http", NewHTTPProviderFromConfig)
}

// Auth is the authorization config
type Auth struct {
	Type, User, Password string
}

// AuthHeaders creates authorization headers from config
func AuthHeaders(log *util.Logger, auth Auth, headers map[string]string) error {
	if strings.ToLower(auth.Type) != "basic" {
		return fmt.Errorf("unsupported auth type: %s", auth.Type)
	}

	basicAuth := auth.User + ":" + auth.Password
	headers["Authorization"] = "Basic " + base64.StdEncoding.EncodeToString([]byte(basicAuth))
	return nil
}

// NewHTTPProviderFromConfig creates a HTTP provider
func NewHTTPProviderFromConfig(other map[string]interface{}) (IntProvider, error) {
	cc := struct {
		URI, Method string
		Headers     map[string]string
		Body        string
		Jq          string
		Scale       float64
		Insecure    bool
		Auth        Auth
	}{
		Headers: make(map[string]string),
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	log := util.NewLogger("http")

	// handle basic auth
	if cc.Auth.Type != "" {
		if err := AuthHeaders(log, cc.Auth, cc.Headers); err != nil {
			return nil, fmt.Errorf("http auth: %w", err)
		}
	}

	return NewHTTP(log,
		cc.Method,
		cc.URI,
		cc.Headers,
		cc.Body,
		cc.Insecure,
		cc.Jq,
		cc.Scale,
	)
}

// NewHTTP create HTTP provider
func NewHTTP(log *util.Logger, method, uri string, headers map[string]string, body string, insecure bool, jq string, scale float64) (*HTTP, error) {
	url := util.DefaultScheme(uri, "http")
	if url != uri {
		log.WARN.Printf("missing scheme for %s, assuming http", uri)
	}

	p := &HTTP{
		Helper:  request.NewHelper(log),
		url:     url,
		method:  method,
		headers: headers,
		body:    body,
		scale:   scale,
	}

	// ignore the self signed certificate
	if insecure {
		p.Client.Transport = request.NewTripper(log, request.InsecureTransport())
	}

	if jq != "" {
		op, err := gojq.Parse(jq)
		if err != nil {
			return nil, fmt.Errorf("invalid jq query '%s': %w", p.jq, err)
		}

		p.jq = op
	}

	return p, nil
}

// request executed the configured request
func (p *HTTP) request(body ...string) ([]byte, error) {
	var b io.Reader
	if len(body) == 1 {
		b = strings.NewReader(body[0])
	}

	// empty method becomes GET
	req, err := request.New(strings.ToUpper(p.method), p.url, b, p.headers)
	if err != nil {
		return []byte{}, err
	}

	return p.DoBody(req)
}

// FloatGetter parses float from request
func (p *HTTP) FloatGetter() func() (float64, error) {
	g := p.StringGetter()

	return func() (float64, error) {
		s, err := g()
		if err != nil {
			return 0, err
		}

		f, err := strconv.ParseFloat(s, 64)
		if err == nil && p.scale != 0 {
			f *= p.scale
		}

		return f, err
	}
}

// IntGetter parses int64 from request
func (p *HTTP) IntGetter() func() (int64, error) {
	g := p.FloatGetter()

	return func() (int64, error) {
		f, err := g()
		return int64(math.Round(f)), err
	}
}

// StringGetter sends string request
func (p *HTTP) StringGetter() func() (string, error) {
	return func() (string, error) {
		b, err := p.request()
		if err != nil {
			return string(b), err
		}

		if p.jq != nil {
			v, err := jq.Query(p.jq, b)
			return fmt.Sprintf("%v", v), err
		}

		return string(b), err
	}
}

// BoolGetter parses bool from request
func (p *HTTP) BoolGetter() func() (bool, error) {
	g := p.StringGetter()

	return func() (bool, error) {
		s, err := g()
		return util.Truish(s), err
	}
}

func (p *HTTP) set(param string, val interface{}) error {
	body, err := setFormattedValue(p.body, param, val)

	if err == nil {
		_, err = p.request(body)
	}

	return err
}

// IntSetter sends int request
func (p *HTTP) IntSetter(param string) func(int64) error {
	return func(val int64) error {
		return p.set(param, val)
	}
}

// StringSetter sends string request
func (p *HTTP) StringSetter(param string) func(string) error {
	return func(val string) error {
		return p.set(param, val)
	}
}

// BoolSetter sends bool request
func (p *HTTP) BoolSetter(param string) func(bool) error {
	return func(val bool) error {
		return p.set(param, val)
	}
}
