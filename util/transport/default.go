package transport

import (
	"crypto/tls"
	"net/http"
)

// Default returns http.DefaultTransport as http.Transport instead of http.RoundTripper
func Default() *http.Transport {
	t, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		panic("http.DefaultTransport is not an http.Transport")
	}
	return t
}

// InsecureTransport is an http.Transport with TLSClientConfig.InsecureSkipVerify enabled
func Insecure() *http.Transport {
	t := Default()
	t.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	return t
}
