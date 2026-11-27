package core

import (
	"os"
	"testing"

	"github.com/evcc-io/evcc/util/sponsor"
)

// TestMain sponsors the package once before any test starts, so goroutines leaked by
// fire-and-forget optimizer triggers never race with a test writing sponsor.Subject
func TestMain(m *testing.M) {
	sponsor.Subject = "test"
	os.Exit(m.Run())
}
