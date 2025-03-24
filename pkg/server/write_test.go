package server

import (
    "context"
    "os"
    "path/filepath"
    "testing"

    "github.com/example/nfsserver/pkg/api"
    "github.com/example/nfsserver/pkg/fs/local"
)

func TestWrite(t *testing.T) {
    // Create temporary directory
    tempDir, err := os.MkdirTemp("", "nfs-test-")
    if err != nil {
        t.Fatalf("Failed to create temp dir: %v", err)
    }
    defer os.RemoveAll(tempDir)

    // Create empty test file
    testFilePath := filepath.Join(tempDir, "testfile.txt")
    if err := os.WriteFile(testFilePath, []byte(""), 0666); err != nil {
        t.Fatalf("Failed to create test file: %v", err)
    }

	if err := os.Chmod(testFilePath, 0666); err != nil {
		t.Fatalf("Failed to set file permissions: %v", err)
	}

    // Create filesystem
    fs, err := local.NewLocalFileSystem(tempDir)
    if err != nil {
        t.Fatalf("Failed to create filesystem: %v", err)
    }

    // Create server with root squashing disabled
    config := DefaultConfig()
    config.EnableRootSquash = false // Disable root squashing for test
    config.MaxWriteSize = 1024 * 1024 // 1MB max write size
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
        name      string
        offset    uint64
        data      string
        stability uint32
    }{
        {"Write at beginning", 0, "Hello", 0},
        {"Append to file", 5, ", NFS!", 1},
        {"Overwrite middle", 7, "NFS", 2},
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            // Create write request
            req := &api.WriteRequest{
                FileHandle:  fileHandle,
                Credentials: creds,
                Offset:      tc.offset,
                Data:        []byte(tc.data),
                Stability:   tc.stability,
            }

            // Call Write
            resp, err := server.Write(context.Background(), req)
            if err != nil {
                t.Fatalf("Write failed: %v", err)
            }

            // Check response
            if resp.Status != api.Status_OK {
                t.Errorf("Unexpected status: got %v, want OK", resp.Status)
            }

            if resp.Count != uint32(len(tc.data)) {
                t.Errorf("Wrong byte count: got %d, want %d", resp.Count, len(tc.data))
            }

            if resp.Stability != tc.stability {
                t.Errorf("Wrong stability level: got %d, want %d", resp.Stability, tc.stability)
            }
        })
    }

    // Read the final content to verify writes
    content, err := os.ReadFile(testFilePath)
    if err != nil {
        t.Fatalf("Failed to read test file: %v", err)
    }

    expected := "Hello, NFS!"
    if string(content) != expected {
        t.Errorf("File content mismatch: got %q, want %q", string(content), expected)
    }

    // Test idempotent writes
    // Make the same write request twice and check that it works correctly
    req := &api.WriteRequest{
        FileHandle:  fileHandle,
        Credentials: creds,
        Offset:      0,
        Data:        []byte("Idempotent"),
        Stability:   2,
    }

    // First write
    resp1, err := server.Write(context.Background(), req)
    if err != nil || resp1.Status != api.Status_OK {
        t.Fatalf("First idempotent write failed: %v, status: %v", err, resp1.Status)
    }

    // Second write (identical request)
    resp2, err := server.Write(context.Background(), req)
    if err != nil || resp2.Status != api.Status_OK {
        t.Fatalf("Second idempotent write failed: %v, status: %v", err, resp2.Status)
    }

    // The responses should be consistent
    if resp1.Count != resp2.Count {
        t.Errorf("Idempotent writes returned different byte counts: %d vs %d", 
            resp1.Count, resp2.Count)
    }

    // Test write size limit
    bigData := make([]byte, config.MaxWriteSize+1)
    reqBig := &api.WriteRequest{
        FileHandle:  fileHandle,
        Credentials: creds,
        Offset:      0,
        Data:        bigData,
        Stability:   0,
    }

    respBig, err := server.Write(context.Background(), reqBig)
    if err != nil {
        t.Fatalf("Write with big data failed unexpectedly: %v", err)
    }

    if respBig.Status != api.Status_ERR_FBIG {
        t.Errorf("Expected ERR_FBIG status for oversized write, got: %v", respBig.Status)
    }

    // Test invalid file handle
    reqInv := &api.WriteRequest{
        FileHandle:  []byte{1, 2, 3}, // Invalid handle
        Credentials: creds,
        Offset:      0,
        Data:        []byte("test"),
        Stability:   0,
    }

    respInv, err := server.Write(context.Background(), reqInv)
    if err != nil {
        t.Fatalf("Write with invalid handle failed unexpectedly: %v", err)
    }

    if respInv.Status != api.Status_ERR_BADHANDLE {
        t.Errorf("Expected ERR_BADHANDLE status for invalid handle, got: %v", respInv.Status)
    }
}