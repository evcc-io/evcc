package sma

import (
	"fmt"

	"github.com/evcc-io/evcc/util/logx"
)

type logAdapter struct {
	log logx.Logger
}

func (a *logAdapter) Printf(format string, v ...interface{}) {
	_ = a.log.Log("msg", fmt.Sprintf(format, v...))
}
