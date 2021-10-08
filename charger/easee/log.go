package easee

import (
	"fmt"
	"strings"

	"github.com/philippseith/signalr"
)

// Logger is a simple logger interface
type Logger interface {
	Println(v ...interface{})
}

type logger struct {
	b   strings.Builder
	log Logger
}

func SignalrLogger(log Logger) signalr.StructuredLogger {
	return &logger{log: log}
}

func (l *logger) Log(keyVals ...interface{}) error {
	for i, v := range keyVals {
		if i%2 == 0 {
			if l.b.Len() > 0 {
				l.b.WriteRune(' ')
			}
			l.b.WriteString(fmt.Sprintf("%v", v))
			l.b.WriteRune('=')
		} else {
			l.b.WriteString(fmt.Sprintf("%v", v))
		}
	}

	l.log.Println(l.b.String())

	return nil
}
