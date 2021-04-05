package client

import (
	"crypto/tls"
	_ "embed"
)

//go:embed client-cert.pem
var cert []byte

//go:embed client-key.pem
var key []byte

func Certificate() (tls.Certificate, error) {
	return tls.X509KeyPair(cert, key)
}
