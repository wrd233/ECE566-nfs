package client

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/example/nfsserver/pkg/api"
)

// Ensure Client implements NFSClient interface
var _ NFSClient = (*Client)(nil)


// Helper function to create default credentials
func (c *Client) defaultCredentials() *api.Credentials {
    return &api.Credentials{
        Uid:    1000,
        Gid:    1000,
        Groups: []uint32{1000},
    }
}

// GetAttr retrieves attributes for a file or directory
func (c *Client) GetAttr(ctx context.Context, fileHandle []byte) (*api.FileAttributes, error) {
	// Create request
	req := &api.GetAttrRequest{
		FileHandle: fileHandle,
		Credentials: c.defaultCredentials(),
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
		Name:            name,
		Credentials: c.defaultCredentials(),
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
	// Limit read size if specified count is too large
	if count <= 0 {
		count = 1024 * 1024 // Default to 1MB if not specified
	} else if count > 10*1024*1024 {
		count = 10 * 1024 * 1024 // Cap at 10MB for safety
	}

	// Create request
	req := &api.ReadRequest{
		FileHandle: fileHandle,
		Credentials: c.defaultCredentials(),
		Offset: uint64(offset),
		Count:  uint32(count),
	}

	// Create a context with timeout
	callCtx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	// Call the RPC method with retry logic
	var resp *api.ReadResponse
	var err error

	err = c.callWithRetry(callCtx, "Read", func(retryCtx context.Context) error {
		resp, err = c.nfsClient.Read(retryCtx, req)
		return err
	})

	if err != nil {
		return nil, false, fmt.Errorf("Read RPC failed: %w", err)
	}

	// Check the status
	if resp.Status != api.Status_OK {
		return nil, false, StatusToError("Read", resp.Status)
	}

	return resp.Data, resp.Eof, nil
}

func (c *Client) Write(ctx context.Context, fileHandle []byte, offset int64, data []byte, stability int) (int, error) {
	return c.writeCache.Write(ctx, fileHandle, offset, data, stability)
}

func (c *Client) FlushAll(ctx context.Context) error {
	return c.writeCache.FlushAll(ctx)
}

// ReadDir reads the contents of a directory
func (c *Client) ReadDir(ctx context.Context, dirHandle []byte) ([]*api.DirEntry, error) {
	// Create the request
	req := &api.ReadDirRequest{
		DirectoryHandle: dirHandle,
		Credentials: c.defaultCredentials(),
		Cookie:         0,
		CookieVerifier: 0,
		Count:          1000, // Request up to 1000 entries
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

// Create creates a new file in the specified directory
func (c *Client) Create(ctx context.Context, dirHandle []byte, name string, attrs *api.FileAttributes, mode api.CreateMode) ([]byte, *api.FileAttributes, error) {
	// If attributes not provided, use defaults
	if attrs == nil {
		attrs = &api.FileAttributes{
			Mode: 0666, // Default to rw-rw-rw-
		}
	}

	// Create request
	req := &api.CreateRequest{
		DirectoryHandle: dirHandle,
		Name:            name,
		Credentials: c.defaultCredentials(),
		Attributes: attrs,
		Mode:       mode,
		Verifier:   uint64(time.Now().UnixNano()), // Use current time as verifier
	}

	// Create a context with timeout
	callCtx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	// Call the RPC method with retry logic
	var resp *api.CreateResponse
	var err error

	err = c.callWithRetry(callCtx, "Create", func(retryCtx context.Context) error {
		resp, err = c.nfsClient.Create(retryCtx, req)
		return err
	})

	if err != nil {
		return nil, nil, fmt.Errorf("Create RPC failed: %w", err)
	}

	// Check the status
	if resp.Status != api.Status_OK {
		return nil, nil, StatusToError("Create", resp.Status)
	}

	return resp.FileHandle, resp.Attributes, nil
}

// Mkdir creates a new directory
func (c *Client) Mkdir(ctx context.Context, dirHandle []byte, name string, attrs *api.FileAttributes) ([]byte, *api.FileAttributes, error) {
	// If attributes not provided, use defaults
	if attrs == nil {
		attrs = &api.FileAttributes{
			Mode: 0777, // Default to rwxrwxrwx for directories
		}
	}

	// Create request
	req := &api.MkdirRequest{
		DirectoryHandle: dirHandle,
		Name:            name,
		Credentials: c.defaultCredentials(),
		Attributes: attrs,
	}

	// Create a context with timeout
	callCtx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	// Call the RPC method with retry logic
	var resp *api.MkdirResponse
	var err error

	err = c.callWithRetry(callCtx, "Mkdir", func(retryCtx context.Context) error {
		resp, err = c.nfsClient.Mkdir(retryCtx, req)
		return err
	})

	if err != nil {
		return nil, nil, fmt.Errorf("Mkdir RPC failed: %w", err)
	}

	// Check the status
	if resp.Status != api.Status_OK {
		return nil, nil, StatusToError("Mkdir", resp.Status)
	}

	return resp.DirectoryHandle, resp.Attributes, nil
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
		Credentials:c.defaultCredentials(),
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

// Commit ensures data written to file is stored to stable storage
func (c *Client) Commit(ctx context.Context, fileHandle []byte) error {
	// Create request
	req := &api.CommitRequest{
		FileHandle: fileHandle,
		Credentials: c.defaultCredentials(),
	}

	// Create a context with timeout
	callCtx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	// Call the RPC method with retry logic
	var resp *api.CommitResponse
	var err error

	err = c.callWithRetry(callCtx, "Commit", func(retryCtx context.Context) error {
		resp, err = c.nfsClient.Commit(retryCtx, req)
		return err
	})

	if err != nil {
		return fmt.Errorf("Commit RPC failed: %w", err)
	}

	// Check the status
	if resp.Status != api.Status_OK {
		// If the server reports ERR_NOT_SYNC, it means the server restarted
		// In a full implementation, we would resend pending writes
		if resp.Status == api.Status_ERR_NOT_SYNC {
			return fmt.Errorf("server restarted, verifier mismatch")
		}
		return StatusToError("Commit", resp.Status)
	}

	return nil
}

// Remove removes a file
func (c *Client) Remove(ctx context.Context, dirHandle []byte, name string) error {
	req := &api.RemoveRequest{
		DirectoryHandle: dirHandle,
		Name:            name,
		Credentials: c.defaultCredentials(),
	}

	callCtx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	var resp *api.RemoveResponse
	var err error

	err = c.callWithRetry(callCtx, "Remove", func(retryCtx context.Context) error {
		resp, err = c.nfsClient.Remove(retryCtx, req)
		return err
	})

	if err != nil {
		return fmt.Errorf("Remove RPC failed: %w", err)
	}

	if resp.Status != api.Status_OK {
		return StatusToError("Remove", resp.Status)
	}

	return nil
}
