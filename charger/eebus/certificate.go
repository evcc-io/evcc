package eebus

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"

	"github.com/enbility/ship-go/cert"
)

// CreateCertificate returns a newly created EEBUS compatible certificate
func CreateCertificate() (tls.Certificate, error) {
	return cert.CreateCertificate("", EEBUSBrandName, "DE", EEBUSDeviceCode)
}

// pemBlockForKey marshals private key into pem block
func pemBlockForKey(priv interface{}) (*pem.Block, error) {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k)}, nil
	case *ecdsa.PrivateKey:
		b, err := x509.MarshalECPrivateKey(k)
		if err != nil {
			return nil, fmt.Errorf("unable to marshal ECDSA private key: %w", err)
		}
		return &pem.Block{Type: "EC PRIVATE KEY", Bytes: b}, nil
	default:
		return nil, errors.New("unknown private key type")
	}
}

// GetX509KeyPair saves returns the cert and key string values
func GetX509KeyPair(cert tls.Certificate) (string, string, error) {
	var certValue, keyValue string

	out := new(bytes.Buffer)
	err := pem.Encode(out, &pem.Block{Type: "CERTIFICATE", Bytes: cert.Certificate[0]})
	if err == nil {
		certValue = out.String()
	}

	if len(certValue) > 0 {
		var pb *pem.Block
		if pb, err = pemBlockForKey(cert.PrivateKey); err == nil {
			out.Reset()
			err = pem.Encode(out, pb)
		}
	}

	if err == nil {
		keyValue = out.String()
	}

	return certValue, keyValue, err
}

// SkiFromX509 extracts SKI from certificate
func skiFromX509(leaf *x509.Certificate) (string, error) {
	if len(leaf.SubjectKeyId) == 0 {
		return "", errors.New("missing SubjectKeyId")
	}
	return fmt.Sprintf("%0x", leaf.SubjectKeyId), nil
}

// SkiFromCert extracts SKI from certificate
func SkiFromCert(cert tls.Certificate) (string, error) {
	leaf, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return "", errors.New("failed parsing certificate: " + err.Error())
	}
	return skiFromX509(leaf)
}
