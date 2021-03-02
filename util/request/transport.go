package request

import (
	"crypto/tls"
	"net/http"
)

// DefaultTransport returns http.DefaultTransport as http.Transport instead of http.RoundTripper
func DefaultTransport() *http.Transport {
	t, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		panic("http.DefaultTransport is not an http.Transport")
	}
	return t
}

// InsecureTransport is an http.Transport with TLSClientConfig.InsecureSkipVerify enabled
func InsecureTransport() *http.Transport {
	t := DefaultTransport()
	t.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	return t
}
