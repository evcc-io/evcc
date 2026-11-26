package mqtt

import (
	"net"
	"testing"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/require"
)

// the connection manager retries forever, the client must fail on the first error instead
func TestConnectError(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := l.Addr().String()
	require.NoError(t, l.Close())

	start := time.Now()
	_, err = NewClient(util.NewLogger("foo"), addr, "", "", ClientID(), 1, false, "", "", "")
	require.Error(t, err)
	require.Less(t, time.Since(start), 5*time.Second)
}
