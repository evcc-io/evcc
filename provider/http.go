package provider

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/jq"
	"github.com/itchyny/gojq"
)

// HTTP implements HTTP request provider
type HTTP struct {
	*util.HTTPHelper
	url, method string
	headers     map[string]string
	body        string
	scale       float64
	jq          *gojq.Query
}

// Auth is the authorization config
type Auth struct {
	Type, User, Password string
}

// NewAuth creates authorization headers from config
func NewAuth(log *util.Logger, auth Auth, headers map[string]string) {
	if strings.ToLower(auth.Type) != "basic" {
		log.FATAL.Fatalf("config: unsupported auth type: %s", auth.Type)
	}

	basicAuth := auth.User + ":" + auth.Password
	headers["Authorization"] = "Basic " + base64.StdEncoding.EncodeToString([]byte(basicAuth))
}

// NewHTTPProviderFromConfig creates a HTTP provider
func NewHTTPProviderFromConfig(other map[string]interface{}) (*HTTP, error) {
	cc := struct {
		URI, Method string
		Headers     map[string]string
		Body        string
		Jq          string
		Scale       float64
		Insecure    bool
		Auth        Auth
	}{}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	log := util.NewLogger("http")

	p := &HTTP{
		HTTPHelper: util.NewHTTPHelper(log),
		url:        cc.URI,
		method:     cc.Method,
		headers:    cc.Headers,
		body:       cc.Body,
		scale:      cc.Scale,
	}

	// handle basic auth
	if cc.Auth.Type != "" {
		if p.headers == nil {
			p.headers = make(map[string]string)
		}
		NewAuth(log, cc.Auth, p.headers)
	}

	// ignore the self signed certificate
	if cc.Insecure {
		customTransport := http.DefaultTransport.(*http.Transport).Clone()
		customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		p.HTTPHelper.Client.Transport = customTransport
	}

	if cc.Jq != "" {
		op, err := gojq.Parse(cc.Jq)
		if err != nil {
			return nil, fmt.Errorf("config: invalid jq query: %s", p.jq)
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
	req, err := http.NewRequest(strings.ToUpper(p.method), p.url, b)
	if err == nil {
		for k, v := range p.headers {
			req.Header.Add(k, v)
		}
		return p.Request(req)
	}

	return []byte{}, err
}

// FloatGetter parses float from request
func (p *HTTP) FloatGetter() (float64, error) {
	s, err := p.StringGetter()
	if err != nil {
		return 0, err
	}

	f, err := strconv.ParseFloat(s, 64)
	if err == nil && p.scale != 0 {
		f *= p.scale
	}

	return f, err
}

// IntGetter parses int64 from request
func (p *HTTP) IntGetter() (int64, error) {
	f, err := p.FloatGetter()
	return int64(math.Round(f)), err
}

// StringGetter sends string request
func (p *HTTP) StringGetter() (string, error) {
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

// BoolGetter parses bool from request
func (p *HTTP) BoolGetter() (bool, error) {
	s, err := p.StringGetter()
	return util.Truish(s), err
}

// IntSetter sends int request
func (p *HTTP) IntSetter(param int64) error {
	body := util.FormatValue(p.body, param)
	_, err := p.request(body)
	return err
}

// StringSetter sends string request
func (p *HTTP) StringSetter(param string) error {
	body := util.FormatValue(p.body, param)
	_, err := p.request(body)
	return err
}

// BoolSetter sends bool request
func (p *HTTP) BoolSetter(param bool) error {
	body := util.FormatValue(p.body, param)
	_, err := p.request(body)
	return err
}
