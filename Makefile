.PHONY: proto build run-server run-client test clean

# Compile proto files
proto:
	protoc --go_out=. --go_opt=paths=source_relative 		--go-grpc_out=. --go-grpc_opt=paths=source_relative 		proto/greeter.proto

# Build server and client
build:
	go build -o bin/server cmd/server/main.go
	go build -o bin/client cmd/client/main.go

# Run server
run-server:
	go run cmd/server/main.go

# Run client
run-client:
	go run cmd/client/main.go

# Run tests
test:
	go test ./...

# Clean generated files
clean:
	rm -f proto/*.pb.go
	rm -rf bin/
