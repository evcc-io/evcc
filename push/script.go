package push

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/kballard/go-shellquote"
)

// Script implements shell script-based message service and setters
type Script struct {
	log     *util.Logger
	script  string
	timeout time.Duration
}

type scriptConfig struct {
	CmdLine string
	Timeout time.Duration
}

// NewScriptMessenger creates a Script messenger. Script execution is aborted after given timeout.
func NewScriptMessenger(script string, timeout time.Duration) (*Script, error) {
	s := &Script{
		log:     util.NewLogger("script"),
		script:  script,
		timeout: timeout,
	}

	return s, nil
}

// Send calls the script
func (m *Script) Send(title, msg string) {
	_, err := m.exec(m.script, title, msg)
	if err != nil {
		m.log.ERROR.Printf("script: %v", err)
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
