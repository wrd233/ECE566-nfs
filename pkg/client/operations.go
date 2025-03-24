package client

import (
	"context"
	"fmt"
	"time"
	"strings"
    "path/filepath"

	"github.com/example/nfsserver/pkg/api"
)

// Ensure Client implements NFSClient interface
var _ NFSClient = (*Client)(nil)

// GetAttr retrieves attributes for a file or directory
func (c *Client) GetAttr(ctx context.Context, fileHandle []byte) (*api.FileAttributes, error) {
    // Create request
    req := &api.GetAttrRequest{
        FileHandle: fileHandle,
        Credentials: &api.Credentials{
            Uid: 1000,
            Gid: 1000,
            Groups: []uint32{1000},
        },
    }
    
    // Create a context with timeout
    callCtx, cancel := context.WithTimeout(ctx, c.config.Timeout)
    defer cancel()
    
    // Call the RPC method with retry logic
    var resp *api.GetAttrResponse
    var err error
    
    err = c.callWithRetry(callCtx, "GetAttr", func(retryCtx context.Context) error {
        resp, err = c.nfsClient.GetAttr(retryCtx, req)
        return err
    })
    
    if err != nil {
        return nil, fmt.Errorf("GetAttr RPC failed: %w", err)
    }
    
    // Check the status
    if resp.Status != api.Status_OK {
        return nil, StatusToError("GetAttr", resp.Status)
    }
    
    return resp.Attributes, nil
}

// Lookup looks up a file name in a directory
func (c *Client) Lookup(ctx context.Context, dirHandle []byte, name string) ([]byte, *api.FileAttributes, error) {
    // Create request
    req := &api.LookupRequest{
        DirectoryHandle: dirHandle,
        Name: name,
        Credentials: &api.Credentials{
            Uid: 1000,
            Gid: 1000,
            Groups: []uint32{1000},
        },
    }
    
    // Create a context with timeout
    callCtx, cancel := context.WithTimeout(ctx, c.config.Timeout)
    defer cancel()
    
    // Call the RPC method with retry logic
    var resp *api.LookupResponse
    var err error
    
    err = c.callWithRetry(callCtx, "Lookup", func(retryCtx context.Context) error {
        resp, err = c.nfsClient.Lookup(retryCtx, req)
        return err
    })
    
    if err != nil {
        return nil, nil, fmt.Errorf("Lookup RPC failed: %w", err)
    }
    
    // Check the status
    if resp.Status != api.Status_OK {
        return nil, nil, StatusToError("Lookup", resp.Status)
    }
    
    // 成功后，尝试更新缓存
    if c.handleCache != nil {
        c.handleCache.StorePathHandle(name, resp.FileHandle)
    }
    
    return resp.FileHandle, resp.Attributes, nil
}

// Read reads data from a file
func (c *Client) Read(ctx context.Context, fileHandle []byte, offset int64, count int) ([]byte, bool, error) {
	// Create request
	_ = &api.ReadRequest{
		FileHandle: fileHandle,
		Credentials: &api.Credentials{
			Uid: 1000,
			Gid: 1000,
		},
		Offset: uint64(offset),
		Count: uint32(count),
	}
	
	// TODO: Implement actual Read operation
	return nil, false, fmt.Errorf("not implemented")
}

// Write writes data to a file
func (c *Client) Write(ctx context.Context, fileHandle []byte, offset int64, data []byte, stability int) (int, error) {
	// Create request
	_ = &api.WriteRequest{
		FileHandle: fileHandle,
		Credentials: &api.Credentials{
			Uid: 1000,
			Gid: 1000,
		},
		Offset: uint64(offset),
		Data: data,
		Stability: uint32(stability),
	}
	
	// TODO: Implement actual Write operation
	return 0, fmt.Errorf("not implemented")
}

// ReadDir reads the contents of a directory
func (c *Client) ReadDir(ctx context.Context, dirHandle []byte) ([]*api.DirEntry, error) {
	// Create the request
	req := &api.ReadDirRequest{
		DirectoryHandle: dirHandle,
		Credentials: &api.Credentials{
			Uid: 1000,
			Gid: 1000,
		},
		Cookie: 0,
		CookieVerifier: 0,
		Count: 1000, // Request up to 1000 entries
	}
	
	// Create a context with timeout
	callCtx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()
	
	// Call the RPC method
	resp, err := c.nfsClient.ReadDir(callCtx, req)
	if err != nil {
		return nil, fmt.Errorf("ReadDir RPC failed: %w", err)
	}
	
	// Check the status
	if resp.Status != api.Status_OK {
		return nil, StatusToError("ReadDir", resp.Status)
	}
	
	return resp.Entries, nil
}

