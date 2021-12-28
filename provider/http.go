package provider

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	xj "github.com/basgys/goxml2json"
	"github.com/evcc-io/evcc/provider/javascript"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/jq"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"github.com/itchyny/gojq"
	"github.com/jpfielding/go-http-digest/pkg/digest"
	"github.com/robertkrimen/otto"
	"github.com/volkszaehler/mbmd/meters/rs485"
)

// HTTP implements HTTP request provider
type HTTP struct {
	*request.Helper
	url, method string
	headers     map[string]string
	body        string
	re          *regexp.Regexp
	jq          *gojq.Query
	unpack      string
	decode      string
	vm          *otto.Otto
	script      string
	scale       float64
	cache       time.Duration
	updated     time.Time
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
		URI, Method string
		Headers     map[string]string
		Body        string
		Regex       string
		Jq          string
		Unpack      string
		Decode      string
		VM          string
		Script      string
		Scale       float64
		Insecure    bool
		Auth        Auth
		Timeout     time.Duration
		Cache       time.Duration
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
	).WithHeaders(cc.Headers).WithBody(cc.Body)
	http.Client.Timeout = cc.Timeout

	var err error
	if err == nil && cc.Regex != "" {
		_, err = http.WithRegex(cc.Regex)
	}

	if err == nil && cc.Jq != "" {
		_, err = http.WithJq(cc.Jq)
	}

	if err == nil && cc.Unpack != "" {
		_, err = http.WithUnpack(cc.Unpack)
	}

	if err == nil && cc.Decode != "" {
		_, err = http.WithDecode(cc.Decode)
	}

	if err == nil && cc.Script != "" {
		_, err = http.WithScript(cc.VM, cc.Script)
	}

	if err == nil && cc.Auth.Type != "" {
		_, err = http.WithAuth(cc.Auth.Type, cc.Auth.User, cc.Auth.Password)
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

// WithRegex adds a regex query applied to the mqtt listener payload
func (p *HTTP) WithRegex(regex string) (*HTTP, error) {
	re, err := regexp.Compile(regex)
	if err != nil {
		return nil, fmt.Errorf("invalid regex '%s': %w", re, err)
	}

	p.re = re

	return p, nil
}

// WithJq adds a jq query applied to the mqtt listener payload
func (p *HTTP) WithJq(jq string) (*HTTP, error) {
	op, err := gojq.Parse(jq)
	if err != nil {
		return nil, fmt.Errorf("invalid jq query '%s': %w", jq, err)
	}

	p.jq = op

	return p, nil
}

// WithUnpack adds data unpacking
func (p *HTTP) WithUnpack(unpack string) (*HTTP, error) {
	p.unpack = strings.ToLower(unpack)

	return p, nil
}

// WithDecode adds data decoding
func (p *HTTP) WithDecode(decode string) (*HTTP, error) {
	p.decode = strings.ToLower(decode)

	return p, nil
}

// WithScript adds a javascript script to process the response
func (p *HTTP) WithScript(vm, script string) (*HTTP, error) {
	regvm := javascript.RegisteredVM(strings.ToLower(vm))

	p.vm = regvm
	p.script = script

	return p, nil
}

// WithAuth adds authorized transport
func (p *HTTP) WithAuth(typ, user, password string) (*HTTP, error) {
	switch strings.ToLower(typ) {
	case "basic":
		basicAuth := transport.BasicAuthHeader(user, password)
		log.Redact(basicAuth)

		p.Client.Transport = transport.BasicAuth(user, password, p.Client.Transport)
	case "digest":
		p.Client.Transport = digest.NewTransport(user, password, p.Client.Transport)
	default:
		return nil, fmt.Errorf("unknown auth type '%s'", typ)
	}

	return p, nil
}

// request executes the configured request or returns the cached value
func (p *HTTP) request(body ...string) ([]byte, error) {
	if time.Since(p.updated) >= p.cache {
		var b io.Reader
		if len(body) == 1 {
			b = strings.NewReader(body[0])
		}

		// empty method becomes GET
		req, err := request.New(strings.ToUpper(p.method), p.url, b, p.headers)
		if err != nil {
			return []byte{}, err
		}

		p.val, p.err = p.DoBody(req)
		p.updated = time.Now()
	}

	return p.val, p.err
}

// transform XML into JSON with attribute names getting 'attr' prefix
func (p *HTTP) transformXML(value []byte) []byte {
	// only do a simple check, as some devices e.g. Kostal Piko MP plus don't seem to send proper XML
	if !bytes.HasPrefix(value, []byte("<")) {
		return value
	}

	xmlReader := bytes.NewReader(value)

	// Decode XML document
	root := new(xj.Node)
	if err := xj.NewDecoder(xmlReader).DecodeWithCustomPrefixes(root, "", "attr"); err != nil {
		return value
	}

	// Then encode it in JSON
	json := new(bytes.Buffer)
	if err := xj.NewEncoder(json).Encode(root); err != nil {
		return value
	}

	return json.Bytes()
}

func (p *HTTP) unpackValue(value []byte) (string, error) {
	switch p.unpack {
	case "hex":
		b, err := hex.DecodeString(string(value))
		if err != nil {
			return "", err
		}
		return string(b), nil
	}

	return "", fmt.Errorf("invalid unpack: %s", p.unpack)
}

// decode a hex string to a proper value
// TODO reuse similar code from Modbus
func (p *HTTP) decodeValue(value []byte) (interface{}, error) {
	switch p.decode {
	case "float32", "ieee754":
		return rs485.RTUIeee754ToFloat64(value), nil
	case "float32s", "ieee754s":
		return rs485.RTUIeee754ToFloat64Swapped(value), nil
	case "float64":
		return rs485.RTUUint64ToFloat64(value), nil
	case "uint16":
		return rs485.RTUUint16ToFloat64(value), nil
	case "uint32":
		return rs485.RTUUint32ToFloat64(value), nil
	case "uint32s":
		return rs485.RTUUint32ToFloat64Swapped(value), nil
	case "uint64":
		return rs485.RTUUint64ToFloat64(value), nil
	case "int16":
		return rs485.RTUInt16ToFloat64(value), nil
	case "int32":
		return rs485.RTUInt32ToFloat64(value), nil
	case "int32s":
		return rs485.RTUInt32ToFloat64Swapped(value), nil
	}

	return nil, fmt.Errorf("invalid decoding: %s", p.decode)
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
		if err == nil {
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
		b, err := p.request(p.body)
		if err != nil {
			return string(b), err
		}

		b = p.transformXML(b)

		if p.re != nil {
			m := p.re.FindSubmatch(b)
			if len(m) > 1 {
				b = m[1] // first submatch
			}
		}

		if p.jq != nil {
			v, err := jq.Query(p.jq, b)
			if err != nil {
				return string(b), err
			}
			b = []byte(fmt.Sprintf("%v", v))
		}

		if p.unpack != "" {
			v, err := p.unpackValue(b)
			if err != nil {
				return string(b), err
			}
			b = []byte(fmt.Sprintf("%v", v))
		}

		if p.decode != "" {
			v, err := p.decodeValue(b)
			if err != nil {
				return string(b), err
			}
			b = []byte(fmt.Sprintf("%v", v))
		}

		if p.vm != nil {
			err := p.vm.Set("val", string(b))
			if err != nil {
				return string(b), err
			}

			v, err := p.vm.Eval(p.script)
			if err != nil {
				return string(b), err
			}

			return v.ToString()
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
