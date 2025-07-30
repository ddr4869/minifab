#!/bin/bash

# proto 파일 컴파일 스크립트

echo "Compiling proto files..."

# common.proto 컴파일
protoc --go_out=. --go_opt=paths=source_relative \
       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
       proto/common/common.proto

# orderer.proto 컴파일
protoc --go_out=. --go_opt=paths=source_relative \
       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
       proto/orderer/orderer.proto

# peer.proto 컴파일
protoc --go_out=. --go_opt=paths=source_relative \
       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
       proto/peer/peer.proto

# configtx.proto 컴파일
protoc --go_out=. --go_opt=paths=source_relative \
       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
       proto/common/configtx.proto

echo "Proto compilation completed!" 