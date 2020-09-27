package request

import (
	"crypto/tls"
	"net/http"
)

// Transport decorates http.Transport with fluent style
type Transport struct {
	*http.Transport
}

// NewTransport creates an HTTP transport
func NewTransport() *Transport {
	t := &Transport{
		Transport: http.DefaultTransport.(*http.Transport).Clone(),
	}

	return t
}

// WithTLSConfig sets the transports TLS configuration
func (t *Transport) WithTLSConfig(tls *tls.Config) *Transport {
	t.Transport.TLSClientConfig = tls
	return t
}
