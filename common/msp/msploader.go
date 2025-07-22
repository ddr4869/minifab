package msp

import (
	"crypto"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

// CreateMSPFromFiles MSP 파일들로부터 MSP 생성
func CreateMSPFromFiles(mspID, mspPath string) (MSP, error) {
	// MSP 구조 검증
	if err := ValidateMSPStructure(mspPath); err != nil {
		return nil, errors.Wrap(err, "MSP structure validation failed")
	}

	// Identity 로드
	msp, err := LoadMSP(mspID, mspPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load identity")
	}

	return msp, nil
}

// LoadMSP MSP 파일들로부터 MSP와 Identity 로드
func LoadMSP(mspID, mspPath string) (MSP, error) {
	cert, err := LoadSignCert(mspPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load sign cert")
	}

	identity := NewIdentity(cert, cert.PublicKey, mspID)

	privateKey, err := LoadPrivateKey(mspPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load private key")
	}

	signer, err := NewSigner(identity, privateKey)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create signer")
	}

	msp, err := NewMSPWithIdentity(mspID, mspPath, signer)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create MSP")
	}

	return msp, nil
}

// NewMSPWithIdentity Identity를 사용하여 MSP 생성
func NewMSPWithIdentity(mspID, mspPath string, identity SigningIdentity) (MSP, error) {
	caCerts, err := LoadCACerts(mspPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load CA certs")
	}

	msp := NewFabricMSP()
	mspConfig := &MSPConfig{
		MSPID:           mspID,
		SigningIdentity: &identity,
		RootCerts:       caCerts,
	}

	if err := msp.Setup(mspConfig); err != nil {
		return nil, errors.Wrap(err, "failed to setup MSP")
	}

	return msp, nil
}

// LoadSignCert signcerts 디렉토리에서 서명 인증서 로드
func LoadSignCert(mspPath string) (*x509.Certificate, error) {
	certPath := filepath.Join(mspPath, "signcerts", "cert.pem")
	return loadCertificateFromFile(certPath)
}

// LoadPrivateKey keystore 디렉토리에서 개인키 로드
func LoadPrivateKey(mspPath string) (crypto.PrivateKey, error) {
	keyPath := filepath.Join(mspPath, "keystore", "key.pem")
	return loadPrivateKeyFromFile(keyPath)
}

// LoadCACerts cacerts 디렉토리에서 CA 인증서들 로드
func LoadCACerts(mspPath string) ([]*x509.Certificate, error) {
	caCertsDir := filepath.Join(mspPath, "cacerts")
	return loadCertificatesFromDir(caCertsDir)
}

// ValidateMSPStructure MSP 디렉토리 구조 검증
func ValidateMSPStructure(mspPath string) error {
	requiredDirs := []string{"signcerts", "keystore", "cacerts"}
	requiredFiles := []string{
		"signcerts/cert.pem",
		"keystore/key.pem",
	}

	// 필수 디렉토리 확인
	for _, dir := range requiredDirs {
		dirPath := filepath.Join(mspPath, dir)
		if err := checkPathExists(dirPath, true); err != nil {
			return errors.Errorf("required directory missing: %s", dir)
		}
	}

	// 필수 파일 확인
	for _, file := range requiredFiles {
		filePath := filepath.Join(mspPath, file)
		if err := checkPathExists(filePath, false); err != nil {
			return errors.Errorf("required file missing: %s", file)
		}
	}

	return nil
}

// loadCertificateFromFile 파일에서 단일 인증서 로드
func loadCertificateFromFile(certPath string) (*x509.Certificate, error) {
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read certificate file %s", certPath)
	}

	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, errors.Errorf("failed to decode PEM block from %s", certPath)
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse certificate from %s", certPath)
	}

	return cert, nil
}

// loadPrivateKeyFromFile 파일에서 개인키 로드
func loadPrivateKeyFromFile(keyPath string) (crypto.PrivateKey, error) {
	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read private key file %s", keyPath)
	}

	block, _ := pem.Decode(keyPEM)
	if block == nil {
		return nil, errors.Errorf("failed to decode PEM block from %s", keyPath)
	}

	// PKCS8 형식 시도
	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		return key, nil
	}

	// PKCS1 RSA 형식 시도
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}

	// EC 형식 시도
	if key, err := x509.ParseECPrivateKey(block.Bytes); err == nil {
		return key, nil
	}

	return nil, errors.Errorf("failed to parse private key from %s", keyPath)
}

// loadCertificatesFromDir 디렉토리에서 모든 인증서 로드
func loadCertificatesFromDir(dirPath string) ([]*x509.Certificate, error) {
	files, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read directory %s", dirPath)
	}

	var certs []*x509.Certificate
	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".pem" {
			continue
		}

		certPath := filepath.Join(dirPath, file.Name())
		cert, err := loadCertificateFromFile(certPath)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to load certificate from %s", certPath)
		}

		certs = append(certs, cert)
	}

	if len(certs) == 0 {
		return nil, errors.Errorf("no certificates found in directory %s", dirPath)
	}

	return certs, nil
}

// checkPathExists 경로 존재 여부 확인
func checkPathExists(path string, isDir bool) error {
	stat, err := os.Stat(path)
	if err != nil {
		return errors.Errorf("path does not exist or not accessible: %s", path)
	}

	if isDir && !stat.IsDir() {
		return errors.Errorf("expected directory but found file: %s", path)
	}

	if !isDir && stat.IsDir() {
		return errors.Errorf("expected file but found directory: %s", path)
	}

	// Check if directory is empty (only for directories)
	if isDir {
		entries, err := os.ReadDir(path)
		if err != nil {
			return errors.Errorf("cannot read directory: %s", path)
		}
		if len(entries) == 0 {
			return errors.Errorf("directory is empty: %s", path)
		}
	}

	return nil
}
