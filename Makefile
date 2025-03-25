.PHONY: proto build test clean run-server run-client build-fuse run-fuse test-client

# Define directories
BIN_DIR := bin
PROTO_DIR := proto
MOUNT_DIR := /tmp/nfs-mount

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
	go build -o $(BIN_DIR)/gethandle cmd/tools/gethandle.go
	go build -o $(BIN_DIR)/nfs-fuse cmd/nfs-fuse/main.go

# Run server
run-server: build
	@echo "Starting NFS server..."
	$(BIN_DIR)/nfsserver

# Run FUSE client
run-fuse: build
	@echo "Creating mount directory..."
	mkdir -p $(MOUNT_DIR)
	@echo "Starting NFS FUSE client..."
	$(BIN_DIR)/nfs-fuse -mount $(MOUNT_DIR)

# Unmount FUSE filesystem
unmount-fuse:
	@echo "Unmounting FUSE filesystem..."
	fusermount -uz $(MOUNT_DIR) || umount -f $(MOUNT_DIR) || true

# Get file handle
get-handle: build
	@echo "Getting file handle..."
	$(BIN_DIR)/gethandle

# Run tests
test:
	@echo "Running tests..."
	go test ./...

# Clean generated files
clean: unmount-fuse
	@echo "Cleaning up..."
	rm -rf $(BIN_DIR)
	rm -f pkg/api/*.pb.go
	rm -rf $(MOUNT_DIR)