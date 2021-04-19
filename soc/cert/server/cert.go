package server

import (
	"bytes"
	"crypto/tls"
	_ "embed"
	"encoding/base64"

	"github.com/andig/evcc/util"
)

//go:embed server-cert.pem
var cert []byte

//--go:embed server-key.pem
var key []byte

func init() {
	var err error
	key, err = base64.StdEncoding.DecodeString(util.Getenv("SERVER_KEY"))
	if err != nil {
		panic(err)
	}
}

func PEM() []byte {
	copy := bytes.NewBuffer(cert)
	return copy.Bytes()
}

func Certificate() (tls.Certificate, error) {
	return tls.X509KeyPair(cert, key)
}
