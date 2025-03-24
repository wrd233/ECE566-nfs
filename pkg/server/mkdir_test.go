package server

import (
    "context"
    "os"
    "path/filepath"
    "testing"

    "github.com/example/nfsserver/pkg/api"
    "github.com/example/nfsserver/pkg/fs/local"
)

func TestMkdir(t *testing.T) {
    // Create temporary directory
    tempDir, err := os.MkdirTemp("", "nfs-test-")
    if err != nil {
        t.Fatalf("Failed to create temp dir: %v", err)
    }
    defer os.RemoveAll(tempDir)

    // Create filesystem
    fs, err := local.NewLocalFileSystem(tempDir)
    if err != nil {
        t.Fatalf("Failed to create filesystem: %v", err)
    }

    // Create server with root squashing disabled
    config := DefaultConfig()
    config.EnableRootSquash = false // Disable root squashing for test
    server, err := NewNFSServer(config, fs)
    if err != nil {
        t.Fatalf("Failed to create server: %v", err)
    }

    // Get root directory handle
    rootHandle, err := fs.PathToFileHandle("/")
    if err != nil {
        t.Fatalf("Failed to get root directory handle: %v", err)
    }

    // Create credentials
    creds := &api.Credentials{
        Uid:    0, // Use root UID
        Gid:    0, // Use root GID
        Groups: []uint32{0},
    }

    // Test creating a directory
    req := &api.MkdirRequest{
        DirectoryHandle: rootHandle,
        Name:            "newdir",
        Credentials:     creds,
        Attributes:      &api.FileAttributes{Mode: 0755},
    }

    // Call Mkdir
    resp, err := server.Mkdir(context.Background(), req)
    if err != nil {
        t.Fatalf("Mkdir failed: %v", err)
    }

    // Check response
    if resp.Status != api.Status_OK {
        t.Errorf("Unexpected status: got %v, want OK", resp.Status)
    }

    if resp.DirectoryHandle == nil || len(resp.DirectoryHandle) == 0 {
        t.Error("DirectoryHandle is empty")
    }
    
    if resp.Attributes == nil {
        t.Error("Attributes is nil")
    } else if resp.Attributes.Type != api.FileType_DIRECTORY {
        t.Errorf("Wrong file type: got %v, want DIRECTORY", resp.Attributes.Type)
    }

    // Verify directory exists
    dirPath := filepath.Join(tempDir, "newdir")
    fi, err := os.Stat(dirPath)
    if err != nil {
        t.Errorf("Directory doesn't exist: %v", err)
    } else if !fi.IsDir() {
        t.Error("Created path is not a directory")
    }

    // Test creating a directory with custom attributes
    customReq := &api.MkdirRequest{
        DirectoryHandle: rootHandle,
        Name:            "customdir",
        Credentials:     creds,
        Attributes:      &api.FileAttributes{Mode: 0700},
    }

    // Call Mkdir
    customResp, err := server.Mkdir(context.Background(), customReq)
    if err != nil {
        t.Fatalf("Mkdir with custom attributes failed: %v", err)
    }

    if customResp.Status != api.Status_OK {
        t.Errorf("Unexpected status for custom dir: got %v, want OK", customResp.Status)
    }

    // Verify permissions
    customPath := filepath.Join(tempDir, "customdir")
    customFi, err := os.Stat(customPath)
    if err != nil {
        t.Errorf("Custom directory doesn't exist: %v", err)
    } else {
        // Check permissions - may need adjustment for different platforms
        if customFi.Mode().Perm()&0700 != 0700 {
            t.Errorf("Wrong permissions: got %o, want 0700", customFi.Mode().Perm())
        }
    }

    // Test creating an already existing directory
    dupReq := &api.MkdirRequest{
        DirectoryHandle: rootHandle,
        Name:            "newdir", // Already exists
        Credentials:     creds,
        Attributes:      &api.FileAttributes{Mode: 0755},
    }

    // Call Mkdir
    dupResp, err := server.Mkdir(context.Background(), dupReq)
    if err != nil {
        t.Fatalf("Mkdir with duplicate name failed unexpectedly: %v", err)
    }

    if dupResp.Status != api.Status_ERR_EXIST {
        t.Errorf("Expected ERR_EXIST for duplicate directory, got: %v", dupResp.Status)
    }

    // Test with invalid handle
    badReq := &api.MkdirRequest{
        DirectoryHandle: []byte{1, 2, 3}, // Invalid handle
        Name:            "baddir",
        Credentials:     creds,
        Attributes:      &api.FileAttributes{Mode: 0755},
    }
    
    badResp, err := server.Mkdir(context.Background(), badReq)
    if err != nil {
        t.Fatalf("Mkdir with invalid handle failed unexpectedly: %v", err)
    }
    
    if badResp.Status != api.Status_ERR_BADHANDLE {
        t.Errorf("Expected ERR_BADHANDLE for invalid handle, got: %v", badResp.Status)
    }
}