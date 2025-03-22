package main

import (
	"context"
	"log"
	"net"

	"github.com/example/nfsserver/proto"
	"google.golang.org/grpc"
)

// Server implementation
type server struct {
	api.UnimplementedGreeterServer
}

// SayHello implementation
func (s *server) SayHello(ctx context.Context, req *api.HelloRequest) (*api.HelloReply, error) {
	log.Printf("Received request: %v", req.GetName())
	return &api.HelloReply{Message: "Hello, " + req.GetName()}, nil
}

func main() {
	// Listen on port
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	
	// Create gRPC server
	s := grpc.NewServer()
	api.RegisterGreeterServer(s, &server{})
	
	log.Println("Server started on :50051")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
