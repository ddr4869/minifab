package cert

import (
	"crypto/x509"
	"errors"
	"reflect"
)

func VerifyRootCA(caCert, signCert *x509.Certificate) (bool, error) {
	// 1. CA 인증서가 자체 서명되었는지 확인 (루트 CA의 특징)
	if !caCert.IsCA || !reflect.DeepEqual(caCert.Subject, caCert.Issuer) {
		return false, errors.New("CA certificate is invalid")
	}

	// 2. CA 인증서의 공개키로 서명 인증서를 검증
	err := signCert.CheckSignatureFrom(caCert)
	if err != nil {
		return false, errors.New("failed to verify signature")
	}

	// 3. CA 인증서의 Subject와 Issuer가 일치하는지 확인
	if !reflect.DeepEqual(caCert.Subject, caCert.Issuer) {
		return false, errors.New("CA certificate is invalid")
	}

	// 4. CA 인증서가 CA 용도로 설정되었는지 확인
	if !caCert.IsCA {
		return false, errors.New("CA certificate is invalid")
	}

	return true, nil
}
