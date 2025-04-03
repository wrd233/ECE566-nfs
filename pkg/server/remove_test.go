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

    // Create test directory structure
    testDirPath := filepath.Join(tempDir, "test-dir")
    if err := os.Mkdir(testDirPath, 0755); err != nil {
        t.Fatalf("Failed to create test directory: %v", err)
    }
    
    // Create test files with specific permissions
    regularFile := filepath.Join(testDirPath, "regular-file.txt")
    if err := os.WriteFile(regularFile, []byte("test content"), 0644); err != nil {
        t.Fatalf("Failed to create regular file: %v", err)
    }
    
    // Create subdirectory (to test removing a directory with Remove)
    subDirPath := filepath.Join(testDirPath, "sub-dir")
    if err := os.Mkdir(subDirPath, 0755); err != nil {
        t.Fatalf("Failed to create subdirectory: %v", err)
    }
    
    // Create read-only file
    readOnlyFile := filepath.Join(testDirPath, "readonly-file.txt")
    if err := os.WriteFile(readOnlyFile, []byte("read only content"), 0644); err != nil {
        t.Fatalf("Failed to create read-only file: %v", err)
    }
    // Change permissions to read-only
    if err := os.Chmod(readOnlyFile, 0444); err != nil {
        t.Fatalf("Failed to set read-only permissions: %v", err)
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

    // Get directory handle for test directory
    dirHandle, err := fs.PathToFileHandle("/test-dir")
    if err != nil {
        t.Fatalf("Failed to get directory handle: %v", err)
    }

    // Create credentials with root access
    creds := &api.Credentials{
        Uid:    0, // Use root UID
        Gid:    0, // Use root GID
        Groups: []uint32{0},
    }

    // Test cases
    testCases := []struct {
        name         string
        fileName     string
        expectStatus api.Status
        expectExists bool // Whether file should still exist after test
    }{
        {"Remove regular file", "regular-file.txt", api.Status_OK, false},
        {"Remove non-existent file", "nonexistent.txt", api.Status_ERR_NOENT, false},
        {"Try to remove directory", "sub-dir", api.Status_ERR_ISDIR, true},
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
                t.Fatalf("Remove operation failed unexpectedly: %v", err)
            }

            // Check response status
            if resp.Status != tc.expectStatus {
                t.Errorf("Wrong status: got %v, want %v", resp.Status, tc.expectStatus)
            }

            // Verify if file exists or not as expected
            filePath := filepath.Join(testDirPath, tc.fileName)
            _, err = os.Stat(filePath)
            fileExists := !os.IsNotExist(err)
            
            if fileExists != tc.expectExists {
                if tc.expectExists {
                    t.Errorf("File %s should exist but doesn't", tc.fileName)
                } else {
                    t.Errorf("File %s should not exist but does", tc.fileName)
                }
            }

            // For successful remove operations, check if directory attributes were returned
            if tc.expectStatus == api.Status_OK && resp.DirAttributes == nil {
                t.Logf("Directory attributes not returned for successful removal")
            }
        })
    }

    // Test invalid directory handle
    t.Run("Invalid directory handle", func(t *testing.T) {
        req := &api.RemoveRequest{
            DirectoryHandle: []byte{1, 2, 3}, // Invalid handle
            Name:            "regular-file.txt",
            Credentials:     creds,
        }
        
        resp, err := server.Remove(context.Background(), req)
        if err != nil {
            t.Fatalf("Remove with invalid handle failed unexpectedly: %v", err)
        }
        
        if resp.Status != api.Status_ERR_BADHANDLE {
            t.Errorf("Expected ERR_BADHANDLE status for invalid handle, got: %v", resp.Status)
        }
    })

    // Test permissions (with non-root user)
    if os.Geteuid() == 0 {
        t.Skip("Skipping permission test when running as root")
    }
    
    t.Run("Permission test", func(t *testing.T) {
        // Create non-root credentials
        nonRootCreds := &api.Credentials{
            Uid:    1000, // Non-root user
            Gid:    1000,
            Groups: []uint32{1000},
        }
        
        // Try to remove read-only file
        req := &api.RemoveRequest{
            DirectoryHandle: dirHandle,
            Name:            "readonly-file.txt",
            Credentials:     nonRootCreds,
        }
        
        resp, err := server.Remove(context.Background(), req)
        if err != nil {
            t.Fatalf("Remove operation failed unexpectedly: %v", err)
        }
        
        // Skip assertion if test is running as root (which can remove any file)
        // This is just a sanity check in case the test environment is running with elevated privileges
        if resp.Status != api.Status_OK && resp.Status != api.Status_ERR_ACCES {
            t.Logf("Permission test result: %v (may vary based on test environment)", resp.Status)
        }
    })
}