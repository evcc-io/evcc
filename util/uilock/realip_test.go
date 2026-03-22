package uilock

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCIDRList(t *testing.T) {
	nets, err := ParseCIDRList([]string{"127.0.0.1/32", "10.0.0.0/8"})
	require.NoError(t, err)
	require.Len(t, nets, 2)

	_, err = ParseCIDRList([]string{"not-a-cidr"})
	require.Error(t, err)
}

func TestEffectiveClientIP(t *testing.T) {
	t.Parallel()

	cidr10, err := ParseCIDRList([]string{"10.0.0.0/8"})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.1.2.3:1234"
	req.Header.Set("X-Forwarded-For", "203.0.113.5, 198.51.100.1")
	ip := EffectiveClientIP(req, cidr10)
	assert.Equal(t, "203.0.113.5", ip.String())

	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.RemoteAddr = "198.51.100.1:443"
	req2.Header.Set("X-Forwarded-For", "203.0.113.5")
	ip2 := EffectiveClientIP(req2, cidr10)
	assert.Equal(t, "198.51.100.1", ip2.String())
}

func TestIpMatchesList(t *testing.T) {
	t.Parallel()
	ip := net.ParseIP("127.0.0.1")
	require.NotNil(t, ip)
	assert.True(t, ipMatchesList(ip, []string{"127.0.0.1", "::1"}))
	assert.False(t, ipMatchesList(ip, []string{"10.0.0.1"}))
}
