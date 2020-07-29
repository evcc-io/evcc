package provider

import (
	"context"
	"errors"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/andig/evcc/util"
	"github.com/kballard/go-shellquote"
)

// Script implements shell script-based providers and setters
type Script struct {
	log     *util.Logger
	timeout time.Duration
}

// NewScriptProvider creates a script provider.
// Script execution is aborted after given timeout.
func NewScriptProvider(timeout time.Duration) (*Script, error) {
	s := &Script{
		log:     util.NewLogger("exec"),
		timeout: timeout,
	}
	return s, nil
}

// StringGetter returns string from exec result. Only STDOUT is considered.
func (e *Script) StringGetter(script string) func() (string, error) {
	args, err := shellquote.Split(script)
	if err != nil {
		panic(err)
	} else if len(args) < 1 {
		panic("exec: missing script")
	}

	// return func to access cached value
	return func() (string, error) {
		ctx, cancel := context.WithTimeout(context.Background(), e.timeout)
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

			e.log.ERROR.Printf("%s: %s", strings.Join(args, " "), s)
			return "", err
		}

		e.log.TRACE.Printf("%s: %s", strings.Join(args, " "), s)
		return s, nil
	}
}

// IntGetter parses int64 from exec result
func (e *Script) IntGetter(script string) func() (int64, error) {
	exec := e.StringGetter(script)

	// return func to access cached value
	return func() (int64, error) {
		s, err := exec()
		if err != nil {
			return 0, err
		}

		return strconv.ParseInt(s, 10, 64)
	}
}

// FloatGetter parses float from exec result
func (e *Script) FloatGetter(script string) func() (float64, error) {
	exec := e.StringGetter(script)

	// return func to access cached value
	return func() (float64, error) {
		s, err := exec()
		if err != nil {
			return 0, err
		}

		return strconv.ParseFloat(s, 64)
	}
}

// BoolGetter parses bool from exec result. "on", "true" and 1 are considered truish.
func (e *Script) BoolGetter(script string) func() (bool, error) {
	exec := e.StringGetter(script)

	// return func to access cached value
	return func() (bool, error) {
		s, err := exec()
		if err != nil {
			return false, err
		}

		return util.Truish(s), nil
	}
}

// IntSetter invokes script with parameter replaced by int value
func (e *Script) IntSetter(param, script string) func(int64) error {
	// return func to access cached value
	return func(i int64) error {
		cmd, err := util.ReplaceFormatted(script, map[string]interface{}{
			param: i,
		})
		if err != nil {
			return err
		}

		exec := e.StringGetter(cmd)
		if _, err := exec(); err != nil {
			return err
		}

		return nil
	}
}

// BoolSetter invokes script with parameter replaced by bool value
func (e *Script) BoolSetter(param, script string) func(bool) error {
	// return func to access cached value
	return func(b bool) error {
		cmd, err := util.ReplaceFormatted(script, map[string]interface{}{
			param: b,
		})
		if err != nil {
			return err
		}

		exec := e.StringGetter(cmd)
		if _, err := exec(); err != nil {
			return err
		}

		return nil
	}
}
