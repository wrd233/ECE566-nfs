package server

import (
    "context"
    "os"
    "path/filepath"
    "testing"

    "github.com/example/nfsserver/pkg/api"
    "github.com/example/nfsserver/pkg/fs/local"
)

func TestReadDir(t *testing.T) {
    // Create temporary directory
    tempDir, err := os.MkdirTemp("", "nfs-test-")
    if err != nil {
        t.Fatalf("Failed to create temp dir: %v", err)
    }
    defer os.RemoveAll(tempDir)

    // Create test directory structure
    testDirPath := filepath.Join(tempDir, "testdir")
    if err := os.Mkdir(testDirPath, 0755); err != nil {
        t.Fatalf("Failed to create test directory: %v", err)
    }
    
    // Create test files with specific permissions
    testFiles := []string{"file1.txt", "file2.txt", "file3.txt"}
    for _, name := range testFiles {
        filePath := filepath.Join(testDirPath, name)
        content := []byte("content for " + name)
        if err := os.WriteFile(filePath, content, 0644); err != nil {
            t.Fatalf("Failed to create test file %s: %v", name, err)
        }
    }
    
    // Create subdirectory
    subDirPath := filepath.Join(testDirPath, "subdir")
    if err := os.Mkdir(subDirPath, 0755); err != nil {
        t.Fatalf("Failed to create subdirectory: %v", err)
    }

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

    // Get directory handle
    dirHandle, err := fs.PathToFileHandle("/testdir")
    if err != nil {
        t.Fatalf("Failed to get directory handle: %v", err)
    }

    // Create credentials
    creds := &api.Credentials{
        Uid:    0, // Use root UID
        Gid:    0, // Use root GID
        Groups: []uint32{0},
    }

    // Test reading directory
    req := &api.ReadDirRequest{
        DirectoryHandle: dirHandle,
        Credentials:     creds,
        Cookie:          0, // Start from beginning
        CookieVerifier:  0,
        Count:           100, // Request more than enough entries
    }

    // Call ReadDir
    resp, err := server.ReadDir(context.Background(), req)
    if err != nil {
        t.Fatalf("ReadDir failed: %v", err)
    }

    // Check response
    if resp.Status != api.Status_OK {
        t.Errorf("Unexpected status: got %v, want OK", resp.Status)
    }

    // Should have at least 6 entries (., .., 3 files, and 1 subdirectory)
    if len(resp.Entries) < 6 {
        t.Errorf("Wrong number of entries: got %d, want at least 6", len(resp.Entries))
    }

    // Verify we have the expected entries (names only)
    expectedNames := map[string]bool{
        ".":        true,
        "..":       true,
        "file1.txt": true,
        "file2.txt": true,
        "file3.txt": true,
        "subdir":   true,
    }

    foundNames := make(map[string]bool)
    for _, entry := range resp.Entries {
        foundNames[entry.Name] = true
    }

    for name := range expectedNames {
        if !foundNames[name] {
            t.Errorf("Missing directory entry: %s", name)
        }
    }

    // Test pagination
    if len(resp.Entries) > 0 {
        // Get the cookie from the first entry
        firstCookie := resp.Entries[0].Cookie
        
        // Make a request starting from that cookie
        req2 := &api.ReadDirRequest{
            DirectoryHandle: dirHandle,
            Credentials:     creds,
            Cookie:          firstCookie,
            CookieVerifier:  resp.CookieVerifier,
            Count:           100,
        }
        
        // Call ReadDir again
        resp2, err := server.ReadDir(context.Background(), req2)
        if err != nil {
            t.Fatalf("ReadDir with cookie failed: %v", err)
        }
        
        if resp2.Status != api.Status_OK {
            t.Errorf("Unexpected status for paginated request: got %v, want OK", resp2.Status)
        }
        
        // Should have one fewer entry than before
        if len(resp2.Entries) != len(resp.Entries) - 1 {
            t.Errorf("Wrong number of entries in paginated response: got %d, want %d", 
                len(resp2.Entries), len(resp.Entries) - 1)
        }
    }

    // Test with invalid handle
    badReq := &api.ReadDirRequest{
        DirectoryHandle: []byte{1, 2, 3}, // Invalid handle
        Credentials:     creds,
        Cookie:          0,
        CookieVerifier:  0,
        Count:           100,
    }
    
    badResp, err := server.ReadDir(context.Background(), badReq)
    if err != nil {
        t.Fatalf("ReadDir with invalid handle failed unexpectedly: %v", err)
    }
    
    if badResp.Status != api.Status_ERR_BADHANDLE {
        t.Errorf("Expected ERR_BADHANDLE for invalid handle, got: %v", badResp.Status)
    }

    // Test reading a file (not a directory)
    fileHandle, err := fs.PathToFileHandle("/testdir/file1.txt")
    if err != nil {
        t.Fatalf("Failed to get file handle: %v", err)
    }
    
    fileReq := &api.ReadDirRequest{
        DirectoryHandle: fileHandle,
        Credentials:     creds,
        Cookie:          0,
        CookieVerifier:  0,
        Count:           100,
    }
    
    fileResp, err := server.ReadDir(context.Background(), fileReq)
    if err != nil {
        t.Fatalf("ReadDir on file failed unexpectedly: %v", err)
    }
    
    if fileResp.Status != api.Status_ERR_NOTDIR {
        t.Errorf("Expected ERR_NOTDIR for file, got: %v", fileResp.Status)
    }
}