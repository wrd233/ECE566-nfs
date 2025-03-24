package server

import (
    "context"
    "os"
    "path/filepath"
    "testing"

    "github.com/example/nfsserver/pkg/api"
    "github.com/example/nfsserver/pkg/fs/local"
)

func TestRead(t *testing.T) {
    // Create temporary directory
    tempDir, err := os.MkdirTemp("", "nfs-test-")
    if err != nil {
        t.Fatalf("Failed to create temp dir: %v", err)
    }
    defer os.RemoveAll(tempDir)

    // Create test file with content
    testContent := "This is test content for read operation testing."
    testFilePath := filepath.Join(tempDir, "testfile.txt")
    if err := os.WriteFile(testFilePath, []byte(testContent), 0666); err != nil {
        t.Fatalf("Failed to create test file: %v", err)
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

    // Get file handle
    fileHandle, err := fs.PathToFileHandle("/testfile.txt")
    if err != nil {
        t.Fatalf("Failed to get file handle: %v", err)
    }

    // Create credentials with root access
    creds := &api.Credentials{
        Uid:    0, // Use root UID
        Gid:    0, // Use root GID
        Groups: []uint32{0},
    }

    // Test cases
    testCases := []struct {
        name     string
        offset   uint64
        count    uint32
        wantLen  int
        wantEof  bool
        wantData string
    }{
        {"Read from start", 0, 10, 10, false, "This is te"},
        {"Read middle", 5, 5, 5, false, "is te"},
        {"Read to end", 30, 50, 18, true, "operation testing."},
        {"Read past end", 100, 10, 0, true, ""},
        {"Read zero bytes", 0, 0, 0, false, ""},
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            // Create read request
            req := &api.ReadRequest{
                FileHandle:  fileHandle,
                Credentials: creds,
                Offset:      tc.offset,
                Count:       tc.count,
            }

            // Call Read
            resp, err := server.Read(context.Background(), req)
            if err != nil {
                t.Fatalf("Read failed: %v", err)
            }

            // Check response
            if resp.Status != api.Status_OK {
                t.Errorf("Unexpected status: got %v, want OK", resp.Status)
            }

            if len(resp.Data) != tc.wantLen {
                t.Errorf("Wrong data length: got %d, want %d", len(resp.Data), tc.wantLen)
            }

            if resp.Eof != tc.wantEof {
                t.Errorf("Wrong EOF flag: got %v, want %v", resp.Eof, tc.wantEof)
            }

            if string(resp.Data) != tc.wantData {
                t.Errorf("Wrong data content: got %q, want %q", string(resp.Data), tc.wantData)
            }
        })
    }

    // Test invalid file handle
    req := &api.ReadRequest{
        FileHandle:  []byte{1, 2, 3}, // Invalid handle
        Credentials: creds,
        Offset:      0,
        Count:       10,
    }

    resp, err := server.Read(context.Background(), req)
    if err != nil {
        t.Fatalf("Read with invalid handle failed unexpectedly: %v", err)
    }

    if resp.Status != api.Status_ERR_BADHANDLE {
        t.Errorf("Expected ERR_BADHANDLE status for invalid handle, got: %v", resp.Status)
    }
}