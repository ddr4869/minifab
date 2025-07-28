#!/bin/bash

# 에러 발생시 스크립트 중단
set -e

echo "=== Minifab Node Build Script ==="
echo "Building orderer and peer binaries..."

# 프로젝트 루트 디렉토리로 이동
cd "$(dirname "$0")"

# bin 디렉토리 생성 (이미 존재하면 무시)
mkdir -p bin

# Go 모듈 체크
echo "Checking Go modules..."
if [ ! -f "go.mod" ]; then
    echo "Error: go.mod not found. Please run 'go mod init' first."
    exit 1
fi

# 의존성 다운로드
echo "Downloading dependencies..."
go mod download
go mod tidy

# Orderer 빌드
echo "Building orderer..."
go build -o bin/orderer ./cmd/orderer
if [ $? -eq 0 ]; then
    echo "✓ Orderer binary built successfully: bin/orderer"
else
    echo "✗ Failed to build orderer"
    exit 1
fi

# Peer 빌드
echo "Building peer..."
go build -o bin/peer ./cmd/peer
if [ $? -eq 0 ]; then
    echo "✓ Peer binary built successfully: bin/peer"
else
    echo "✗ Failed to build peer"
    exit 1
fi

# 빌드된 바이너리 정보 출력
echo ""
echo "=== Build Summary ==="
echo "Built binaries:"
ls -la bin/orderer bin/peer 2>/dev/null || echo "Warning: Some binaries not found"

# 실행 권한 부여
chmod +x bin/orderer bin/peer

echo ""
echo "✓ Build completed successfully!"
echo "Usage:"
echo "  ./bin/orderer --help"
echo "  ./bin/peer --help"
