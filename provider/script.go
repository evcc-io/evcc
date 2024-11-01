package provider

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/jq"
	"github.com/evcc-io/evcc/util/request"
	"github.com/itchyny/gojq"
	"github.com/kballard/go-shellquote"
)

// Script implements shell script-based providers and setters
type Script struct {
	*getter
	log     *util.Logger
	script  string
	timeout time.Duration
	cache   time.Duration
	updated time.Time
	val     string
	err     error
	re      *regexp.Regexp
	jq      *gojq.Query
}

func init() {
	registry.Add("script", NewScriptProviderFromConfig)
}

// NewScriptProviderFromConfig creates a script provider.
func NewScriptProviderFromConfig(other map[string]interface{}) (Provider, error) {
	cc := struct {
		Cmd     string
		Timeout time.Duration
		Cache   time.Duration
		Regex   string
		Jq      string
		Scale   float64
	}{
		Timeout: request.Timeout,
		Scale:   1,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	p, err := NewScriptProvider(cc.Cmd, cc.Timeout, cc.Scale, cc.Cache)
	p.getter = defaultGetters(p, cc.Scale)

	if err == nil && cc.Regex != "" {
		_, err = p.WithRegex(cc.Regex)
	}

	if err == nil && cc.Jq != "" {
		_, err = p.WithJq(cc.Jq)
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

func (p *Script) WithRegex(regex string) (*Script, error) {
	re, err := regexp.Compile(regex)
	if err != nil {
		return nil, fmt.Errorf("invalid regex '%s': %w", re, err)
	}

	p.re = re

	return p, nil
}

func (p *Script) WithJq(jq string) (*Script, error) {
	op, err := gojq.Parse(jq)
	if err != nil {
		return nil, fmt.Errorf("invalid jq query '%s': %w", jq, err)
	}

	p.jq = op

	return p, nil
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

var _ StringProvider = (*Script)(nil)

// StringGetter returns string from exec result. Only STDOUT is considered.
func (p *Script) StringGetter() (func() (string, error), error) {
	return func() (string, error) {
		if time.Since(p.updated) > p.cache {
			p.val, p.err = p.exec(p.script)
			p.updated = time.Now()

			if p.err == nil && p.re != nil {
				m := p.re.FindStringSubmatch(p.val)
				if len(m) > 1 {
					p.val = m[1] // first submatch
				}
			}

			if p.err == nil && p.jq != nil {
				var v interface{}
				if v, p.err = jq.Query(p.jq, []byte(p.val)); p.err == nil {
					p.val = fmt.Sprintf("%v", v)
				}
			}
		}

		return p.val, p.err
	}, nil
}

var _ SetIntProvider = (*Script)(nil)

// IntSetter invokes script with parameter replaced by int value
func (p *Script) IntSetter(param string) (func(int64) error, error) {
	// return func to access cached value
	return func(i int64) error {
		cmd, err := util.ReplaceFormatted(p.script, map[string]interface{}{
			param: i,
		})

		if err == nil {
			_, err = p.exec(cmd)
		}

		return err
	}, nil
}

var _ SetBoolProvider = (*Script)(nil)

// BoolSetter invokes script with parameter replaced by bool value
func (p *Script) BoolSetter(param string) (func(bool) error, error) {
	// return func to access cached value
	return func(b bool) error {
		cmd, err := util.ReplaceFormatted(p.script, map[string]interface{}{
			param: b,
		})

		if err == nil {
			_, err = p.exec(cmd)
		}

		return err
	}, nil
}
