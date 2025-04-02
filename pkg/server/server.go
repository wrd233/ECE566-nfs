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
	"hash/crc32"
    "syscall"
    "encoding/binary"

	"github.com/example/nfsserver/pkg/api"
	"github.com/example/nfsserver/pkg/fs"
	"github.com/example/nfsserver/pkg/nfs"
	"google.golang.org/grpc"
)

// Config contains the NFS server configuration
type Config struct {
	ListenAddress string    // default ":2049"
	MaxConcurrent int
	MaxReadSize int
	MaxWriteSize int
	RequestTimeout int
	EnableRootSquash bool
	AnonUID uint32
	AnonGID uint32
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		ListenAddress:    ":2049",
		MaxConcurrent:    100,
		MaxReadSize:      1024 * 1024, 
		MaxWriteSize:     1024 * 1024, 
		RequestTimeout:   30,          
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

    // Write verifier used for commit
    writeVerifier uint64
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

    // Generate a unique write verifier
    var verifier uint64
    binary.Read(rand.Reader, binary.BigEndian, &verifier)
    
    // If random source fails, use time and process ID
    if verifier == 0 {
        verifier = uint64(time.Now().UnixNano())
    }

	return &NFSServer{
		config:      config,
		fileSystem:  fileSystem,
		handleKey:   handleKey,
		reqCache:    make(map[string]interface{}),
		reqCacheTTL: time.Duration(2) * time.Minute,
		workerPool:  workerPool,
        writeVerifier: verifier,
	}, nil
}

