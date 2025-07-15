package msp

import (
	"crypto"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
)

// MSPFileLoader MSP 파일들을 로드하는 구조체
type MSPFileLoader struct {
	mspPath string
}

// NewMSPFileLoader MSP 파일 로더 생성
func NewMSPFileLoader(mspPath string) *MSPFileLoader {
	return &MSPFileLoader{
		mspPath: mspPath,
	}
}

// LoadIdentityFromFiles MSP 디렉토리에서 Identity 로드
func (loader *MSPFileLoader) LoadIdentityFromFiles(mspID string) (Identity, crypto.PrivateKey, error) {
	// 1. 인증서 로드
	cert, err := loader.LoadSignCert()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load sign cert: %v", err)
	}

	// 2. 개인키 로드
	privateKey, err := loader.LoadPrivateKey()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load private key: %v", err)
	}

	// 3. CA 인증서 로드
	caCerts, err := loader.LoadCACerts()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load CA certs: %v", err)
	}

	// 4. MSP 생성 및 설정
	msp := NewFabricMSP()
	mspConfig := &MSPConfig{
		Name:         mspID,
		RootCerts:    caCerts,
		TLSRootCerts: caCerts,
	}

	if err := msp.Setup(mspConfig); err != nil {
		return nil, nil, fmt.Errorf("failed to setup MSP: %v", err)
	}

	// 5. Identity 생성
	identity := NewIdentity(msp, cert, cert.PublicKey, mspID)

	// 6. Identity 검증
	if err := identity.Validate(); err != nil {
		return nil, nil, fmt.Errorf("loaded identity is invalid: %v", err)
	}

	return identity, privateKey, nil
}

// LoadSignCert signcerts 디렉토리에서 서명 인증서 로드
func (loader *MSPFileLoader) LoadSignCert() (*x509.Certificate, error) {
	certPath := filepath.Join(loader.mspPath, "signcerts", "cert.pem")
	return loader.loadCertificateFromFile(certPath)
}

// LoadPrivateKey keystore 디렉토리에서 개인키 로드
func (loader *MSPFileLoader) LoadPrivateKey() (crypto.PrivateKey, error) {
	keyPath := filepath.Join(loader.mspPath, "keystore", "key.pem")
	return loader.loadPrivateKeyFromFile(keyPath)
}

// LoadCACerts cacerts 디렉토리에서 CA 인증서들 로드
func (loader *MSPFileLoader) LoadCACerts() ([]*x509.Certificate, error) {
	caCertsDir := filepath.Join(loader.mspPath, "cacerts")
	return loader.loadCertificatesFromDir(caCertsDir)
}

// LoadTLSCACerts tlscacerts 디렉토리에서 TLS CA 인증서들 로드
func (loader *MSPFileLoader) LoadTLSCACerts() ([]*x509.Certificate, error) {
	tlsCaCertsDir := filepath.Join(loader.mspPath, "tlscacerts")
	return loader.loadCertificatesFromDir(tlsCaCertsDir)
}

// loadCertificateFromFile 파일에서 단일 인증서 로드
func (loader *MSPFileLoader) loadCertificateFromFile(certPath string) (*x509.Certificate, error) {
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read certificate file %s: %v", certPath, err)
	}

	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block from %s", certPath)
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate from %s: %v", certPath, err)
	}

	return cert, nil
}

// loadPrivateKeyFromFile 파일에서 개인키 로드
func (loader *MSPFileLoader) loadPrivateKeyFromFile(keyPath string) (crypto.PrivateKey, error) {
	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key file %s: %v", keyPath, err)
	}

	block, _ := pem.Decode(keyPEM)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block from %s", keyPath)
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

	return nil, fmt.Errorf("failed to parse private key from %s", keyPath)
}

// loadCertificatesFromDir 디렉토리에서 모든 인증서 로드
func (loader *MSPFileLoader) loadCertificatesFromDir(dirPath string) ([]*x509.Certificate, error) {
	files, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %v", dirPath, err)
	}

	var certs []*x509.Certificate
	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".pem" {
			continue
		}

		certPath := filepath.Join(dirPath, file.Name())
		cert, err := loader.loadCertificateFromFile(certPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load certificate from %s: %v", certPath, err)
		}

		certs = append(certs, cert)
	}

	if len(certs) == 0 {
		return nil, fmt.Errorf("no certificates found in directory %s", dirPath)
	}

	return certs, nil
}

// ValidateMSPStructure MSP 디렉토리 구조 검증
func (loader *MSPFileLoader) ValidateMSPStructure() error {
	requiredDirs := []string{"signcerts", "keystore", "cacerts"}
	requiredFiles := []string{
		"signcerts/cert.pem",
		"keystore/key.pem",
	}

	// 필수 디렉토리 확인
	for _, dir := range requiredDirs {
		dirPath := filepath.Join(loader.mspPath, dir)
		if err := loader.checkPathExists(dirPath, true); err != nil {
			return fmt.Errorf("required directory missing: %s", dir)
		}
	}

	// 필수 파일 확인
	for _, file := range requiredFiles {
		filePath := filepath.Join(loader.mspPath, file)
		if err := loader.checkPathExists(filePath, false); err != nil {
			return fmt.Errorf("required file missing: %s", file)
		}
	}

	return nil
}

// checkPathExists 경로 존재 여부 확인
func (loader *MSPFileLoader) checkPathExists(path string, isDir bool) error {
	stat, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("path does not exist or not accessible: %s", path)
	}

	if isDir && !stat.IsDir() {
		return fmt.Errorf("expected directory but found file: %s", path)
	}

	if !isDir && stat.IsDir() {
		return fmt.Errorf("expected file but found directory: %s", path)
	}

	// Check if directory is empty (only for directories)
	if isDir {
		entries, err := os.ReadDir(path)
		if err != nil {
			return fmt.Errorf("cannot read directory: %s", path)
		}
		if len(entries) == 0 {
			return fmt.Errorf("directory is empty: %s", path)
		}
	}

	return nil
}

// CreateMSPFromFiles fabric-ca로 생성된 MSP 파일들로부터 MSP 생성 (헬퍼 함수)
func CreateMSPFromFiles(mspPath, mspID string) (*FabricMSP, Identity, crypto.PrivateKey, error) {
	loader := NewMSPFileLoader(mspPath)

	// MSP 구조 검증
	if err := loader.ValidateMSPStructure(); err != nil {
		return nil, nil, nil, fmt.Errorf("MSP structure validation failed: %v", err)
	}

	// Identity 로드
	identity, privateKey, err := loader.LoadIdentityFromFiles(mspID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to load identity: %v", err)
	}

	// MSP는 identity를 생성할 때 사용된 msp를 다시 생성해서 반환
	// (실제로는 이미 LoadIdentityFromFiles에서 생성된 msp를 재사용)
	loader2 := NewMSPFileLoader(mspPath)
	caCerts, err := loader2.LoadCACerts()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to reload CA certs: %v", err)
	}
	msp := NewFabricMSP()
	mspConfig := &MSPConfig{
		Name:         mspID,
		RootCerts:    caCerts,
		TLSRootCerts: caCerts,
	}

	if err := msp.Setup(mspConfig); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to setup MSP: %v", err)
	}

	return msp, identity, privateKey, nil
}
