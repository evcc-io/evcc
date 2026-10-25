package templates

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	strictRequired = true
	os.Exit(m.Run())
}
