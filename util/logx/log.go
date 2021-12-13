package logx

import (
	"os"
	"time"

	kit "github.com/go-kit/log"
)

type Logger = kit.Logger

var root = newRoot("component", "main")

const (
	MODULE = "module"
)

func newRoot(keyvals ...interface{}) kit.Logger {
	var log kit.Logger
	w := kit.NewSyncWriter(os.Stdout)
	log = &legacyLogger{w, kit.NewLogfmtLogger(w)}
	log = kit.With(log, "ts", kit.TimestampFormat(time.Now, "2006/01/02 15:04:05"))
	log = kit.With(log, keyvals...)
	return log
}

// New created a new logger as decendent of the root logger
func New(keyvals ...interface{}) kit.Logger {
	return kit.With(root, keyvals...)
}

// New created a new logger as decendent of the root logger
func NewModule(module string) kit.Logger {
	return kit.WithPrefix(root, MODULE, module)
}
