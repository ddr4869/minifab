package msp

import (
	"crypto"
	"crypto/x509"
	"io"

	"github.com/pkg/errors"
)

type Signer struct {
	Identity   identity
	PrivateKey crypto.PrivateKey
	PublicKey  crypto.PublicKey
}

func NewSigner(identity *identity, privateKey crypto.PrivateKey) (*Signer, error) {

	signer := &Signer{
		Identity:   *identity,
		PrivateKey: privateKey,
		PublicKey:  identity.pk,
	}

	if err := signer.Identity.Validate(); err != nil {
		return nil, errors.Wrap(err, "loaded identity is invalid")
	}

	return signer, nil
}

func (s *Signer) Public() crypto.PublicKey {
	return s.PublicKey
}

func (s *Signer) Sign(rand io.Reader, digest []byte, opts crypto.SignerOpts) ([]byte, error) {

	signer, ok := s.PrivateKey.(crypto.Signer)
	if !ok {
		return nil, errors.New("private key does not implement crypto.Signer")
	}

	signature, err := signer.Sign(rand, digest, opts)
	if err != nil {
		return nil, err
	}
	return signature, nil
}

func (s *Signer) GetIdentifier() *IdentityIdentifier {
	return s.Identity.GetIdentifier()
}

func (s *Signer) GetCertificate() *x509.Certificate {
	return s.Identity.GetCertificate()
}

func (s *Signer) Validate() error {
	return s.Identity.Validate()
}
