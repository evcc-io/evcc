package provider

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"time"

	"github.com/evcc-io/evcc/provider/pipeline"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/kballard/go-shellquote"
)

// Script implements shell script-based providers and setters
type Script struct {
	*getter
	log      *util.Logger
	script   string
	timeout  time.Duration
	cache    time.Duration
	updated  time.Time
	val      string
	err      error
	pipeline *pipeline.Pipeline
}

func init() {
	registry.Add("script", NewScriptProviderFromConfig)
}

// NewScriptProviderFromConfig creates a script provider.
func NewScriptProviderFromConfig(other map[string]interface{}) (Provider, error) {
	cc := struct {
		Cmd               string
		pipeline.Settings `mapstructure:",squash"`
		Scale             float64
		Timeout           time.Duration
		Cache             time.Duration
	}{
		Timeout: request.Timeout,
		Scale:   1,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	p, err := NewScriptProvider(cc.Cmd, cc.Timeout, cc.Scale, cc.Cache)
	p.getter = defaultGetters(p, cc.Scale)

	if err == nil {
		var pipe *pipeline.Pipeline
		pipe, err = pipeline.New(log, cc.Settings)
		p.pipeline = pipe
	}

	return p, err
}

// NewScriptProvider creates a script provider.
// Script execution is aborted after given timeout.
func NewScriptProvider(script string, timeout time.Duration, scale float64, cache time.Duration) (*Script, error) {
	if strings.TrimSpace(script) == "" {
		return nil, errors.New("script is required")
	}

	s := &Script{
		log:     util.NewLogger("script"),
		script:  script,
		timeout: timeout,
		cache:   cache,
	}

	return s, nil
}

func (p *Script) exec(script string) (string, error) {
	args, err := shellquote.Split(script)
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), p.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	b, err := cmd.Output()

	s := strings.TrimSpace(string(b))

	if err != nil {
		// use STDOUT if available
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			s = strings.TrimSpace(string(ee.Stderr))
		}

		p.log.ERROR.Printf("%s: %s", strings.Join(args, " "), s)
		return "", err
	}

	p.log.DEBUG.Printf("%s: %s", strings.Join(args, " "), s)

	return s, nil
}

var _ Getters = (*Script)(nil)

// StringGetter returns string from exec result. Only STDOUT is considered.
func (p *Script) StringGetter() (func() (string, error), error) {
	return func() (string, error) {
		if time.Since(p.updated) > p.cache {
			p.val, p.err = p.exec(p.script)
			p.updated = time.Now()

			if p.err == nil && p.pipeline != nil {
				var b []byte
				b, p.err = p.pipeline.Process([]byte(p.val))
				p.val = string(b)
			}
		}

		return p.val, p.err
	}, nil
}

func scriptSetter[T any](p *Script, param string) (func(T) error, error) {
	return func(val T) error {
		cmd, err := util.ReplaceFormatted(p.script, map[string]interface{}{
			param: val,
		})

		if err == nil {
			_, err = p.exec(cmd)
		}

		return err
	}, nil
}

var _ SetIntProvider = (*Script)(nil)

// IntSetter invokes script with parameter replaced by int value
func (p *Script) IntSetter(param string) (func(int64) error, error) {
	return scriptSetter[int64](p, param)
}

var _ SetBoolProvider = (*Script)(nil)

// BoolSetter invokes script with parameter replaced by bool value
func (p *Script) BoolSetter(param string) (func(bool) error, error) {
	return scriptSetter[bool](p, param)
}

var _ SetStringProvider = (*Script)(nil)

// StringSetter returns a function that invokes a script with parameter by a string value
func (p *Script) StringSetter(param string) (func(string) error, error) {
	return scriptSetter[string](p, param)
}
