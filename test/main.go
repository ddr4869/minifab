package main

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"

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

// LoadSingleCertificateFromDir 디렉토리에서 첫 번째 인증서 파일을 찾아서 로드
func LoadSingleCertificateFromDir(dirPath string) (*x509.Certificate, error) {
	data, err := LoadSingleFileFromDir(dirPath)
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

func main() {
	cacert, err := LoadSingleCertificateFromDir("/Users/mac/go/src/github.com/ddr4869/minifab/test/cacerts")
	if err != nil {
		log.Fatal(err)
	}

	signcert, err := LoadSingleCertificateFromDir("/Users/mac/go/src/github.com/ddr4869/minifab/test/signcerts")
	if err != nil {
		log.Fatal(err)
	}

	// cacerts가 signcert의 root CA 인증서인지 확인
	isRootCA := verifyRootCA(cacert, signcert)
	if isRootCA {
		fmt.Println("✅ CA 인증서가 서명 인증서의 루트 CA입니다.")
	} else {
		fmt.Println("❌ CA 인증서가 서명 인증서의 루트 CA가 아닙니다.")
	}

	if signcert.IsCA {
		fmt.Println("✅ CA 인증서가 CA 용도로 설정되었습니다.")
	} else {
		fmt.Println("❌ CA 인증서가 CA 용도로 설정되지 않았습니다.")
	}

}

// verifyRootCA CA 인증서가 서명 인증서의 루트 CA인지 확인
func verifyRootCA(caCert, signCert *x509.Certificate) bool {
	// 1. CA 인증서가 자체 서명되었는지 확인 (루트 CA의 특징)
	if !isSelfSigned(caCert) {
		fmt.Println("CA 인증서가 자체 서명되지 않았습니다.")
		return false
	}

	// 2. CA 인증서의 공개키로 서명 인증서를 검증
	err := signCert.CheckSignatureFrom(caCert)
	if err != nil {
		fmt.Printf("서명 인증서 검증 실패: %v\n", err)
		return false
	}

	// 3. CA 인증서의 Subject와 Issuer가 일치하는지 확인
	if !reflect.DeepEqual(caCert.Subject, caCert.Issuer) {
		fmt.Println("CA 인증서의 Subject와 Issuer가 일치하지 않습니다.")
		return false
	}

	// 4. CA 인증서가 CA 용도로 설정되었는지 확인
	if !caCert.IsCA {
		fmt.Println("CA 인증서가 CA 용도로 설정되지 않았습니다.")
		return false
	}

	return true
}

// isSelfSigned 인증서가 자체 서명되었는지 확인
func isSelfSigned(cert *x509.Certificate) bool {
	return reflect.DeepEqual(cert.Subject, cert.Issuer)
}
