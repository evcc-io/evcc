package plugwise

import (
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConnection(t *testing.T) {
	c, err := NewConnection("192.168.0.1", "testpass", time.Second)
	require.NoError(t, err)
	assert.NotNil(t, c)
}

func TestCurrentPower(t *testing.T) {
	fixture, err := os.ReadFile("testdata/domain_objects.xml")
	require.NoError(t, err, "testdata/domain_objects.xml must exist")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixture)
	}))
	defer srv.Close()

	c, err := NewConnection(srv.URL, "testpass", time.Second)
	require.NoError(t, err)

	power, err := c.CurrentPower()
	require.NoError(t, err)
	// electricity_consumed (312.0 W) - electricity_produced (0.0 W) = 312.0 W
	assert.Equal(t, 312.0, power)
}

func TestCaching(t *testing.T) {
	fixture, err := os.ReadFile("testdata/domain_objects.xml")
	require.NoError(t, err, "testdata/domain_objects.xml must exist")

	var requestCount atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixture)
	}))
	defer srv.Close()

	c, err := NewConnection(srv.URL, "testpass", time.Second)
	require.NoError(t, err)

	// First call — hits the server
	_, err = c.CurrentPower()
	require.NoError(t, err)

	// Second call within the 1-second TTL — must be served from cache
	_, err = c.CurrentPower()
	require.NoError(t, err)

	// Only one HTTP request should have been made
	assert.Equal(t, int32(1), requestCount.Load(), "second CurrentPower() call within TTL must not make a second HTTP request")
}
