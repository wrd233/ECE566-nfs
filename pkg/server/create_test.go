package server

import (
    "context"
    "os"
    "path/filepath"
    "testing"

    "github.com/example/nfsserver/pkg/api"
    "github.com/example/nfsserver/pkg/fs/local"
)

func TestCreate(t *testing.T) {
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

    // Test cases for Create
    testCases := []struct {
        name       string
        fileName   string
        createMode api.CreateMode
        fileShouldExist bool // Whether the file should exist after the test
        expectedStatus api.Status
    }{
        {"UNCHECKED mode new file", "unchecked.txt", api.CreateMode_UNCHECKED, true, api.Status_OK},
        {"GUARDED mode new file", "guarded.txt", api.CreateMode_GUARDED, true, api.Status_OK},
        {"EXCLUSIVE mode new file", "exclusive.txt", api.CreateMode_EXCLUSIVE, true, api.Status_OK},
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            // Create file attributes
            attrs := &api.FileAttributes{
                Mode: 0644,
            }

            // Create request
            req := &api.CreateRequest{
                DirectoryHandle: dirHandle,
                Name:            tc.fileName,
                Credentials:     creds,
                Attributes:      attrs,
                Mode:            tc.createMode,
                Verifier:        12345, // Arbitrary verifier for EXCLUSIVE mode
            }

            // Call Create
            resp, err := server.Create(context.Background(), req)
            if err != nil {
                t.Fatalf("Create failed: %v", err)
            }

            // Check response
            if resp.Status != tc.expectedStatus {
                t.Errorf("Unexpected status: got %v, want %v", resp.Status, tc.expectedStatus)
            }

            // If expected to succeed, verify file handle and attributes
            if tc.expectedStatus == api.Status_OK {
                if resp.FileHandle == nil || len(resp.FileHandle) == 0 {
                    t.Error("FileHandle is empty")
                }
                
                if resp.Attributes == nil {
                    t.Error("Attributes is nil")
                } else if resp.Attributes.Type != api.FileType_REGULAR {
                    t.Errorf("Wrong file type: got %v, want REGULAR", resp.Attributes.Type)
                }
            }

            // Verify file exists or not as expected
            filePath := filepath.Join(testDirPath, tc.fileName)
            _, err = os.Stat(filePath)
            fileExists := !os.IsNotExist(err)
            
            if fileExists != tc.fileShouldExist {
                if tc.fileShouldExist {
                    t.Errorf("File %s should exist but doesn't", tc.fileName)
                } else {
                    t.Errorf("File %s should not exist but does", tc.fileName)
                }
            }
        })
    }

    // Test GUARDED mode with existing file
    existingFilePath := filepath.Join(testDirPath, "existing.txt")
    if err := os.WriteFile(existingFilePath, []byte("existing content"), 0644); err != nil {
        t.Fatalf("Failed to create existing file: %v", err)
    }

    // Create request for existing file with GUARDED mode
    guardedReq := &api.CreateRequest{
        DirectoryHandle: dirHandle,
        Name:            "existing.txt",
        Credentials:     creds,
        Attributes:      &api.FileAttributes{Mode: 0644},
        Mode:            api.CreateMode_GUARDED,
    }

    // Call Create
    guardedResp, err := server.Create(context.Background(), guardedReq)
    if err != nil {
        t.Fatalf("Create with GUARDED mode failed unexpectedly: %v", err)
    }

    // Check response - should fail with ERR_EXIST
    if guardedResp.Status != api.Status_ERR_EXIST {
        t.Errorf("Expected ERR_EXIST for GUARDED mode on existing file, got: %v", guardedResp.Status)
    }

    // Test EXCLUSIVE mode with same verifier (should return same file handle)
    createExclReq1 := &api.CreateRequest{
        DirectoryHandle: dirHandle,
        Name:            "exclusive_idempotent.txt",
        Credentials:     creds,
        Attributes:      &api.FileAttributes{Mode: 0644},
        Mode:            api.CreateMode_EXCLUSIVE,
        Verifier:        67890,
    }

    // First create
    resp1, err := server.Create(context.Background(), createExclReq1)
    if err != nil || resp1.Status != api.Status_OK {
        t.Fatalf("First exclusive create failed: %v, status: %v", err, resp1.Status)
    }

    // Second create with same parameters
    resp2, err := server.Create(context.Background(), createExclReq1)
    if err != nil || resp2.Status != api.Status_OK {
        t.Fatalf("Second exclusive create failed: %v, status: %v", err, resp2.Status)
    }

    // FileHandles should be identical
    if string(resp1.FileHandle) != string(resp2.FileHandle) {
        t.Error("File handles differ for idempotent EXCLUSIVE create")
    }
}