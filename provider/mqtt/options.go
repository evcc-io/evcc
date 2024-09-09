//go:build mqtt

package mqtt

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"

	paho "github.com/eclipse/paho.mqtt.golang"
)

type Option func(*paho.ClientOptions)

func (c Config) WithTlsConfig(tlsConfig *tls.Config) Option {
	return func(o *paho.ClientOptions) {
		o.SetTLSConfig(tlsConfig)
	}
}

func (c Config) TlsConfig() (*tls.Config, error) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: c.Insecure,
	}

	if c.CaCert != "" {
		caCertPool := x509.NewCertPool()
		if ok := caCertPool.AppendCertsFromPEM([]byte(c.CaCert)); !ok {
			return nil, fmt.Errorf("failed to add ca cert to cert pool")
		}
		tlsConfig.RootCAs = caCertPool
	}

	if c.ClientCert != "" && c.ClientKey != "" {
		clientKeyPair, err := tls.X509KeyPair([]byte(c.ClientCert), []byte(c.ClientKey))
		if err != nil {
			return nil, fmt.Errorf("failed to add client cert: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{clientKeyPair}
	}

	return tlsConfig, nil
}

// func (c Config) Options() []Option {
// 	var res []Option
// 	if c.Insecure || c.CaCert != "" || c.ClientCert != "" {
// 		res = append(res, c.WithTlsConfig())
// 	}
// 	return res
// }
