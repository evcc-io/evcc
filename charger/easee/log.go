package easee

import (
	"fmt"
	"strings"

	"github.com/philippseith/signalr"
	"golang.org/x/exp/slices"
)

// Logger is a simple logger interface
type Logger interface {
	Println(v ...interface{})
}

type logger struct {
	log Logger
}

func SignalrLogger(log Logger) signalr.StructuredLogger {
	return &logger{log: log}
}

var skipped = []string{"ts", "class", "hub", "protocol", "value"}

func (l *logger) Log(keyVals ...interface{}) error {
	b := new(strings.Builder)

	var skip bool
	for i, v := range keyVals {
		if skip {
			skip = false
			continue
		}

		if i%2 == 0 {
			// don't log if key is not a string or if key should be skipped
			if _, ok := v.(string); !ok || slices.Contains(skipped, v.(string)) {
				skip = true
				continue
			}

			if b.Len() > 0 {
				b.WriteRune(' ')
			}

			b.WriteString(fmt.Sprintf("%v", v))
			b.WriteRune('=')
		} else {
			b.WriteString(fmt.Sprintf("%v", v))
		}
	}

	l.log.Println(b.String())

	return nil
}
