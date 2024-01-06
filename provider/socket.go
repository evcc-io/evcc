package provider

import (
	"context"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/evcc-io/evcc/api"
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
	url      string
	headers  map[string]string
	scale    float64
	pipeline *pipeline.Pipeline
	val      *util.Monitor[[]byte]
}

func init() {
	registry.Add("ws", NewSocketProviderFromConfig)
	registry.Add("websocket", NewSocketProviderFromConfig)
}

// NewSocketProviderFromConfig creates a HTTP provider
func NewSocketProviderFromConfig(other map[string]interface{}) (Provider, error) {
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
		url:     url,
		headers: cc.Headers,
		scale:   cc.Scale,
		val:     util.NewMonitor[[]byte](cc.Timeout),
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
	if p.pipeline, err = pipeline.New(log, cc.Settings); err != nil {
		return nil, err
	}

	go p.listen()

	if cc.Timeout > 0 {
		select {
		case <-p.val.Done():
		case <-time.After(cc.Timeout):
			return nil, api.ErrTimeout
		}
	}

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
				p.val.Set(v)
			}
		}
	}
}

var _ StringProvider = (*Socket)(nil)

// StringGetter sends string request
func (p *Socket) StringGetter() (func() (string, error), error) {
	return func() (string, error) {
		val, err := p.val.Get()
		return string(val), err
	}, nil
}

var _ FloatProvider = (*Socket)(nil)

// FloatGetter parses float from string getter
func (p *Socket) FloatGetter() (func() (float64, error), error) {
	g, err := p.StringGetter()

	return func() (float64, error) {
		s, err := g()
		if err != nil {
			return 0, err
		}

		f, err := strconv.ParseFloat(s, 64)

		return f * p.scale, err
	}, err
}

var _ IntProvider = (*Socket)(nil)

// IntGetter parses int64 from float getter
func (p *Socket) IntGetter() (func() (int64, error), error) {
	g, err := p.FloatGetter()

	return func() (int64, error) {
		f, err := g()
		return int64(math.Round(f)), err
	}, err
}

var _ BoolProvider = (*Socket)(nil)

// BoolGetter parses bool from string getter
func (p *Socket) BoolGetter() (func() (bool, error), error) {
	g, err := p.StringGetter()

	return func() (bool, error) {
		s, err := g()
		return util.Truish(s), err
	}, err
}
