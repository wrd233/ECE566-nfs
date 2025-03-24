// Package client implements the NFS client core functionality
package client

import (
	"context"
	"fmt"
	"time"

	"github.com/example/nfsserver/pkg/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Config contains the NFS client configuration options
type Config struct {
	// ServerAddress is the address of the NFS server (e.g., "localhost:2049")
	ServerAddress string
	
	// Timeout is the default timeout for RPC operations
	Timeout time.Duration
	
	// MaxRetries is the maximum number of retries for operations
	MaxRetries int
	
	// RetryDelay is the initial delay between retries (will be multiplied by backoff factor)
	RetryDelay time.Duration
	
	// BackoffFactor is the multiplier for retry delay after each attempt
	BackoffFactor float64
	
	// MaxCacheSize is the maximum number of entries in the file handle cache
	MaxCacheSize int
	
	// CacheTTL is the time-to-live for cache entries
	CacheTTL time.Duration
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		ServerAddress: "localhost:2049",
		Timeout:       30 * time.Second,
		MaxRetries:    3,
		RetryDelay:    500 * time.Millisecond,
		BackoffFactor: 2.0,
		MaxCacheSize:  1000,
		CacheTTL:      5 * time.Minute,
	}
}

// Client represents an NFS client
type Client struct {
	// gRPC connection to the server
	conn *grpc.ClientConn
	
	// NFS service client
	nfsClient api.NFSServiceClient
	
	// Client configuration
	config *Config
	
	// File handle cache
	handleCache *HandleCache
	
	// TODO: Add attribute cache when implemented
	// attrCache *AttrCache
}

// NewClient creates a new NFS client
func NewClient(config *Config) (*Client, error) {
	if config == nil {
		config = DefaultConfig()
	}
	
	// Create gRPC connection
	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()
	
	conn, err := grpc.DialContext(
		ctx,
		config.ServerAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to server: %w", err)
	}
	
	// Create NFS service client
	nfsClient := api.NewNFSServiceClient(conn)
	
	// Create handle cache stub
	// TODO: Implement proper handle cache
	handleCache := NewHandleCache(config.MaxCacheSize, config.CacheTTL)
	
	// Create and return the client
	return &Client{
		conn:        conn,
		nfsClient:   nfsClient,
		config:      config,
		handleCache: handleCache,
	}, nil
}

// Close closes the client connection
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// GetRootFileHandle retrieves the root directory file handle from the server
func (c *Client) GetRootFileHandle() ([]byte, error) {
	// TODO: Implement this method
	// This would typically involve a LOOKUP operation on "/" or a special
	// operation to get the root handle, depending on the server implementation
	
	return nil, fmt.Errorf("not implemented")
}

// TODO: Add more client methods that map to NFS operations