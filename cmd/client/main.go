package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/example/nfsserver/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	// Connect to server
	conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()
	
	// Create client
	c := api.NewGreeterClient(conn)
	
	// Prepare name
	name := "world"
	if len(os.Args) > 1 {
		name = os.Args[1]
	}
	
	// Set timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	
	// Send request
	r, err := c.SayHello(ctx, &api.HelloRequest{Name: name})
	if err != nil {
		log.Fatalf("Request failed: %v", err)
	}
	
	log.Printf("Response received: %s", r.GetMessage())
}
