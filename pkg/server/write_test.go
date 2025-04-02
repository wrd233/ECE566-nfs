package server

import (
    "context"
    "os"
    "path/filepath"
    "testing"
    "time"
    "fmt"
    "bytes"

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



func TestWriteStabilityPerformance(t *testing.T) {
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
    config.EnableRootSquash = false
    server, err := NewNFSServer(config, fs)
    if err != nil {
        t.Fatalf("Failed to create server: %v", err)
    }

    // Create test parameters
    numWrites := 100
    writeSize := 1024 // 1KB per write
    creds := &api.Credentials{
        Uid:    0,
        Gid:    0,
        Groups: []uint32{0},
    }

    // Test function to perform writes with specified stability
    performWrites := func(filePath string, stability uint32) (time.Duration, error) {
        // Create test file with explicit permissions
        testFilePath := filepath.Join(tempDir, filePath)
        if err := os.WriteFile(testFilePath, []byte{}, 0666); err != nil {
            return 0, fmt.Errorf("failed to create test file: %v", err)
        }
        
        // Explicitly set file permissions to ensure writability
        if err := os.Chmod(testFilePath, 0666); err != nil {
            return 0, fmt.Errorf("failed to set file permissions: %v", err)
        }

        // Get file handle
        fileHandle, err := fs.PathToFileHandle("/" + filePath)
        if err != nil {
            return 0, fmt.Errorf("failed to get file handle: %v", err)
        }

        // Generate test data
        data := make([]byte, writeSize)
        for i := range data {
            data[i] = byte(i % 256)
        }

        // Perform writes and measure time
        start := time.Now()
        
        for i := 0; i < numWrites; i++ {
            // Create write request
            req := &api.WriteRequest{
                FileHandle:  fileHandle,
                Credentials: creds,
                Offset:      uint64(i * writeSize),
                Data:        data,
                Stability:   stability,
            }

            // Call Write
            resp, err := server.Write(context.Background(), req)
            if err != nil || resp.Status != api.Status_OK {
                return 0, fmt.Errorf("write failed: %v, status: %v", err, resp.Status)
            }
        }
        
        elapsed := time.Since(start)
        return elapsed, nil
    }

    // Test 1: UNSTABLE writes followed by a COMMIT
    unstableFile := "unstable_test.dat"
    unstableTime, err := performWrites(unstableFile, 0) // 0 = UNSTABLE
    if err != nil {
        t.Fatalf("UNSTABLE write test failed: %v", err)
    }

    // Get file handle for commit
    unstableHandle, err := fs.PathToFileHandle("/" + unstableFile)
    if err != nil {
        t.Fatalf("Failed to get file handle: %v", err)
    }

    // Measure time for commit
    commitStart := time.Now()
    commitReq := &api.CommitRequest{
        FileHandle:    unstableHandle,
        Credentials:   creds,
        WriteVerifier: server.writeVerifier, // Access the server's verifier
    }
    
    commitResp, err := server.Commit(context.Background(), commitReq)
    commitTime := time.Since(commitStart)
    
    if err != nil || commitResp.Status != api.Status_OK {
        t.Fatalf("Commit failed: %v, status: %v", err, commitResp.Status)
    }

    // Total time for UNSTABLE + COMMIT
    totalUnstableTime := unstableTime + commitTime

    // Test 2: FILE_SYNC writes
    syncFile := "sync_test.dat"
    syncTime, err := performWrites(syncFile, 2) // 2 = FILE_SYNC
    if err != nil {
        t.Fatalf("FILE_SYNC write test failed: %v", err)
    }

    // Verify both files have the same content
    unstableContent, err := os.ReadFile(filepath.Join(tempDir, unstableFile))
    if err != nil {
        t.Fatalf("Failed to read unstable file: %v", err)
    }
    
    syncContent, err := os.ReadFile(filepath.Join(tempDir, syncFile))
    if err != nil {
        t.Fatalf("Failed to read sync file: %v", err)
    }
    
    if !bytes.Equal(unstableContent, syncContent) {
        t.Fatalf("File contents differ: unstable size=%d, sync size=%d", 
            len(unstableContent), len(syncContent))
    }

    // Log results
    t.Logf("Performance comparison for %d writes of %d bytes each:", numWrites, writeSize)
    t.Logf("UNSTABLE writes: %v", unstableTime)
    t.Logf("Final COMMIT: %v", commitTime)
    t.Logf("Total UNSTABLE+COMMIT: %v", totalUnstableTime)
    t.Logf("FILE_SYNC writes: %v", syncTime)
    t.Logf("Performance difference: FILE_SYNC takes %.2fx longer than UNSTABLE+COMMIT", 
        float64(syncTime)/float64(totalUnstableTime))
    
    // Optional assertion - in most environments, FILE_SYNC should be slower
    if syncTime <= totalUnstableTime {
        t.Logf("Warning: Expected FILE_SYNC to be slower than UNSTABLE+COMMIT, but it wasn't." +
               "This might happen in environments with fast storage or filesystem caching.")
    }
}