// Start launches the NFS server
func (s *NFSServer) Start() error {
    // 设置umask为0，允许创建具有完整权限的文件
    oldUmask := syscall.Umask(0)
    defer syscall.Umask(oldUmask) // 在服务器关闭时恢复

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
	
	// TODO: For now, we just return the handle
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
	
	result, err := s.processRequest(ctx, "GetAttr", reqID, clientAddr, func() (interface{}, error) {
		_, err := s.validateFileHandle(req.FileHandle)
		if err != nil {
			return &api.GetAttrResponse{Status: api.Status_ERR_BADHANDLE}, nil
		}
		
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
		
		if err := s.fileSystem.Access(ctx, path, fs.FileMode(4), creds); err != nil {
			return &api.GetAttrResponse{Status: nfs.MapErrorToStatus(err)}, nil
		}
		
		fileInfo, err := s.fileSystem.GetAttr(ctx, path)
		if err != nil {
			return &api.GetAttrResponse{Status: nfs.MapErrorToStatus(err)}, nil
		}
		
		attrs := nfs.FSInfoToProtoAttributes(fileInfo)
		
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
        _, err := s.validateFileHandle(req.DirectoryHandle)
        if err != nil {
            return &api.LookupResponse{Status: api.Status_ERR_BADHANDLE}, nil
        }
        
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
        
        attrs := nfs.FSInfoToProtoAttributes(fileInfo)
        
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
        _, err := s.validateFileHandle(req.FileHandle)
        if err != nil {
            return &api.ReadResponse{Status: api.Status_ERR_BADHANDLE}, nil
        }
        
        // Convert file handle to path
        path, err := s.fileSystem.FileHandleToPath(req.FileHandle)
        if err != nil {
            return &api.ReadResponse{Status: nfs.MapErrorToStatus(err)}, nil
        }
        
        creds := nfs.ProtoCredsToFSCreds(req.Credentials)
        
        if s.config.EnableRootSquash && creds.UID == 0 {
            creds.UID = s.config.AnonUID
            creds.GID = s.config.AnonGID
        }
        
        // Check read permission
        if err := s.fileSystem.Access(ctx, path, fs.FileMode(4), creds); err != nil { // 4 = read
            return &api.ReadResponse{Status: nfs.MapErrorToStatus(err)}, nil
        }
        
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
    // Create a unique request ID and get client address
    reqID := fmt.Sprintf("write-%d", time.Now().UnixNano())
    clientAddr := "unknown"
    if peer, ok := ctx.Value("peer").(*net.Addr); ok && peer != nil {
        clientAddr = (*peer).String()
    }
    
    result, err := s.processRequest(ctx, "Write", reqID, clientAddr, func() (interface{}, error) {
        _, err := s.validateFileHandle(req.FileHandle)
        if err != nil {
            return &api.WriteResponse{Status: api.Status_ERR_BADHANDLE}, nil
        }
        
        path, err := s.fileSystem.FileHandleToPath(req.FileHandle)
        if err != nil {
            return &api.WriteResponse{Status: nfs.MapErrorToStatus(err)}, nil
        }
        
        // Get credentials
        creds := nfs.ProtoCredsToFSCreds(req.Credentials)
        
        // Apply root squashing if enabled
        if s.config.EnableRootSquash && creds.UID == 0 {
            creds.UID = s.config.AnonUID
            creds.GID = s.config.AnonGID
        }
        
        // Check write permission
        if err := s.fileSystem.Access(ctx, path, fs.FileMode(2), creds); err != nil { // 2 = write
            return &api.WriteResponse{Status: nfs.MapErrorToStatus(err)}, nil
        }
        
        fileInfo, err := s.fileSystem.GetAttr(ctx, path)
        if err != nil {
            return &api.WriteResponse{Status: nfs.MapErrorToStatus(err)}, nil
        }
        
        if fileInfo.Type != fs.FileTypeRegular {
            return &api.WriteResponse{Status: api.Status_ERR_ISDIR}, nil
        }
        
        dataSize := len(req.Data)
        if dataSize > s.config.MaxWriteSize {
            return &api.WriteResponse{Status: api.Status_ERR_FBIG}, nil
        }
        
        // Check for idempotent write using request ID
        cacheKey := fmt.Sprintf("write-%s-%d-%d", string(req.FileHandle), req.Offset, crc32.ChecksumIEEE(req.Data))
        if cachedResp, found := s.getCachedResponse(cacheKey); found {
            log.Printf("Found cached response for write operation: %s", cacheKey)
            return cachedResp, nil
        }
        
        // Determine if synchronous write is required
        sync := req.Stability == 2 // FILE_SYNC = 2
        
        // Write data to file
        bytesWritten, err := s.fileSystem.Write(ctx, path, int64(req.Offset), req.Data, sync)
        if err != nil {
            return &api.WriteResponse{Status: nfs.MapErrorToStatus(err)}, nil
        }
        
        // Sync to disk if requested
        if req.Stability == 1 { // DATA_SYNC = 1
            // For DATA_SYNC, we need to ensure the data is on stable storage
            if err := s.fileSystem.Commit(ctx, path); err != nil {
                return &api.WriteResponse{Status: nfs.MapErrorToStatus(err)}, nil
            }
        }
        
        // Get updated file attributes
        newFileInfo, _ := s.fileSystem.GetAttr(ctx, path)
        attrs := nfs.FSInfoToProtoAttributes(newFileInfo)
        
        resp := &api.WriteResponse{
            Status:     api.Status_OK,
            Count:      uint32(bytesWritten),
            Stability:  req.Stability, // Return the same stability level that was requested
            Verifier:   s.writeVerifier,
            Attributes: attrs,
        }
        
        // TODO: Cache the response for idempotent operations
        s.cacheResponse(cacheKey, resp, 5*time.Minute)
        
        return resp, nil
    })
    
    if err != nil {
        return nil, err
    }
    
    return result.(*api.WriteResponse), nil
}

// getCachedResponse retrieves a cached response if available
func (s *NFSServer) getCachedResponse(key string) (interface{}, bool) {
    s.reqCacheMu.RLock()
    defer s.reqCacheMu.RUnlock()
    
    if resp, ok := s.reqCache[key]; ok {
        return resp, true
    }
    return nil, false
}

// cacheResponse stores a response in the cache with an expiration time
func (s *NFSServer) cacheResponse(key string, response interface{}, ttl time.Duration) {
    s.reqCacheMu.Lock()
    defer s.reqCacheMu.Unlock()
    
    s.reqCache[key] = response
    
    // Set up automatic cleanup after TTL expires
    time.AfterFunc(ttl, func() {
        s.reqCacheMu.Lock()
        delete(s.reqCache, key)
        s.reqCacheMu.Unlock()
    })
}

// ReadDir implements the ReadDir RPC method
func (s *NFSServer) ReadDir(ctx context.Context, req *api.ReadDirRequest) (*api.ReadDirResponse, error) {
    reqID := fmt.Sprintf("readdir-%d", time.Now().UnixNano())
    clientAddr := "unknown"
    if peer, ok := ctx.Value("peer").(*net.Addr); ok && peer != nil {
        clientAddr = (*peer).String()
    }
    
    result, err := s.processRequest(ctx, "ReadDir", reqID, clientAddr, func() (interface{}, error) {
        _, err := s.validateFileHandle(req.DirectoryHandle)
        if err != nil {
            return &api.ReadDirResponse{Status: api.Status_ERR_BADHANDLE}, nil
        }
        
        dirPath, err := s.fileSystem.FileHandleToPath(req.DirectoryHandle)
        if err != nil {
            return &api.ReadDirResponse{Status: nfs.MapErrorToStatus(err)}, nil
        }
        
        // Get credentials
        creds := nfs.ProtoCredsToFSCreds(req.Credentials)
        
        if s.config.EnableRootSquash && creds.UID == 0 {
            creds.UID = s.config.AnonUID
            creds.GID = s.config.AnonGID
        }
        
        // Check directory access permission
        if err := s.fileSystem.Access(ctx, dirPath, fs.FileMode(4), creds); err != nil { // 4 = read
            return &api.ReadDirResponse{Status: nfs.MapErrorToStatus(err)}, nil
        }
        
        fileInfo, err := s.fileSystem.GetAttr(ctx, dirPath)
        if err != nil {
            return &api.ReadDirResponse{Status: nfs.MapErrorToStatus(err)}, nil
        }
        
        if fileInfo.Type != fs.FileTypeDirectory {
            return &api.ReadDirResponse{Status: api.Status_ERR_NOTDIR}, nil
        }
        
        // Determine the maximum number of entries to return
        maxCount := int(req.Count)
        if maxCount <= 0 {
            maxCount = 1000 // Default limit if not specified
        } else if maxCount > 10000 {
            maxCount = 10000 // Hard upper limit
        }
        
        // Read directory entries
        entries, _, err := s.fileSystem.ReadDir(ctx, dirPath, int64(req.Cookie), maxCount)
        if err != nil {
            return &api.ReadDirResponse{Status: nfs.MapErrorToStatus(err)}, nil
        }
        
        // Convert file system entries to protocol entries
        protoEntries := make([]*api.DirEntry, len(entries))
        for i, entry := range entries {
            protoEntries[i] = &api.DirEntry{
                FileId: entry.FileId,
                Name:   entry.Name,
                Cookie: uint64(entry.Cookie),
            }
        }
        
        // Generate a cookie verifier (simple timestamp-based)
        cookieVerifier := uint64(time.Now().UnixNano())
        
        // Check if we've reached the end of the directory
        eof := len(entries) < maxCount
        
        // Return the response
        return &api.ReadDirResponse{
            Status:         api.Status_OK,
            CookieVerifier: cookieVerifier,
            Entries:        protoEntries,
            Eof:            eof,
        }, nil
    })
    
    if err != nil {
        return nil, err
    }
    
    return result.(*api.ReadDirResponse), nil
}

// Create implements the Create RPC method
func (s *NFSServer) Create(ctx context.Context, req *api.CreateRequest) (*api.CreateResponse, error) {
    reqID := fmt.Sprintf("create-%d", time.Now().UnixNano())
    clientAddr := "unknown"
    if peer, ok := ctx.Value("peer").(*net.Addr); ok && peer != nil {
        clientAddr = (*peer).String()
    }
    
    // Process the request
    result, err := s.processRequest(ctx, "Create", reqID, clientAddr, func() (interface{}, error) {
        _, err := s.validateFileHandle(req.DirectoryHandle)
        if err != nil {
            return &api.CreateResponse{Status: api.Status_ERR_BADHANDLE}, nil
        }
        
        dirPath, err := s.fileSystem.FileHandleToPath(req.DirectoryHandle)
        if err != nil {
            return &api.CreateResponse{Status: nfs.MapErrorToStatus(err)}, nil
        }
        
        // Get credentials
        creds := nfs.ProtoCredsToFSCreds(req.Credentials)
        
        if s.config.EnableRootSquash && creds.UID == 0 {
            creds.UID = s.config.AnonUID
            creds.GID = s.config.AnonGID
        }
        
        // Check write permission on directory
        // if err := s.fileSystem.Access(ctx, dirPath, fs.FileMode(2|1), creds); err != nil {
        //     return &api.CreateResponse{Status: nfs.MapErrorToStatus(err)}, nil
        // }
        
        attr := nfs.ProtoAttributesToFSAttr(req.Attributes)
        
        // Set exclusive create flag based on mode
        exclusive := false
        if req.Mode == api.CreateMode_GUARDED || req.Mode == api.CreateMode_EXCLUSIVE {
            exclusive = true
        }
        
        // Handle EXCLUSIVE mode (check if file exists with this verifier)
        if req.Mode == api.CreateMode_EXCLUSIVE {
            // Check if file exists
            targetPath := filepath.Join(dirPath, req.Name)
            
            // If we have this verifier cached for this path, return the cached response
            cacheKey := fmt.Sprintf("create-excl-%s-%d", targetPath, req.Verifier)
            if cachedResp, found := s.getCachedResponse(cacheKey); found {
                log.Printf("Found cached exclusive create response: %s", cacheKey)
                return cachedResp, nil
            }
            
            // Try to get file info
            _, err := s.fileSystem.GetAttr(ctx, targetPath)
            if err == nil {
                // File exists, return error for GUARDED mode
                if req.Mode == api.CreateMode_GUARDED {
                    return &api.CreateResponse{Status: api.Status_ERR_EXIST}, nil
                }
                
                // For EXCLUSIVE mode, create a new response and cache it
                fileHandle, err := s.fileSystem.PathToFileHandle(targetPath)
                if err != nil {
                    return &api.CreateResponse{Status: nfs.MapErrorToStatus(err)}, nil
                }
                
                fileInfo, err := s.fileSystem.GetAttr(ctx, targetPath)
                if err != nil {
                    return &api.CreateResponse{Status: nfs.MapErrorToStatus(err)}, nil
                }
                
                var dirAttrs *api.FileAttributes
                dirInfo, err := s.fileSystem.GetAttr(ctx, dirPath)
                if err == nil {
                    dirAttrs = nfs.FSInfoToProtoAttributes(dirInfo)
                }
                
                resp := &api.CreateResponse{
                    Status:        api.Status_OK,
                    FileHandle:    fileHandle,
                    Attributes:    nfs.FSInfoToProtoAttributes(fileInfo),
                    DirAttributes: dirAttrs,
                }
                
                // Cache the response for this exclusive create
                s.cacheResponse(cacheKey, resp, 10*time.Minute)
                
                return resp, nil
            }
        }
        
        // Create the file
        filePath, fileInfo, err := s.fileSystem.Create(ctx, dirPath, req.Name, attr, exclusive)
        if err != nil {
            return &api.CreateResponse{Status: nfs.MapErrorToStatus(err)}, nil
        }
        
        // Generate file handle for the new file
        fileHandle, err := s.fileSystem.PathToFileHandle(filePath)
        if err != nil {
            return &api.CreateResponse{Status: nfs.MapErrorToStatus(err)}, nil
        }
        
        // Get directory attributes if requested
        var dirAttrs *api.FileAttributes
        dirInfo, err := s.fileSystem.GetAttr(ctx, dirPath)
        if err == nil {
            dirAttrs = nfs.FSInfoToProtoAttributes(dirInfo)
        }
        
        // If EXCLUSIVE mode, cache the response
        if req.Mode == api.CreateMode_EXCLUSIVE {
            cacheKey := fmt.Sprintf("create-excl-%s-%d", filepath.Join(dirPath, req.Name), req.Verifier)
            s.cacheResponse(cacheKey, &api.CreateResponse{
                Status:        api.Status_OK,
                FileHandle:    fileHandle,
                Attributes:    nfs.FSInfoToProtoAttributes(fileInfo),
                DirAttributes: dirAttrs,
            }, 10*time.Minute)
        }
        
        return &api.CreateResponse{
            Status:        api.Status_OK,
            FileHandle:    fileHandle,
            Attributes:    nfs.FSInfoToProtoAttributes(fileInfo),
            DirAttributes: dirAttrs,
        }, nil
    })
    
    if err != nil {
        return nil, err
    }
    
    return result.(*api.CreateResponse), nil
}

// Mkdir implements the Mkdir RPC method
func (s *NFSServer) Mkdir(ctx context.Context, req *api.MkdirRequest) (*api.MkdirResponse, error) {
    reqID := fmt.Sprintf("mkdir-%d", time.Now().UnixNano())
    clientAddr := "unknown"
    if peer, ok := ctx.Value("peer").(*net.Addr); ok && peer != nil {
        clientAddr = (*peer).String()
    }
    
    result, err := s.processRequest(ctx, "Mkdir", reqID, clientAddr, func() (interface{}, error) {
        _, err := s.validateFileHandle(req.DirectoryHandle)
        if err != nil {
            return &api.MkdirResponse{Status: api.Status_ERR_BADHANDLE}, nil
        }
        
        dirPath, err := s.fileSystem.FileHandleToPath(req.DirectoryHandle)
        if err != nil {
            return &api.MkdirResponse{Status: nfs.MapErrorToStatus(err)}, nil
        }
        
        // Get credentials
        creds := nfs.ProtoCredsToFSCreds(req.Credentials)
        
        if s.config.EnableRootSquash && creds.UID == 0 {
            creds.UID = s.config.AnonUID
            creds.GID = s.config.AnonGID
        }
        
        // Check write permission on parent directory
        // if err := s.fileSystem.Access(ctx, dirPath, fs.FileMode(2|1), creds); err != nil {
        //     return &api.MkdirResponse{Status: nfs.MapErrorToStatus(err)}, nil
        // }
        
        attr := nfs.ProtoAttributesToFSAttr(req.Attributes)
        
        // Create the directory
        newDirPath, dirInfo, err := s.fileSystem.Mkdir(ctx, dirPath, req.Name, attr)
        if err != nil {
            return &api.MkdirResponse{Status: nfs.MapErrorToStatus(err)}, nil
        }
        
        // Generate directory handle for the new directory
        dirHandle, err := s.fileSystem.PathToFileHandle(newDirPath)
        if err != nil {
            return &api.MkdirResponse{Status: nfs.MapErrorToStatus(err)}, nil
        }
        
        // Get parent directory attributes if requested
        var parentAttrs *api.FileAttributes
        parentInfo, err := s.fileSystem.GetAttr(ctx, dirPath)
        if err == nil {
            parentAttrs = nfs.FSInfoToProtoAttributes(parentInfo)
        }
        
        return &api.MkdirResponse{
            Status:         api.Status_OK,
            DirectoryHandle: dirHandle,
            Attributes:     nfs.FSInfoToProtoAttributes(dirInfo),
            DirAttributes:  parentAttrs,
        }, nil
    })
    
    if err != nil {
        return nil, err
    }
    
    return result.(*api.MkdirResponse), nil
}

// GetRootHandle implements the GetRootHandle RPC method
func (s *NFSServer) GetRootHandle(ctx context.Context, req *api.GetRootHandleRequest) (*api.GetRootHandleResponse, error) {
    reqID := fmt.Sprintf("getroot-%d", time.Now().UnixNano())
    clientAddr := "unknown"
    if peer, ok := ctx.Value("peer").(*net.Addr); ok && peer != nil {
        clientAddr = (*peer).String()
    }
    
    // Process the request
    result, err := s.processRequest(ctx, "GetRootHandle", reqID, clientAddr, func() (interface{}, error) {
        creds := nfs.ProtoCredsToFSCreds(req.Credentials)
        
        if s.config.EnableRootSquash && creds.UID == 0 {
            creds.UID = s.config.AnonUID
            creds.GID = s.config.AnonGID
        }
        
        // Get root directory handle
        rootHandle, err := s.fileSystem.PathToFileHandle("/")
        if err != nil {
            return &api.GetRootHandleResponse{Status: nfs.MapErrorToStatus(err)}, nil
        }
        
        rootInfo, err := s.fileSystem.GetAttr(ctx, "/")
        if err != nil {
            return &api.GetRootHandleResponse{Status: nfs.MapErrorToStatus(err)}, nil
        }
        
        // Convert to NFS attributes
        attrs := nfs.FSInfoToProtoAttributes(rootInfo)
        
        return &api.GetRootHandleResponse{
            Status:     api.Status_OK,
            FileHandle: rootHandle,
            Attributes: attrs,
        }, nil
    })
    
    if err != nil {
        return nil, err
    }
    
    return result.(*api.GetRootHandleResponse), nil
}

// Commit implements the Commit RPC method
func (s *NFSServer) Commit(ctx context.Context, req *api.CommitRequest) (*api.CommitResponse, error) {
    // Create a unique request ID and get client address
    reqID := fmt.Sprintf("commit-%d", time.Now().UnixNano())
    clientAddr := "unknown"
    if peer, ok := ctx.Value("peer").(*net.Addr); ok && peer != nil {
        clientAddr = (*peer).String()
    }
    
    // Process the request
    result, err := s.processRequest(ctx, "Commit", reqID, clientAddr, func() (interface{}, error) {
        // Validate file handle
        _, err := s.validateFileHandle(req.FileHandle)
        if err != nil {
            return &api.CommitResponse{Status: api.Status_ERR_BADHANDLE}, nil
        }
        
        // Convert file handle to path
        path, err := s.fileSystem.FileHandleToPath(req.FileHandle)
        if err != nil {
            return &api.CommitResponse{Status: nfs.MapErrorToStatus(err)}, nil
        }
        
        // Check if verifier matches
        if req.WriteVerifier != s.writeVerifier {
            // Verifier doesn't match, server may have restarted
            return &api.CommitResponse{
                Status:   api.Status_ERR_NOT_SYNC,
                Verifier: s.writeVerifier,
            }, nil
        }
        
        // Commit data to stable storage
        err = s.fileSystem.Commit(ctx, path)
        if err != nil {
            return &api.CommitResponse{Status: nfs.MapErrorToStatus(err)}, nil
        }
        
        // Return successful response
        return &api.CommitResponse{
            Status:   api.Status_OK,
            Verifier: s.writeVerifier,
        }, nil
    })
    
    if err != nil {
        return nil, err
    }
    
    return result.(*api.CommitResponse), nil
}