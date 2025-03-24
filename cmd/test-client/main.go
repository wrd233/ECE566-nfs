package main

import (
	"fmt"
	"os"
	"time"

	"github.com/example/nfsserver/pkg/client"
)

func main() {
	// Parse command line arguments
	serverAddr := "localhost:2049"
	if len(os.Args) > 1 {
		serverAddr = os.Args[1]
	}

	fmt.Printf("Testing NFS client connection to %s\n", serverAddr)

	// Create client configuration
	config := &client.Config{
		ServerAddress: serverAddr,
		Timeout:       5 * time.Second,
		MaxRetries:    2,
	}

	// Create NFS client
	fmt.Println("Creating NFS client...")
	nfsClient, err := client.NewClient(config)
	if err != nil {
		fmt.Printf("Error creating client: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Successfully created NFS client!")
	fmt.Println("Client configuration:")
	fmt.Printf("- Server: %s\n", config.ServerAddress)
	fmt.Printf("- Timeout: %v\n", config.Timeout)
	fmt.Printf("- Max Retries: %d\n", config.MaxRetries)

	// Clean up
	fmt.Println("Closing client connection...")
	if err := nfsClient.Close(); err != nil {
		fmt.Printf("Error closing connection: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Client connection closed successfully")
	fmt.Println("NFS client module initialization test passed!")
}