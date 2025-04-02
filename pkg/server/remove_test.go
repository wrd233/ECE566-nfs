package server

import (
    "context"
    "os"
    "path/filepath"
    "testing"

    "github.com/example/nfsserver/pkg/api"
    "github.com/example/nfsserver/pkg/fs/local"
)

func TestRemove(t *testing.T) {
    // Create temporary directory
    tempDir, err := os.MkdirTemp("", "nfs-test-")
    if err != nil {
        t.Fatalf("Failed to create temp dir: %v", err)
    }
    defer os.RemoveAll(tempDir)

    // Create test directory
    testDirPath := filepath.Join(tempDir, "testdir")
    if err := os.Mkdir(testDirPath, 0755); err != nil {
        t.Fatalf("Failed to create test directory: %v", err)
    }

    // Create test files
    testFiles := []string{
        "regular_file.txt",
        "empty_file.txt",
    }
    
    for _, filename := range testFiles {
        filePath := filepath.Join(testDirPath, filename)
        content := []byte("test content for " + filename)
        if err := os.WriteFile(filePath, content, 0644); err != nil {
            t.Fatalf("Failed to create test file %s: %v", filename, err)
        }
    }
    
    // Create a subdirectory
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

    // Test cases
    testCases := []struct {
        name       string
        fileName   string
        shouldExist bool
        expectedStatus api.Status
    }{
        {"Remove regular file", "regular_file.txt", false, api.Status_OK},
        {"Remove non-existent file", "nonexistent.txt", false, api.Status_ERR_NOENT},
        {"Try to remove directory", "subdir", true, api.Status_ERR_ISDIR},
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            // Create remove request
            req := &api.RemoveRequest{
                DirectoryHandle: dirHandle,
                Name:            tc.fileName,
                Credentials:     creds,
            }

            // Call Remove
            resp, err := server.Remove(context.Background(), req)
            if err != nil {
                t.Fatalf("Remove failed: %v", err)
            }

            // Check status
            if resp.Status != tc.expectedStatus {
                t.Errorf("Unexpected status: got %v, want %v", resp.Status, tc.expectedStatus)
            }

            // Verify file existence
            filePath := filepath.Join(testDirPath, tc.fileName)
            _, err = os.Stat(filePath)
            fileExists := !os.IsNotExist(err)
            
            if fileExists != tc.shouldExist {
                if tc.shouldExist {
                    t.Errorf("File %s should exist but was removed", tc.fileName)
                } else {
                    t.Errorf("File %s should have been removed but still exists", tc.fileName)
                }
            }
        })
    }

    // Test invalid directory handle
    badReq := &api.RemoveRequest{
        DirectoryHandle: []byte{1, 2, 3}, // Invalid handle
        Name:            "anything.txt",
        Credentials:     creds,
    }
    
    badResp, err := server.Remove(context.Background(), badReq)
    if err != nil {
        t.Fatalf("Remove with invalid handle failed unexpectedly: %v", err)
    }
    
    if badResp.Status != api.Status_ERR_BADHANDLE {
        t.Errorf("Expected ERR_BADHANDLE for invalid handle, got: %v", badResp.Status)
    }
}