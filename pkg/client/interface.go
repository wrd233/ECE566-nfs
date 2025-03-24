// Package client implements the NFS client core functionality
package client

import (
	"context"
	"time"

	"github.com/example/nfsserver/pkg/api"
)

// NFSClient defines the interface for NFS client operations
type NFSClient interface {
    // File attribute and lookup operations
    
    // GetAttr retrieves attributes for a file or directory
    GetAttr(ctx context.Context, fileHandle []byte) (*api.FileAttributes, error)
    
    // Lookup looks up a file name in a directory
    // Returns the file handle, attributes, and any error
    Lookup(ctx context.Context, dirHandle []byte, name string) ([]byte, *api.FileAttributes, error)
    
    // Read and write operations
    
    // Read reads data from a file at the specified offset
    // Returns the data read, a boolean indicating if EOF was reached, and any error
    Read(ctx context.Context, fileHandle []byte, offset int64, count int) ([]byte, bool, error)
    
    // Write writes data to a file at the specified offset
    // stability levels: 0=UNSTABLE, 1=DATA_SYNC, 2=FILE_SYNC
    // Returns the number of bytes written and any error
    Write(ctx context.Context, fileHandle []byte, offset int64, data []byte, stability int) (int, error)
    
    // Directory operations
    
    // ReadDir reads the contents of a directory
    // Returns directory entries and any error
    ReadDir(ctx context.Context, dirHandle []byte) ([]*api.DirEntry, error)
    
    // File system modification operations
    
    // Create creates a new file in the specified directory
    // Returns the file handle, attributes, and any error
    Create(ctx context.Context, dirHandle []byte, name string, attrs *api.FileAttributes, mode api.CreateMode) ([]byte, *api.FileAttributes, error)
    
    // Mkdir creates a new directory
    // Returns the directory handle, attributes, and any error
    Mkdir(ctx context.Context, dirHandle []byte, name string, attrs *api.FileAttributes) ([]byte, *api.FileAttributes, error)
    
    // Remove removes a file from the specified directory
    Remove(ctx context.Context, dirHandle []byte, name string) error
    
    // Rmdir removes a directory
    Rmdir(ctx context.Context, dirHandle []byte, name string) error
    
    // Rename renames a file or directory
    Rename(ctx context.Context, fromDirHandle []byte, fromName string, toDirHandle []byte, toName string) error
    
    // Resource management
    
    // Close closes the client connection and releases all resources
    Close() error
    
    // Path operations
    
    // GetRootFileHandle retrieves the root directory file handle from the server
    GetRootFileHandle(ctx context.Context) ([]byte, error)
    
    // LookupPath resolves a file path to a file handle, starting from the root
    LookupPath(ctx context.Context, path string) ([]byte, error)
}


type ExtendedNFSClient interface {
    NFSClient
    // TODO 未来可能添加其他高级方法
}

// CacheableClient extends NFSClient with cache management capabilities
type CacheableClient interface {
	NFSClient
	
	// ClearCache clears all cached handles and attributes
	ClearCache() error
	
	// SetCacheTTL sets the time-to-live for cache entries
	SetCacheTTL(duration time.Duration)
}

// StatisticsClient extends NFSClient with statistics reporting
type StatisticsClient interface {
	NFSClient
	
	// GetStatistics returns client operation statistics
	GetStatistics() ClientStats
}

// ClientStats contains statistics about client operations
type ClientStats struct {
	Operations      uint64        // Total number of operations performed
	Errors          uint64        // Number of operations that resulted in errors
	BytesRead       uint64        // Total bytes read
	BytesWritten    uint64        // Total bytes written
	AvgResponseTime time.Duration // Average operation response time
}