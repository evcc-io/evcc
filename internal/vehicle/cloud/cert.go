package cloud

import (
	"bytes"
	_ "embed"
)

//go:embed ca-cert.pem
var caCert []byte

func caPEM() []byte {
	copy := bytes.NewBuffer(caCert)
	return copy.Bytes()
}
