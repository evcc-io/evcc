package request

import (
	"testing"
)

func EnableRequestLogging(t *testing.T) {
	oldLogHeaders := LogHeaders
	LogHeaders = true
	t.Cleanup(func() {
		LogHeaders = oldLogHeaders
	})
}
