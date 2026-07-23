package mercedes

import (
	"crypto/tls"
	"crypto/x509"
	_ "embed"
	"net/http"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
)

// Since 2026-07-23 the Mercedes-Benz API endpoints (*.mobilesdk.mercedes-benz.com)
// serve a Let's Encrypt "Generation Y" certificate chain (leaf <- YR2 <- ISRG Root YR)
// without the cross-sign to ISRG Root X1. ISRG Root YR is not yet included in the
// common system trust stores, so certificate verification fails with
// "x509: certificate signed by unknown authority". Until Mercedes serves the complete
// chain or the new root reaches the trust stores, the official cross-signed root
// (https://letsencrypt.org/certs/gen-y/root-yr-by-x1.pem, valid until 2032-09-02)
// is added to the verification pool. Verification itself remains fully enabled.
//
//go:embed isrg-root-yr-by-x1.pem
var isrgRootYrByX1 []byte

// newTransport returns the default logging transport with the cross-signed
// Let's Encrypt root added to the system certificate pool
func newTransport(log *util.Logger) http.RoundTripper {
	pool, err := x509.SystemCertPool()
	if err != nil {
		pool = x509.NewCertPool()
	}
	pool.AppendCertsFromPEM(isrgRootYrByX1)

	t := transport.Default()
	t.TLSClientConfig = &tls.Config{RootCAs: pool}

	return request.NewTripper(log, t)
}
