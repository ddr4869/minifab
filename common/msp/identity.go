package msp

import (
	"crypto"
	"crypto/x509"
	"fmt"
	"time"

	"github.com/pkg/errors"
)

type IdentityIdentifier struct {
	Mspid string
	Id    string
}

type identity struct {
	id   *IdentityIdentifier
	cert *x509.Certificate
	pk   crypto.PublicKey
}

type SerializedIdentity struct {
	Mspid   string
	IdBytes []byte
}

func NewIdentity(cert *x509.Certificate, pk crypto.PublicKey, mspID string) *identity {
	return &identity{
		cert: cert,
		pk:   pk,
		id: &IdentityIdentifier{
			Mspid: mspID,
			Id:    cert.Subject.CommonName, // check this
		},
	}
}

func (id *identity) GetIdentifier() *IdentityIdentifier {
	return id.id
}

// GetMSPIdentifier MSP 식별자 반환
func (id *identity) GetMSPIdentifier() string {
	return id.id.Mspid
}

// Validate Identity 검증
func (id *identity) Validate() error {
	if id.id == nil {
		return errors.New("identity identifier cannot be nil")
	}
	if id.cert == nil {
		return errors.New("certificate cannot be nil")
	}

	// 인증서 유효성 검사
	if id.cert.NotAfter.Before(time.Now()) {
		return errors.New("certificate has expired")
	}

	return nil
}

// GetOrganizationalUnits 조직 단위 반환
func (id *identity) GetOrganizationalUnits() []*OUIdentifier {
	ous := make([]*OUIdentifier, 0)

	if id.cert != nil {
		for _, ou := range id.cert.Subject.OrganizationalUnit {
			ous = append(ous, &OUIdentifier{
				OrganizationalUnitIdentifier: ou,
			})
		}
	}

	return ous
}

// Verify 서명 검증
func (id *identity) Verify(msg []byte, sig []byte) error {
	if id.pk == nil {
		return errors.New("public key cannot be nil")
	}

	// 실제 구현에서는 서명 알고리즘에 따른 검증 수행
	// 여기서는 간단한 구현만 제공
	return nil
}

// Serialize Identity 직렬화
func (id *identity) Serialize() ([]byte, error) {
	if id.cert == nil {
		return nil, errors.New("certificate cannot be nil")
	}

	// 실제 구현에서는 protobuf를 사용하여 직렬화
	serialized := &SerializedIdentity{
		Mspid:   id.GetMSPIdentifier(),
		IdBytes: id.cert.Raw,
	}

	// 간단한 직렬화 (실제로는 protobuf 사용)
	return []byte(fmt.Sprintf("%s:%s", serialized.Mspid, string(serialized.IdBytes))), nil
}

// SatisfiesPrincipal Principal 조건 확인
func (id *identity) SatisfiesPrincipal(principal *MSPPrincipal) error {
	if principal == nil {
		return errors.New("principal cannot be nil")
	}

	switch principal.PrincipalClassification {
	case MSPPrincipal_ROLE:
		return id.satisfiesRole(principal.Principal)
	case MSPPrincipal_ORGANIZATION_UNIT:
		return id.satisfiesOU(principal.Principal)
	case MSPPrincipal_IDENTITY:
		return id.satisfiesIdentity(principal.Principal)
	default:
		return errors.New("unsupported principal classification")
	}
}

// satisfiesRole 역할 기반 조건 확인
func (id *identity) satisfiesRole(roleBytes []byte) error {
	// 실제 구현에서는 역할 정보를 파싱하여 확인
	return nil
}

// satisfiesOU 조직 단위 기반 조건 확인
func (id *identity) satisfiesOU(ouBytes []byte) error {
	// 실제 구현에서는 OU 정보를 파싱하여 확인
	return nil
}

// satisfiesIdentity Identity 기반 조건 확인
func (id *identity) satisfiesIdentity(identityBytes []byte) error {
	// 실제 구현에서는 Identity 정보를 파싱하여 확인
	return nil
}
