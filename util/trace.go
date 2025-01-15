package util

import (
	"context"
	"runtime/trace"
)

func TraceRegion(name string, f func() error) error {
	var err error
	trace.WithRegion(context.Background(), name, func() {
		err = f()
	})
	return err
}
