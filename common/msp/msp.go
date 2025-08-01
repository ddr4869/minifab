package msp

import (
	"crypto"
	"crypto/x509"
)

type MSP interface {
	Setup(config *MSPConfig) error
	GetSigningIdentity() SigningIdentity
	GetRootCertificates() *x509.Certificate
	// ValidateIdentity(identity Identity) error
	// DeserializeIdentity(serializedIdentity []byte) (Identity, error)
	// IsWellFormed(identity *SerializedIdentity) error
}

// Identity 인터페이스
type Identity interface {
	GetIdentifier() *IdentityIdentifier
	GetCertificate() *x509.Certificate
	Validate() error
	// GetMSPIdentifier() string
	// GetOrganizationalUnits() []*OUIdentifier
	// Verify(msg []byte, sig []byte) error
	// Serialize() ([]byte, error)
	// SatisfiesPrincipal(principal *MSPPrincipal) error
}

type SigningIdentity interface {
	Identity
	crypto.Signer
}

// 키 인터페이스
type Key interface {
	Bytes() ([]byte, error)
	SKI() []byte
	Symmetric() bool
	Private() bool
	PublicKey() (Key, error)
}
