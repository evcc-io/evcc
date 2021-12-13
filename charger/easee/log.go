package easee

import (
	"github.com/evcc-io/evcc/util/logx"
	"github.com/philippseith/signalr"
	"github.com/thoas/go-funk"
)

type logger struct {
	log logx.Logger
}

func SignalrLogger(log logx.Logger) signalr.StructuredLogger {
	return &logger{log: log}
}

var skipped = []string{"ts", "class", "hub", "protocol", "value"}

func (l *logger) Log(keyVals ...interface{}) error {
	var kv []interface{}

	var skip bool
	for i, v := range keyVals {
		if skip {
			skip = false
			continue
		}

		if i%2 == 0 {
			if funk.Contains(skipped, v) {
				skip = true
				continue
			}
		}
		kv = append(kv, v)
	}

	return l.log.Log(kv...)
}
