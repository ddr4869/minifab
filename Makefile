.PHONY: proto build clean

proto:
	@echo "Compiling proto files..."
	@chmod +x scripts/compile_proto.sh
	@./scripts/compile_proto.sh

build: proto
	@echo "Building orderer..."
	@go build -o bin/orderer cmd/orderer/main.go
	@echo "Building peer..."
	@go build -o bin/peer cmd/peer/main.go

clean:
	@echo "Cleaning build artifacts..."
	@rm ./bin/peer
	@rm ./bin/orderer
	@find . -name "*.pb.go" -delete
	@find . -name "*_grpc.pb.go" -delete

deps:
	@echo "Installing dependencies..."
	@go mod tidy
	@go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	@go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest 