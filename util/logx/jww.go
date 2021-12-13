package logx

import (
	"fmt"
	"io"
	"strings"

	"github.com/evcc-io/evcc/util"
)

type legacyLogger struct {
	w   io.Writer
	log Logger
}

func (a *legacyLogger) Log(keyvals ...interface{}) error {
	var kv []interface{}

	var component, level, ts, payload interface{}

	for i := 0; i < len(keyvals); i += 2 {
		switch keyvals[i] {
		case "component":
			component = keyvals[i+1]
		case "level":
			level = keyvals[i+1]
		case "ts":
			ts = keyvals[i+1]
		case "payload":
			payload = keyvals[i+1]
		default:
			kv = append(kv, keyvals[i], keyvals[i+1])
		}
	}

	lvl := fmt.Sprint(level)
	if len(lvl) > 4 {
		lvl = lvl[:4]
	}

	fmt.Fprintf(a.w, "[%-6s] %-4s %s ", component, strings.ToUpper(lvl), ts)
	err := a.log.Log(kv...)
	if payload != nil {
		fmt.Fprintf(a.w, "%s\n", payload)
	}
	return err
}

type jwwAdapter struct {
	Logger
}

func NewJwwAdapter(kitLogger Logger, area string) *util.Logger {
	jwwAdapter := &jwwAdapter{kitLogger}
	jwwLogger := util.NewLogger(area)
	jwwLogger.SetLogOutput(jwwAdapter)
	return jwwLogger
}

func (a *jwwAdapter) Write(p []byte) (int, error) {
	err := a.Logger.Log("msg", string(p))
	return len(p), err
}
