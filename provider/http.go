package provider

import (
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
	log *util.Logger
	*util.HTTPHelper
	url, method string
	headers     map[string]string
	body        string
	scale       float64
	jq          *gojq.Query
}

// NewHTTPProviderFromConfig creates a HTTP provider
func NewHTTPProviderFromConfig(log *util.Logger, other map[string]interface{}) *HTTP {
	cc := struct {
		URI, Method string
		Headers     map[string]string
		Body        string
		Jq          string
		Scale       float64
	}{}
	util.DecodeOther(log, other, &cc)

	logger := util.NewLogger("http")

	p := &HTTP{
		log:        logger,
		HTTPHelper: util.NewHTTPHelper(logger),
		url:        cc.URI,
		method:     cc.Method,
		headers:    cc.Headers,
		body:       cc.Body,
		scale:      cc.Scale,
	}

	if cc.Jq != "" {
		op, err := gojq.Parse(cc.Jq)
		if err != nil {
			log.FATAL.Fatalf("config: invalid jq query: %s", p.jq)
		}

		p.jq = op
	}

	return p
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
	if err == nil && p.scale > 0 {
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
