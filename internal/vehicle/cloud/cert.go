package cloud

import (
	"bytes"
	"crypto/tls"
	_ "embed"
)

//go:embed client-cert.pem
var cert []byte

//go:embed client-key.pem
var key []byte

func clientCertificate() (tls.Certificate, error) {
	return tls.X509KeyPair(cert, key)
}

//go:embed ca-cert.pem
var caCert []byte

func caPEM() []byte {
	copy := bytes.NewBuffer(caCert)
	return copy.Bytes()
}
