package push

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/kballard/go-shellquote"
)

func init() {
	registry.Add("script", NewScriptFromConfig)
}

// Script implements shell script-based message service and setters
type Script struct {
	log     *util.Logger
	script  string
	timeout time.Duration
}

// NewScriptFromConfig creates a Script messenger. Script execution is aborted after given timeout.
func NewScriptFromConfig(other map[string]interface{}) (Messenger, error) {
	cc := struct {
		CmdLine string
		Timeout time.Duration
	}{
		Timeout: request.Timeout,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	s := &Script{
		log:     util.NewLogger("script"),
		script:  cc.CmdLine,
		timeout: cc.Timeout,
	}

	return s, nil
}

// Send calls the script
func (m *Script) Send(title, msg string) {
	_, err := m.exec(m.script, title, msg)
	if err != nil {
		m.log.ERROR.Printf("exec: %v", err)
	}
}

func (m *Script) exec(script, title, msg string) (string, error) {
	args, err := shellquote.Split(script)
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
	defer cancel()

	args = append(args, title, msg)
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
