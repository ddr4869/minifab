package msp

import (
	"crypto"
)

type MSP interface {
	Setup(config *MSPConfig) error
	GetIdentifier() *IdentityIdentifier
	// ValidateIdentity(identity Identity) error
	// DeserializeIdentity(serializedIdentity []byte) (Identity, error)
	// IsWellFormed(identity *SerializedIdentity) error
}

// Identity 인터페이스
type Identity interface {
	GetIdentifier() *IdentityIdentifier
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
