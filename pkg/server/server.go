// Package server implements the NFS server functionality
package server

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
	"path/filepath"

	"github.com/example/nfsserver/pkg/api"
	"github.com/example/nfsserver/pkg/fs"
	"github.com/example/nfsserver/pkg/nfs"
	"google.golang.org/grpc"
)

// Config contains the NFS server configuration
type Config struct {
	// Network address to listen on (e.g. ":2049")
	ListenAddress string

	// Maximum concurrent requests
	MaxConcurrent int

	// Maximum read size in bytes
	MaxReadSize int

	// Maximum write size in bytes
	MaxWriteSize int

	// Request timeout in seconds
	RequestTimeout int

	// Enable root squashing (map root to anonymous user)
	EnableRootSquash bool

	// Anonymous user ID
	AnonUID uint32

	// Anonymous group ID
	AnonGID uint32
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		ListenAddress:    ":2049",
		MaxConcurrent:    100,
		MaxReadSize:      1024 * 1024, // 1MB
		MaxWriteSize:     1024 * 1024, // 1MB
		RequestTimeout:   30,          // 30 seconds
		EnableRootSquash: true,
		AnonUID:          65534, // nobody
		AnonGID:          65534, // nogroup
	}
}

// NFSServer implements the NFS service
type NFSServer struct {
	api.UnimplementedNFSServiceServer

	// Configuration
	config *Config

	// The underlying filesystem implementation
	fileSystem fs.FileSystem

	// Secret key for file handle signatures
	handleKey []byte

	// Request cache for idempotent operations
	reqCache     map[string]interface{}
	reqCacheMu   sync.RWMutex
	reqCacheTTL  time.Duration

	// Worker pool for limiting concurrent requests
	workerPool chan struct{}
}

// NewNFSServer creates a new NFS server
func NewNFSServer(config *Config, fileSystem fs.FileSystem) (*NFSServer, error) {
	// Generate random key for file handle signatures
	handleKey := make([]byte, 32)
	if _, err := rand.Read(handleKey); err != nil {
		return nil, fmt.Errorf("failed to generate handle key: %w", err)
	}

	// Create worker pool for controlling concurrency
	workerPool := make(chan struct{}, config.MaxConcurrent)

	return &NFSServer{
		config:      config,
		fileSystem:  fileSystem,
		handleKey:   handleKey,
		reqCache:    make(map[string]interface{}),
		reqCacheTTL: time.Duration(2) * time.Minute,
		workerPool:  workerPool,
	}, nil
}

// Start launches the NFS server
func (s *NFSServer) Start() error {
	// Create listener
	lis, err := net.Listen("tcp", s.config.ListenAddress)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	// Create gRPC server
	grpcServer := grpc.NewServer()
	
	// Register NFS service
	api.RegisterNFSServiceServer(grpcServer, s)

	// Start serving
	log.Printf("NFS server starting on %s", s.config.ListenAddress)
	if err := grpcServer.Serve(lis); err != nil {
		return fmt.Errorf("failed to serve: %w", err)
	}
	
	return nil
}

