.PHONY: proto build test clean run-server run-client

# Define directories
BIN_DIR := bin
PROTO_DIR := proto

# Compiles protobuf files
proto:
	@echo "Generating protobuf code..."
	mkdir -p pkg/api
	protoc -I. \
		--go_out=pkg/api --go_opt=module=github.com/example/nfsserver/pkg/api \
		--go-grpc_out=pkg/api --go-grpc_opt=module=github.com/example/nfsserver/pkg/api \
		proto/*.proto

# Build server and client
build: proto
	@echo "Building NFS server and client..."
	mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/nfsserver cmd/server/main.go
	go build -o $(BIN_DIR)/nfsclient cmd/client/main.go
	go build -o $(BIN_DIR)/gethandle cmd/tools/gethandle.go

# Run server
run-server: build
	@echo "Starting NFS server..."
	$(BIN_DIR)/nfsserver

# Run client
run-client: build
	@echo "Running NFS client..."
	$(BIN_DIR)/nfsclient

# Get file handle
get-handle: build
	@echo "Getting file handle..."
	$(BIN_DIR)/gethandle

# Run tests
test:
	@echo "Running tests..."
	go test ./...

# Clean generated files
clean:
	@echo "Cleaning up..."
	rm -rf $(BIN_DIR)
	rm -f pkg/api/*.pb.go
