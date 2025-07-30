package cert

import (
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

// LoadSingleFileFromDir 디렉토리에서 첫 번째 파일을 찾아서 로드
func LoadSingleFileFromDir(dirPath string) ([]byte, error) {
	// 디렉토리 읽기
	files, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read directory %s", dirPath)
	}

	// 파일 찾기 (디렉토리가 아닌 파일 중 첫 번째)
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

	// 파일 경로 생성
	filePath := filepath.Join(dirPath, fileName)

	// 파일 읽기
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read file %s", filePath)
	}

	return data, nil
}

// LoadCertFromDir는 cacerts or signcerts 폴더에서 인증서 로드
func LoadCertFromDir(dirPath, certType string) (*x509.Certificate, error) {
	data, err := LoadSingleFileFromDir(dirPath + "/" + certType)
	if err != nil {
		return nil, err
	}

	// PEM 디코딩
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.Errorf("failed to decode PEM block from directory %s", dirPath)
	}

	// 인증서 파싱
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse certificate from directory %s", dirPath)
	}

	return cert, nil
}
