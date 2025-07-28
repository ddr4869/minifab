.PHONY: proto build clean

# proto 파일 컴파일
proto:
	@echo "Compiling proto files..."
	@chmod +x scripts/compile_proto.sh
	@./scripts/compile_proto.sh

# 빌드
build: proto
	@echo "Building orderer..."
	@go build -o bin/orderer cmd/orderer/main.go
	@echo "Building peer..."
	@go build -o bin/peer cmd/peer/main.go

# 정리
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@find . -name "*.pb.go" -delete
	@find . -name "*_grpc.pb.go" -delete

# 의존성 설치
deps:
	@echo "Installing dependencies..."
	@go mod tidy
	@go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	@go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest 