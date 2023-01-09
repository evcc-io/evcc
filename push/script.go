package push

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/jq"
	"github.com/itchyny/gojq"
	"github.com/kballard/go-shellquote"
)

// Script implements shell script-based message service and setters
type Script struct {
	log     *util.Logger
	script  string
	timeout time.Duration
	cache   time.Duration
	updated time.Time
	val     string
	err     error
	re      *regexp.Regexp
	jq      *gojq.Query
	scale   float64
}

type scriptConfig struct {
	CmdLine string
	Timeout time.Duration
	Scale   float64
	Cache   time.Duration
}

// NewScriptMessenger creates a Script messenger. Script execution is aborted after given timeout.
func NewScriptMessenger(script string, timeout time.Duration, scale float64, cache time.Duration) (*Script, error) {
	s := &Script{
		log:     util.NewLogger("script"),
		script:  script,
		timeout: timeout,
		scale:   scale,
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

// Send calls the script
func (m *Script) Send(title, msg string) {
	_, err := m.exec(m.script + " '" + title + "' '" + msg + "'")
	if err != nil {
		m.log.ERROR.Printf("Script message error: %v", err)
	}
}

func (m *Script) exec(script string) (string, error) {
	args, err := shellquote.Split(script)
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
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

		m.log.ERROR.Printf("%s: %s", strings.Join(args, " "), s)
		return "", err
	}

	m.log.DEBUG.Printf("%s: %s", strings.Join(args, " "), s)

	return s, nil
}

// StringGetter returns string from exec result. Only STDOUT is considered.
func (m *Script) StringGetter() func() (string, error) {
	return func() (string, error) {
		if time.Since(m.updated) > m.cache {
			m.val, m.err = m.exec(m.script)
			m.updated = time.Now()

			if m.err == nil && m.re != nil {
				ma := m.re.FindStringSubmatch(m.val)
				if len(ma) > 1 {
					m.val = ma[1] // first submatch
				}
			}

			if m.err == nil && m.jq != nil {
				var v interface{}
				if v, m.err = jq.Query(m.jq, []byte(m.val)); m.err == nil {
					m.val = fmt.Sprintf("%v", v)
				}
			}
		}

		return m.val, m.err
	}
}

// FloatGetter parses float from exec result
func (m *Script) FloatGetter() func() (float64, error) {
	g := m.StringGetter()

	return func() (float64, error) {
		s, err := g()
		if err != nil {
			return 0, err
		}

		f, err := strconv.ParseFloat(s, 64)
		if err == nil {
			f *= m.scale
		}

		return f, err
	}
}

// IntGetter parses int64 from exec result
func (m *Script) IntGetter() func() (int64, error) {
	g := m.FloatGetter()

	return func() (int64, error) {
		f, err := g()
		return int64(math.Round(f)), err
	}
}

// BoolGetter parses bool from exec result. "on", "true" and 1 are considered truish.
func (m *Script) BoolGetter() func() (bool, error) {
	g := m.StringGetter()

	return func() (bool, error) {
		s, err := g()
		if err != nil {
			return false, err
		}

		return util.Truish(s), nil
	}
}

// IntSetter invokes script with parameter replaced by int value
func (m *Script) IntSetter(param string) func(int64) error {
	// return func to access cached value
	return func(i int64) error {
		cmd, err := util.ReplaceFormatted(m.script, map[string]interface{}{
			param: i,
		})

		if err == nil {
			_, err = m.exec(cmd)
		}

		return err
	}
}

// BoolSetter invokes script with parameter replaced by bool value
func (m *Script) BoolSetter(param string) func(bool) error {
	// return func to access cached value
	return func(b bool) error {
		cmd, err := util.ReplaceFormatted(m.script, map[string]interface{}{
			param: b,
		})

		if err == nil {
			_, err = m.exec(cmd)
		}

		return err
	}
}