// Create creates a new file
func (c *Client) Create(ctx context.Context, dirHandle []byte, name string, attrs *api.FileAttributes, mode api.CreateMode) ([]byte, *api.FileAttributes, error) {
	// Create request
	_ = &api.CreateRequest{
		DirectoryHandle: dirHandle,
		Name: name,
		Credentials: &api.Credentials{
			Uid: 1000,
			Gid: 1000,
		},
		Attributes: attrs,
		Mode: mode,
		Verifier: uint64(time.Now().UnixNano()), // Use current time as verifier
	}
	
	// TODO: Implement actual Create operation
	return nil, nil, fmt.Errorf("not implemented")
}

// Mkdir creates a new directory
func (c *Client) Mkdir(ctx context.Context, dirHandle []byte, name string, attrs *api.FileAttributes) ([]byte, *api.FileAttributes, error) {
	// Create request
	_ = &api.MkdirRequest{
		DirectoryHandle: dirHandle,
		Name: name,
		Credentials: &api.Credentials{
			Uid: 1000,
			Gid: 1000,
		},
		Attributes: attrs,
	}
	
	// TODO: Implement actual Mkdir operation
	return nil, nil, fmt.Errorf("not implemented")
}

// Remove removes a file
func (c *Client) Remove(ctx context.Context, dirHandle []byte, name string) error {
	// TODO: Implement actual Remove operation
	return fmt.Errorf("not implemented")
}

// Rmdir removes a directory
func (c *Client) Rmdir(ctx context.Context, dirHandle []byte, name string) error {
	// TODO: Implement actual Rmdir operation
	return fmt.Errorf("not implemented")
}

// Rename renames a file or directory
func (c *Client) Rename(ctx context.Context, fromDirHandle []byte, fromName string, toDirHandle []byte, toName string) error {
	// TODO: Implement actual Rename operation
	return fmt.Errorf("not implemented")
}

// GetRootFileHandle retrieves the root directory file handle from the server
func (c *Client) GetRootFileHandle(ctx context.Context) ([]byte, error) {
    // Create request
    req := &api.GetRootHandleRequest{
        Credentials: &api.Credentials{
            Uid: 1000,
            Gid: 1000,
            Groups: []uint32{1000},
        },
    }
    
    // Create a context with timeout
    callCtx, cancel := context.WithTimeout(ctx, c.config.Timeout)
    defer cancel()
    
    // Call the RPC method with retry logic
    var resp *api.GetRootHandleResponse
    var err error
    
    err = c.callWithRetry(callCtx, "GetRootFileHandle", func(retryCtx context.Context) error {
        resp, err = c.nfsClient.GetRootHandle(retryCtx, req)
        return err
    })
    
    if err != nil {
        return nil, fmt.Errorf("Fail to get root: %w", err)
    }
    
    // Check the status
    if resp.Status != api.Status_OK {
        return nil, StatusToError("GetRootFileHandle", resp.Status)
    }
    
    return resp.FileHandle, nil
}

// LookupPath resolves a file path to a file handle, starting from the root
// Implementation of the ExtendedNFSClient interface
func (c *Client) LookupPath(ctx context.Context, path string) ([]byte, error) {
    // 检查路径是否有效
    if path == "" {
        return nil, fmt.Errorf("路径不能为空")
    }
    
    // 规范化路径，确保以/开头，移除末尾的/（除非是根路径）
    if !strings.HasPrefix(path, "/") {
        path = "/" + path
    }
    
    // 如果路径是根目录，则直接返回根目录句柄
    if path == "/" {
        return c.GetRootFileHandle(ctx)
    }
    
    // 获取根目录句柄
    currentHandle, err := c.GetRootFileHandle(ctx)
    if err != nil {
        return nil, fmt.Errorf("无法获取根目录句柄: %w", err)
    }
    
    // 将路径拆分为组件
    components := strings.Split(strings.TrimPrefix(path, "/"), "/")
    
    // 逐个组件查找
    currentPath := "/"
    for _, component := range components {
        // 跳过空组件
        if component == "" {
            continue
        }
        
        // 更新当前路径
        currentPath = filepath.Join(currentPath, component)
        
        // 调用Lookup获取下一级组件的句柄
        nextHandle, _, err := c.Lookup(ctx, currentHandle, component)
        if err != nil {
            return nil, fmt.Errorf("查找路径组件 '%s' 失败: %w", component, err)
        }
        
        // 更新当前句柄
        currentHandle = nextHandle
    }
    
    return currentHandle, nil
}