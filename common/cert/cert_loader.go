package cert

import (
	"crypto"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

func LoadCaCertFromDir(dirPath string) (*x509.Certificate, error) {
	return LoadCertFromDir(dirPath, "cacerts")
}

func LoadSignCertFromDir(dirPath string) (*x509.Certificate, error) {
	return LoadCertFromDir(dirPath, "signcerts")
}

func LoadCertFromDir(dirPath, certType string) (*x509.Certificate, error) {
	data, err := LoadSingleFileFromDir(dirPath + "/" + certType)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.Errorf("failed to decode PEM block from directory %s", dirPath)
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse certificate from directory %s", dirPath)
	}

	return cert, nil
}

func LoadPrivateKeyFromDir(dirPath string) (crypto.PrivateKey, error) {
	key, err := LoadSingleFileFromDir(dirPath + "/keystore")
	if err != nil {
		return nil, err
	}

	return LoadPrivateKeyFromFile(key)
}

func LoadPrivateKeyFromFile(key []byte) (crypto.PrivateKey, error) {

	block, _ := pem.Decode(key)
	if block == nil {
		return nil, errors.Errorf("failed to decode PEM block from %s", key)
	}

	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		return key, nil
	}

	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}

	if key, err := x509.ParseECPrivateKey(block.Bytes); err == nil {
		return key, nil
	}

	return nil, errors.Errorf("failed to parse private key from %s", key)
}

func LoadSingleFileFromDir(dirPath string) ([]byte, error) {
	files, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read directory %s", dirPath)
	}

	var fileName string
	for _, file := range files {
		if !file.IsDir() {
			fileName = file.Name()
			break
		}
	}

	if fileName == "" {
		return nil, errors.Errorf("no files found in directory %s", dirPath)
	}

	filePath := filepath.Join(dirPath, fileName)

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read file %s", filePath)
	}

	return data, nil
}
