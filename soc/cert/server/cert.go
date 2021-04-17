package server

import (
	"bytes"
	"crypto/tls"
	_ "embed"

	"github.com/andig/evcc/util"
)

//go:embed server-cert.pem
var cert []byte

//--go:embed server-key.pem
var key []byte

func init() {
	key = []byte(util.Getenv("SERVER_KEY"))
}

func PEM() []byte {
	copy := bytes.NewBuffer(cert)
	return copy.Bytes()
}

func Certificate() (tls.Certificate, error) {
	return tls.X509KeyPair(cert, key)
}
