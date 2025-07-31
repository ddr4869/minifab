package msp

import (
	"crypto/x509"
	"os"
	"path/filepath"

	"github.com/ddr4869/minifab/common/cert"
	"github.com/pkg/errors"
)

func LoadMSPFromFiles(mspID, mspPath string) (MSP, error) {
	if err := ValidateMSPStructure(mspPath); err != nil {
		return nil, errors.Wrap(err, "MSP structure validation failed")
	}

	msp, err := LoadMSP(mspID, mspPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load identity")
	}

	return msp, nil
}

func LoadMSP(mspID, mspPath string) (MSP, error) {
	x509Cert, err := cert.LoadSignCertFromDir(mspPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load sign cert")
	}

	identity := NewIdentity(x509Cert, x509Cert.PublicKey, mspID)

	privateKey, err := cert.LoadPrivateKeyFromDir(mspPath)
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
	caCerts, err := cert.LoadCaCertFromDir(mspPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load CA certs")
	}

	msp := NewFabricMSP()
	mspConfig := &MSPConfig{
		MSPID:           mspID,
		SigningIdentity: &identity,
		RootCerts:       []*x509.Certificate{caCerts},
	}

	if err := msp.Setup(mspConfig); err != nil {
		return nil, errors.Wrap(err, "failed to setup MSP")
	}

	return msp, nil
}

func ValidateMSPStructure(mspPath string) error {
	requiredDirs := []string{"signcerts", "keystore", "cacerts"}

	for _, dir := range requiredDirs {
		dirPath := filepath.Join(mspPath, dir)
		if err := checkPathExists(dirPath, true); err != nil {
			return errors.Errorf("required directory missing: %s", dir)
		}
	}
	return nil
}

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
