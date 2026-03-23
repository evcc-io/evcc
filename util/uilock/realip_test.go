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
	t.Parallel()

	tests := []struct {
		name     string
		input    []string
		wantLen  int
		wantNets []string
	}{
		{
			name:     "valid IPv4 CIDRs",
			input:    []string{"127.0.0.1/32", "10.0.0.0/8"},
			wantLen:  2,
			wantNets: []string{"127.0.0.1/32", "10.0.0.0/8"},
		},
		{
			name:     "plain IPv4 normalized to /32",
			input:    []string{"192.168.1.1"},
			wantLen:  1,
			wantNets: []string{"192.168.1.1/32"},
		},
		{
			name:     "plain IPv6 normalized to /128",
			input:    []string{"::1"},
			wantLen:  1,
			wantNets: []string{"::1/128"},
		},
		{
			name:     "valid IPv6 CIDR",
			input:    []string{"fd00::/64"},
			wantLen:  1,
			wantNets: []string{"fd00::/64"},
		},
		{
			name:    "invalid entry skipped",
			input:   []string{"not-a-cidr"},
			wantLen: 0,
		},
		{
			name:     "mixed valid and invalid",
			input:    []string{"10.0.0.0/8", "garbage", "192.168.1.0/24"},
			wantLen:  2,
			wantNets: []string{"10.0.0.0/8", "192.168.1.0/24"},
		},
		{
			name:    "empty and whitespace entries",
			input:   []string{"", "  ", "\t"},
			wantLen: 0,
		},
		{
			name:     "whitespace around valid entry",
			input:    []string{"  10.0.0.0/8  "},
			wantLen:  1,
			wantNets: []string{"10.0.0.0/8"},
		},
		{
			name:    "nil input",
			input:   nil,
			wantLen: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			nets := ParseCIDRList(tc.input)
			require.Len(t, nets, tc.wantLen)
			for i, want := range tc.wantNets {
				assert.Equal(t, want, nets[i].String())
			}
		})
	}
}

func TestEffectiveClientIP(t *testing.T) {
	t.Parallel()

	cidr10 := ParseCIDRList([]string{"10.0.0.0/8"})
	require.Len(t, cidr10, 1)

	tests := []struct {
		name           string
		remoteAddr     string
		xff            string
		xRealIP        string
		trustedProxies []*net.IPNet
		want           string
	}{
		{
			name:           "trusted proxy with X-Forwarded-For",
			remoteAddr:     "10.1.2.3:1234",
			xff:            "203.0.113.5, 198.51.100.1",
			trustedProxies: cidr10,
			want:           "203.0.113.5",
		},
		{
			name:           "non-trusted proxy ignores XFF",
			remoteAddr:     "198.51.100.1:443",
			xff:            "203.0.113.5",
			trustedProxies: cidr10,
			want:           "198.51.100.1",
		},
		{
			name:           "trusted proxy falls back to X-Real-IP",
			remoteAddr:     "10.0.0.1:80",
			xRealIP:        "172.16.0.5",
			trustedProxies: cidr10,
			want:           "172.16.0.5",
		},
		{
			name:           "trusted proxy falls back to RemoteAddr when headers missing",
			remoteAddr:     "10.0.0.1:80",
			trustedProxies: cidr10,
			want:           "10.0.0.1",
		},
		{
			name:           "empty trustedProxies ignores forwarded headers",
			remoteAddr:     "10.0.0.1:80",
			xff:            "203.0.113.5",
			trustedProxies: nil,
			want:           "10.0.0.1",
		},
		{
			name:           "IPv6 RemoteAddr",
			remoteAddr:     "[::1]:1234",
			trustedProxies: nil,
			want:           "::1",
		},
		{
			name:           "IPv6 trusted proxy with XFF containing bracketed IPv6",
			remoteAddr:     "[fd00::1]:443",
			xff:            "[2001:db8::1]",
			trustedProxies: ParseCIDRList([]string{"fd00::/16"}),
			want:           "2001:db8::1",
		},
		{
			name:           "malformed RemoteAddr returns nil",
			remoteAddr:     "not-an-ip",
			trustedProxies: nil,
			want:           "<nil>",
		},
		{
			name:           "trusted proxy with invalid XFF falls back to X-Real-IP",
			remoteAddr:     "10.0.0.1:80",
			xff:            "not-an-ip",
			xRealIP:        "192.168.1.1",
			trustedProxies: cidr10,
			want:           "192.168.1.1",
		},
		{
			name:           "trusted proxy with invalid XFF and invalid X-Real-IP falls back to RemoteAddr",
			remoteAddr:     "10.0.0.1:80",
			xff:            "garbage",
			xRealIP:        "also-garbage",
			trustedProxies: cidr10,
			want:           "10.0.0.1",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tc.remoteAddr
			if tc.xff != "" {
				req.Header.Set("X-Forwarded-For", tc.xff)
			}
			if tc.xRealIP != "" {
				req.Header.Set("X-Real-IP", tc.xRealIP)
			}
			ip := EffectiveClientIP(req, tc.trustedProxies)
			if tc.want == "<nil>" {
				assert.Nil(t, ip)
			} else {
				require.NotNil(t, ip)
				assert.Equal(t, tc.want, ip.String())
			}
		})
	}
}

func TestIpMatchesList(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ip   string
		list []string
		want bool
	}{
		{
			name: "IPv4 match",
			ip:   "127.0.0.1",
			list: []string{"127.0.0.1", "::1"},
			want: true,
		},
		{
			name: "IPv4 no match",
			ip:   "127.0.0.1",
			list: []string{"10.0.0.1"},
			want: false,
		},
		{
			name: "IPv6 match",
			ip:   "::1",
			list: []string{"127.0.0.1", "::1"},
			want: true,
		},
		{
			name: "IPv6 no match",
			ip:   "::1",
			list: []string{"::2"},
			want: false,
		},
		{
			name: "whitespace around entry",
			ip:   "10.0.0.1",
			list: []string{"  10.0.0.1  "},
			want: true,
		},
		{
			name: "empty entries skipped",
			ip:   "10.0.0.1",
			list: []string{"", "  ", "10.0.0.1"},
			want: true,
		},
		{
			name: "empty list",
			ip:   "10.0.0.1",
			list: nil,
			want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ip := net.ParseIP(tc.ip)
			require.NotNil(t, ip)
			assert.Equal(t, tc.want, ipMatchesList(ip, tc.list))
		})
	}
}
