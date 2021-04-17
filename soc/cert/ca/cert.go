package ca

import (
	"bytes"
	_ "embed"
)

//go:embed ca-cert.pem
var cert []byte

func PEM() []byte {
	copy := bytes.NewBuffer(cert)
	return copy.Bytes()
}
