package cert

import (
	"crypto/x509"

	"github.com/pkg/errors"
)

func VerifyCertificateChain(cert *x509.Certificate, caCert *x509.Certificate) error {
	if caCert.CheckSignatureFrom(caCert) != nil {
		return errors.New("consortium certificate is not a root CA (not self-signed)")
	}

	if err := cert.CheckSignatureFrom(caCert); err != nil {
		return errors.Wrap(err, "certificate is not signed by consortium CA")
	}

	if !caCert.IsCA {
		return errors.New("consortium certificate is not marked as CA")
	}
	return nil
}
