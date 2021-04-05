package server

import (
	"bytes"
	"crypto/tls"
	_ "embed"
)

//go:embed server-cert.pem
var cert []byte

//go:embed server-key.pem
var key []byte

func PEM() []byte {
	copy := bytes.NewBuffer(cert)
	return copy.Bytes()
}

func Certificate() (tls.Certificate, error) {
	return tls.X509KeyPair(cert, key)
}