// acquireWorker gets a worker from the pool or times out
func (s *NFSServer) acquireWorker(ctx context.Context) error {
	select {
	case s.workerPool <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// releaseWorker returns a worker to the pool
func (s *NFSServer) releaseWorker() {
	<-s.workerPool
}

// processRequest handles common request processing logic
func (s *NFSServer) processRequest(ctx context.Context, op string, reqID string, clientAddr string, 
	process func() (interface{}, error)) (interface{}, error) {
	
	// Log request
	nfs.LogRequest(op, reqID, clientAddr)
	startTime := time.Now()
	
	// Acquire worker
	if err := s.acquireWorker(ctx); err != nil {
		nfs.LogError(op, reqID, err)
		return nil, err
	}
	defer s.releaseWorker()

	// Execute the operation
	result, err := process()
	
	// Log the result
	duration := time.Since(startTime)
	var status api.Status
	if err != nil {
		nfs.LogError(op, reqID, err)
		status = nfs.MapErrorToStatus(err)
	} else {
		status = api.Status_OK
	}
	
	nfs.LogResponse(op, reqID, status, duration.String())
	return result, err
}

// validateFileHandle verifies a file handle is valid
func (s *NFSServer) validateFileHandle(handle []byte) ([]byte, error) {
	log.Printf("Validating handle: %x (length: %d)", handle, len(handle))
	if len(handle) < 16 {
		return nil, nfs.NewNFSError(api.Status_ERR_BADHANDLE, "handle too short", nil)
	}
	
	// In a real implementation, we would verify the handle signature here
	// For now, we just return the handle
	return handle, nil
}

// GetAttr implements the GetAttr RPC method
func (s *NFSServer) GetAttr(ctx context.Context, req *api.GetAttrRequest) (*api.GetAttrResponse, error) {
	// Create a unique request ID and get client address
	reqID := fmt.Sprintf("getattr-%d", time.Now().UnixNano())
	clientAddr := "unknown"
	if peer, ok := ctx.Value("peer").(*net.Addr); ok && peer != nil {
		clientAddr = (*peer).String()
	}
	
	// Process the request
	result, err := s.processRequest(ctx, "GetAttr", reqID, clientAddr, func() (interface{}, error) {
		// Validate file handle
		_, err := s.validateFileHandle(req.FileHandle)
		if err != nil {
			return &api.GetAttrResponse{Status: api.Status_ERR_BADHANDLE}, nil
		}
		
		// Convert file handle to path
		path, err := s.fileSystem.FileHandleToPath(req.FileHandle)
		if err != nil {
			return &api.GetAttrResponse{Status: nfs.MapErrorToStatus(err)}, nil
		}
		
		// Get credentials
		creds := nfs.ProtoCredsToFSCreds(req.Credentials)
		
		// Apply root squashing if enabled
		if s.config.EnableRootSquash && creds.UID == 0 {
			creds.UID = s.config.AnonUID
			creds.GID = s.config.AnonGID
		}
		
		// Check read permission
		if err := s.fileSystem.Access(ctx, path, fs.FileMode(4), creds); err != nil {
			return &api.GetAttrResponse{Status: nfs.MapErrorToStatus(err)}, nil
		}
		
		// Get file attributes
		fileInfo, err := s.fileSystem.GetAttr(ctx, path)
		if err != nil {
			return &api.GetAttrResponse{Status: nfs.MapErrorToStatus(err)}, nil
		}
		
		// Convert to NFS attributes
		attrs := nfs.FSInfoToProtoAttributes(fileInfo)
		
		// Return successful response
		response := &api.GetAttrResponse{
			Status:     api.Status_OK,
			Attributes: attrs,
		}
		
		log.Printf("Sending response: %+v", response)
		return response, nil
	})
	
	if err != nil {
		return nil, err
	}
	
	return result.(*api.GetAttrResponse), nil
}

// Lookup implements the Lookup RPC method
func (s *NFSServer) Lookup(ctx context.Context, req *api.LookupRequest) (*api.LookupResponse, error) {
    // Create a unique request ID and get client address
    reqID := fmt.Sprintf("lookup-%d", time.Now().UnixNano())
    clientAddr := "unknown"
    if peer, ok := ctx.Value("peer").(*net.Addr); ok && peer != nil {
        clientAddr = (*peer).String()
    }
    
    // Process the request
    result, err := s.processRequest(ctx, "Lookup", reqID, clientAddr, func() (interface{}, error) {
        // Validate directory handle
        _, err := s.validateFileHandle(req.DirectoryHandle)
        if err != nil {
            return &api.LookupResponse{Status: api.Status_ERR_BADHANDLE}, nil
        }
        
        // Convert directory handle to path
        dirPath, err := s.fileSystem.FileHandleToPath(req.DirectoryHandle)
        if err != nil {
            return &api.LookupResponse{Status: nfs.MapErrorToStatus(err)}, nil
        }
        
        // Get credentials
        creds := nfs.ProtoCredsToFSCreds(req.Credentials)
        
        // Apply root squashing if enabled
        if s.config.EnableRootSquash && creds.UID == 0 {
            creds.UID = s.config.AnonUID
            creds.GID = s.config.AnonGID
        }
        
        // Check directory access permission
        if err := s.fileSystem.Access(ctx, dirPath, fs.FileMode(5), creds); err != nil { // 5 = read + execute
            return &api.LookupResponse{Status: nfs.MapErrorToStatus(err)}, nil
        }
        
        // Handle special directory entries
        targetPath := ""
        switch req.Name {
        case ".":
            targetPath = dirPath
        case "..":
            // For parent directory, we need to find the actual parent
            if dirPath == "/" {
                // Root directory has itself as parent
                targetPath = "/"
            } else {
                // Get parent directory path
                targetPath = filepath.Dir(dirPath)
                if targetPath == "." {
                    targetPath = "/"
                }
            }
        default:
            // Look up the file in the directory
            targetPath, _ , err = s.fileSystem.Lookup(ctx, dirPath, req.Name)
            if err != nil {
                return &api.LookupResponse{Status: nfs.MapErrorToStatus(err)}, nil
            }
        }
        
        // Get file attributes
        fileInfo, err := s.fileSystem.GetAttr(ctx, targetPath)
        if err != nil {
            return &api.LookupResponse{Status: nfs.MapErrorToStatus(err)}, nil
        }
        
        // Get directory attributes if needed (optional)
        var dirAttrs *api.FileAttributes
        if dirPath != targetPath { // Not looking up "."
            dirInfo, err := s.fileSystem.GetAttr(ctx, dirPath)
            if err == nil {
                dirAttrs = nfs.FSInfoToProtoAttributes(dirInfo)
            }
        }
        
        // Generate file handle for the target
        fileHandle, err := s.fileSystem.PathToFileHandle(targetPath)
        if err != nil {
            return &api.LookupResponse{Status: nfs.MapErrorToStatus(err)}, nil
        }
        
        // Convert file info to NFS attributes
        attrs := nfs.FSInfoToProtoAttributes(fileInfo)
        
        // Return successful response
        return &api.LookupResponse{
            Status:        api.Status_OK,
            FileHandle:    fileHandle,
            Attributes:    attrs,
            DirAttributes: dirAttrs,
        }, nil
    })
    
    if err != nil {
        return nil, err
    }
    
    return result.(*api.LookupResponse), nil
}

// Read implements the Read RPC method
func (s *NFSServer) Read(ctx context.Context, req *api.ReadRequest) (*api.ReadResponse, error) {
    // Create a unique request ID and get client address
    reqID := fmt.Sprintf("read-%d", time.Now().UnixNano())
    clientAddr := "unknown"
    if peer, ok := ctx.Value("peer").(*net.Addr); ok && peer != nil {
        clientAddr = (*peer).String()
    }
    
    // Process the request
    result, err := s.processRequest(ctx, "Read", reqID, clientAddr, func() (interface{}, error) {
        // Validate file handle
        _, err := s.validateFileHandle(req.FileHandle)
        if err != nil {
            return &api.ReadResponse{Status: api.Status_ERR_BADHANDLE}, nil
        }
        
        // Convert file handle to path
        path, err := s.fileSystem.FileHandleToPath(req.FileHandle)
        if err != nil {
            return &api.ReadResponse{Status: nfs.MapErrorToStatus(err)}, nil
        }
        
        // Get credentials
        creds := nfs.ProtoCredsToFSCreds(req.Credentials)
        
        // Apply root squashing if enabled
        if s.config.EnableRootSquash && creds.UID == 0 {
            creds.UID = s.config.AnonUID
            creds.GID = s.config.AnonGID
        }
        
        // Check read permission
        if err := s.fileSystem.Access(ctx, path, fs.FileMode(4), creds); err != nil { // 4 = read
            return &api.ReadResponse{Status: nfs.MapErrorToStatus(err)}, nil
        }
        
        // Get file attributes
        fileInfo, err := s.fileSystem.GetAttr(ctx, path)
        if err != nil {
            return &api.ReadResponse{Status: nfs.MapErrorToStatus(err)}, nil
        }
        
        // Check if it's a regular file (not a directory)
        if fileInfo.Type != fs.FileTypeRegular {
            return &api.ReadResponse{Status: api.Status_ERR_ISDIR}, nil
        }
        
        // Limit read size for security
        count := req.Count
        if count > uint32(s.config.MaxReadSize) {
            count = uint32(s.config.MaxReadSize)
        }
        
        // Read data from file
        data, eof, err := s.fileSystem.Read(ctx, path, int64(req.Offset), int(count))
        if err != nil {
            return &api.ReadResponse{Status: nfs.MapErrorToStatus(err)}, nil
        }
        
        // Update file attributes after read
        newFileInfo, _ := s.fileSystem.GetAttr(ctx, path)
        attrs := nfs.FSInfoToProtoAttributes(newFileInfo)
        
        // Return successful response
        return &api.ReadResponse{
            Status:     api.Status_OK,
            Data:       data,
            Eof:        eof,
            Attributes: attrs,
        }, nil
    })
    
    if err != nil {
        return nil, err
    }
    
    return result.(*api.ReadResponse), nil
}

// Write implements the Write RPC method
func (s *NFSServer) Write(ctx context.Context, req *api.WriteRequest) (*api.WriteResponse, error) {
	// This is a placeholder implementation
	// It will be implemented in a future step
	return &api.WriteResponse{
		Status: api.Status_ERR_NOTSUPP,
	}, nil
}