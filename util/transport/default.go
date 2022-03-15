package transport

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"
)

// Default returns an http.DefaultTransport as http.Transport with reduced dial timeout
func Default() *http.Transport {
	return &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second, // reduced from 30s
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
}

// InsecureTransport is an http.Transport with TLSClientConfig.InsecureSkipVerify enabled
func Insecure() *http.Transport {
	t := Default()
	t.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	return t
}
