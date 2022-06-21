package provider

import (
	"fmt"
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/jq"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"github.com/gorilla/websocket"
	"github.com/itchyny/gojq"
)

const retryDelay = 5 * time.Second

// Socket implements websocket request provider
type Socket struct {
	*request.Helper
	log     *util.Logger
	mux     sync.Mutex
	wait    *util.Waiter
	url     string
	headers map[string]string
	scale   float64
	jq      *gojq.Query
	val     interface{}
}

func init() {
	registry.Add("ws", NewSocketProviderFromConfig)
	registry.Add("websocket", NewSocketProviderFromConfig)
}

// NewSocketProviderFromConfig creates a HTTP provider
func NewSocketProviderFromConfig(other map[string]interface{}) (IntProvider, error) {
	cc := struct {
		URI      string
		Headers  map[string]string
		Jq       string
		Scale    float64
		Insecure bool
		Auth     Auth
		Timeout  time.Duration
	}{
		Headers: make(map[string]string),
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	log := util.NewLogger("ws")

	url := util.DefaultScheme(cc.URI, "ws")
	if url != cc.URI {
		log.WARN.Printf("missing scheme for %s, assuming ws", cc.URI)
	}

	p := &Socket{
		log:     log,
		Helper:  request.NewHelper(log),
		wait:    util.NewWaiter(cc.Timeout, func() { log.DEBUG.Println("wait for initial value") }),
		url:     url,
		headers: cc.Headers,
		scale:   cc.Scale,
	}

	// handle basic auth
	if cc.Auth.Type != "" {
		basicAuth := transport.BasicAuthHeader(cc.Auth.User, cc.Auth.Password)
		log.Redact(basicAuth)

		p.headers["Authorization"] = basicAuth
	}

	// ignore the self signed certificate
	if cc.Insecure {
		p.Client.Transport = request.NewTripper(log, transport.Insecure())
	}

	if cc.Jq != "" {
		op, err := gojq.Parse(cc.Jq)
		if err != nil {
			return nil, fmt.Errorf("invalid jq query: %s", p.jq)
		}

		p.jq = op
	}

	go p.listen()

	return p, nil
}

func (p *Socket) listen() {
	headers := make(http.Header)
	for k, v := range p.headers {
		headers.Set(k, v)
	}

	dialer := &websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: request.Timeout,
	}

	for {
		client, _, err := dialer.Dial(p.url, headers)
		if err != nil {
			p.log.ERROR.Println(err)
			time.Sleep(retryDelay)
			continue
		}

		for {
			_, b, err := client.ReadMessage()
			if err != nil {
				p.log.TRACE.Println("read:", err)
				_ = client.Close()
				break
			}

			p.log.TRACE.Printf("recv: %s", b)

			p.mux.Lock()
			if p.jq != nil {
				v, err := jq.Query(p.jq, b)
				if err == nil {
					p.val = v
					p.wait.Update()
				}
			} else {
				p.val = string(b)
				p.wait.Update()
			}
			p.mux.Unlock()
		}
	}
}

func (p *Socket) hasValue() (interface{}, error) {
	if late := p.wait.Overdue(); late > 0 {
		return nil, fmt.Errorf("outdated: %v", late.Truncate(time.Second))
	}

	p.mux.Lock()
	defer p.mux.Unlock()

	return p.val, nil
}

// StringGetter sends string request
func (p *Socket) StringGetter() func() (string, error) {
	return func() (string, error) {
		v, err := p.hasValue()
		if err != nil {
			return "", err
		}

		return jq.String(v)
	}
}

// FloatGetter parses float from string getter
func (p *Socket) FloatGetter() func() (float64, error) {
	return func() (float64, error) {
		v, err := p.hasValue()
		if err != nil {
			return 0, err
		}

		// v is always string when jq not used
		if p.jq == nil {
			v, err = strconv.ParseFloat(v.(string), 64)
			if err != nil {
				return 0, err
			}
		}

		f, err := jq.Float64(v)
		return f * p.scale, err
	}
}

// IntGetter parses int64 from float getter
func (p *Socket) IntGetter() func() (int64, error) {
	return func() (int64, error) {
		v, err := p.hasValue()
		if err != nil {
			return 0, err
		}

		// v is always string when jq not used
		if p.jq == nil {
			v, err = strconv.ParseInt(v.(string), 10, 64)
			if err != nil {
				return 0, err
			}
		}

		i, err := jq.Int64(v)
		f := float64(i) * p.scale

		return int64(math.Round(f)), err
	}
}

// BoolGetter parses bool from string getter
func (p *Socket) BoolGetter() func() (bool, error) {
	return func() (bool, error) {
		v, err := p.hasValue()
		if err != nil {
			return false, err
		}

		// v is always string when jq not used
		if p.jq == nil {
			v = util.Truish(v.(string))
		}

		return jq.Bool(v)
	}
}
