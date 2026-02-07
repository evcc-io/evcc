package plugin

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/plugin/pipeline"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
)

const retryDelay = 5 * time.Second

// Socket implements websocket request provider
type Socket struct {
	*getter
	*request.Helper
	log      *util.Logger
	url      string
	headers  map[string]string
	pipeline *pipeline.Pipeline
	val      *util.Monitor[[]byte]
}

func init() {
	registry.Add("ws", NewSocketPluginFromConfig)
	registry.Add("websocket", NewSocketPluginFromConfig)
}

// NewSocketPluginFromConfig creates a HTTP provider
func NewSocketPluginFromConfig(other map[string]any) (Plugin, error) {
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
		val:     util.NewMonitor[[]byte](cc.Timeout),
	}

	p.getter = defaultGetters(p, cc.Scale)

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

	errC := make(chan error, 1)
	go p.run(errC)

	if cc.Timeout > 0 {
		select {
		case <-p.val.Done():
		case <-time.After(cc.Timeout):
			return nil, api.ErrTimeout
		case err := <-errC:
			return nil, err
		}
	}

	return p, nil
}

func (p *Socket) run(errC chan error) {
	var once sync.Once

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
			// handle initial connection error immediately
			once.Do(func() { errC <- err })

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

var _ Getters = (*Socket)(nil)

// StringGetter sends string request
func (p *Socket) StringGetter() (func() (string, error), error) {
	return func() (string, error) {
		val, err := p.val.Get()
		if err != nil {
			return "", err
		}

		if err := knownErrors(val); err != nil {
			return "", err
		}

		return string(val), nil
	}, nil
}
