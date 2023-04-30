package provider

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/evcc-io/evcc/provider/pipeline"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"nhooyr.io/websocket"
)

const retryDelay = 5 * time.Second

// Socket implements websocket request provider
type Socket struct {
	*request.Helper
	log      *util.Logger
	mux      sync.Mutex
	wait     *util.Waiter
	url      string
	headers  map[string]string
	scale    float64
	pipeline *pipeline.Pipeline
	val      []byte // Cached http response value
}

func init() {
	registry.Add("ws", NewSocketProviderFromConfig)
	registry.Add("websocket", NewSocketProviderFromConfig)
}

// NewSocketProviderFromConfig creates a HTTP provider
func NewSocketProviderFromConfig(other map[string]interface{}) (IntProvider, error) {
	cc := struct {
		URI               string
		Headers           map[string]string
		pipeline.Settings `mapstructure:",squash"`
		Scale             float64
		Insecure          bool
		Auth              Auth
		Timeout           time.Duration
	}{
		Headers: make(map[string]string),
		Scale:   1,
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

	var err error
	if p.pipeline, err = pipeline.New(cc.Settings); err != nil {
		return nil, err
	}

	go p.listen()

	return p, nil
}

func (p *Socket) listen() {
	headers := make(http.Header)
	for k, v := range p.headers {
		headers.Set(k, v)
	}

	opts := &websocket.DialOptions{
		HTTPHeader: headers,
	}

	for {
		ctx, cancel := context.WithTimeout(context.Background(), request.Timeout)
		conn, _, err := websocket.Dial(ctx, p.url, opts)
		cancel()

		if err != nil {
			p.log.ERROR.Println(err)
			time.Sleep(retryDelay)
			continue
		}

		for {
			_, b, err := conn.Read(context.Background())
			if err != nil {
				p.log.TRACE.Println("read:", err)
				_ = conn.Close(websocket.StatusAbnormalClosure, "done")
				break
			}

			p.log.TRACE.Printf("recv: %s", b)

			if v, err := p.pipeline.Process(b); err == nil {
				p.mux.Lock()
				p.val = v
				p.wait.Update()
				p.mux.Unlock()
			}
		}
	}
}

func (p *Socket) hasValue() ([]byte, error) {
	if late := p.wait.Overdue(); late > 0 {
		return nil, fmt.Errorf("outdated: %v", late.Truncate(time.Second))
	}

	p.mux.Lock()
	defer p.mux.Unlock()

	return p.val, nil
}

var _ StringProvider = (*Socket)(nil)

// StringGetter sends string request
func (p *Socket) StringGetter() func() (string, error) {
	return func() (string, error) {
		v, err := p.hasValue()
		if err != nil {
			return "", err
		}

		return string(v), err
	}
}

var _ FloatProvider = (*Socket)(nil)

// FloatGetter parses float from string getter
func (p *Socket) FloatGetter() func() (float64, error) {
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

var _ IntProvider = (*Socket)(nil)

// IntGetter parses int64 from float getter
func (p *Socket) IntGetter() func() (int64, error) {
	g := p.FloatGetter()

	return func() (int64, error) {
		f, err := g()
		return int64(math.Round(f)), err
	}
}

var _ BoolProvider = (*Socket)(nil)

// BoolGetter parses bool from string getter
func (p *Socket) BoolGetter() func() (bool, error) {
	g := p.StringGetter()

	return func() (bool, error) {
		s, err := g()
		return util.Truish(s), err
	}
}
