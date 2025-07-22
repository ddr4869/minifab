package msp

import (
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"time"

	"github.com/pkg/errors"
)

// MSP 설정 구조체
type MSPConfig struct {
	MSPID           string
	SigningIdentity *SigningIdentity
	RootCerts       []*x509.Certificate
	//Admins                        []*x509.Certificate
	// RevocationList                []*x509.Certificate
	// OrganizationalUnitIdentifiers []*FabricOUIdentifier
	// CryptoConfig                  *FabricCryptoConfig

	// NodeOUs *FabricNodeOUs
}

type FabricMSP struct {
	MSPID           string
	SigningIdentity SigningIdentity
	RootCerts       []*x509.Certificate
	// Admins          []*identity
	// Bccsp           BCCSP
	//CryptoConfig    *FabricCryptoConfig
	// OuIdentifiers   map[string]*FabricOUIdentifier
	// OuEnforcement   bool
	// NodeOUs         *FabricNodeOUs
}

// Fabric 암호화 설정
type FabricCryptoConfig struct {
	SignatureHashFamily            string
	IdentityIdentifierHashFunction string
}

const (
	MSPPrincipal_ROLE              MSPPrincipal_Classification = 0
	MSPPrincipal_ORGANIZATION_UNIT MSPPrincipal_Classification = 1
	MSPPrincipal_IDENTITY          MSPPrincipal_Classification = 2
)

// MSP Principal
type MSPPrincipal struct {
	PrincipalClassification MSPPrincipal_Classification
	Principal               []byte
}

type MSPPrincipal_Classification int32

func (msp *FabricMSP) GetIdentifier() *IdentityIdentifier {
	return msp.SigningIdentity.GetIdentifier()
}

func NewFabricMSP() *FabricMSP {
	return &FabricMSP{}
}

// Setup MSP 설정
func (msp *FabricMSP) Setup(config *MSPConfig) error {
	if config == nil {
		return errors.New("MSP config cannot be nil")
	}

	msp.MSPID = config.MSPID
	msp.SigningIdentity = *config.SigningIdentity
	msp.RootCerts = config.RootCerts
	// msp.CryptoConfig = config.CryptoConfig
	// msp.NodeOUs = config.NodeOUs
	return nil
}

// DeserializeIdentity 직렬화된 Identity를 역직렬화
func (msp *FabricMSP) DeserializeIdentity(serializedIdentity []byte) (Identity, error) {
	return &identity{
		id: &IdentityIdentifier{
			Mspid: msp.MSPID,
			Id:    "user",
		},
		// mock x509.Certificate
		cert: &x509.Certificate{
			Subject: pkix.Name{
				CommonName: "test",
			},
			NotBefore:          time.Now(),
			NotAfter:           time.Now().Add(time.Hour * 24 * 365),
			PublicKeyAlgorithm: x509.RSA,
			PublicKey:          nil,
		},
		pk: &rsa.PublicKey{
			N: big.NewInt(1),
			E: 1,
		}, // mock crypto.PublicKey
	}, nil
}

// IsWellFormed Identity가 올바른 형식인지 확인
func (msp *FabricMSP) IsWellFormed(identity *SerializedIdentity) error {
	if identity == nil {
		return errors.New("identity cannot be nil")
	}
	if identity.Mspid != msp.MSPID {
		return errors.New("identity MSP ID does not match")
	}
	return nil
}

func (msp *FabricMSP) ValidateIdentity(identity Identity) error {
	if identity == nil {
		return errors.New("identity cannot be nil")
	}
	return identity.Validate()
}
