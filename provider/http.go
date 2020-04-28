package provider

import (
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/andig/evcc/util"
)

// HTTP implements shell script-based providers and setters
type HTTP struct {
	log *util.Logger
	*util.HTTPHelper
	url, method string
	headers     map[string]string
	scale       float64
}

// NewHTTPProviderFromConfig creates a script provider
func NewHTTPProviderFromConfig(log *util.Logger, other map[string]interface{}) *HTTP {
	cc := struct {
		URL, Method string
		Headers     map[string]string
		Scale       float64
	}{}
	util.DecodeOther(log, other, &cc)

	logger := util.NewLogger("http")

	p := &HTTP{
		log:        logger,
		HTTPHelper: util.NewHTTPHelper(logger),
		url:        cc.URL,
		method:     cc.Method,
		headers:    cc.Headers,
		scale:      cc.Scale,
	}

	return p
}

// request executed the configured request
func (p *HTTP) request() ([]byte, error) {
	req, err := http.NewRequest(strings.ToUpper(p.method), p.url, nil)
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
	b, err := p.request()
	if err == nil {
		return strconv.ParseFloat(string(b), 64)
	}

	return 0, err
}

// IntGetter parses int64 from request
func (p *HTTP) IntGetter() (int64, error) {
	f, err := p.FloatGetter()
	return int64(math.Round(f)), err
}

// StringGetter returns string from request
func (p *HTTP) StringGetter() (string, error) {
	b, err := p.request()
	return string(b), err
}

// BoolGetter parses bool from request
func (p *HTTP) BoolGetter() (bool, error) {
	s, err := p.StringGetter()
	return util.Truish(s), err
}
