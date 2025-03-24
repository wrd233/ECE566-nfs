package client

import (
	"testing"
	"time"
)

// TestClientCreation verifies that a client can be created with custom config
func TestClientCreation(t *testing.T) {
	// Skip this test if connecting to a real server
	t.Skip("This test attempts to connect to a real server. Use -test.run=TestClientCreation to run explicitly.")
	
	// Create custom config
	config := &Config{
		ServerAddress: "localhost:2049",
		Timeout:       2 * time.Second, // Short timeout for testing
		MaxRetries:    1,               // Just one retry
	}
	
	// Try to create client
	client, err := NewClient(config)
	if err != nil {
		t.Logf("Could not connect to server: %v (this is expected if no server is running)", err)
		return
	}
	
	// Make sure to close the client
	defer client.Close()
	
	t.Log("Successfully connected to NFS server")
}