// pkg/fs/local/local_fs_test.go
package local

import (
    "context"
    "os"
    "path/filepath"
    "testing"

    "github.com/example/nfsserver/pkg/fs"
)

// setupTestFS creates a temporary directory and initializes a LocalFileSystem
// instance for testing.
func setupTestFS(t *testing.T) (*LocalFileSystem, string, func()) {
    // Create temporary directory
    tempDir, err := os.MkdirTemp("", "localfs-test-")
    if err != nil {
        t.Fatalf("Failed to create temp dir: %v", err)
    }
    
    // Initialize filesystem
    localFS, err := NewLocalFileSystem(tempDir)
    if err != nil {
        os.RemoveAll(tempDir)
        t.Fatalf("Failed to create LocalFileSystem: %v", err)
    }
    
    // Return cleanup function
    cleanup := func() {
        os.RemoveAll(tempDir)
    }
    
    return localFS, tempDir, cleanup
}

// createTestFile creates a test file with the specified content
func createTestFile(t *testing.T, dir, name, content string) string {
    path := filepath.Join(dir, name)
    err := os.WriteFile(path, []byte(content), 0644)
    if err != nil {
        t.Fatalf("Failed to create test file: %v", err)
    }
    return path
}

// createTestDir creates a test directory
func createTestDir(t *testing.T, dir, name string) string {
    path := filepath.Join(dir, name)
    err := os.Mkdir(path, 0755)
    if err != nil {
        t.Fatalf("Failed to create test directory: %v", err)
    }
    return path
}

// TestLocalFileSystem_Interface verifies that LocalFileSystem implements fs.FileSystem
func TestLocalFileSystem_Interface(t *testing.T) {
    var _ fs.FileSystem = (*LocalFileSystem)(nil)
}

// TestLocalFileSystem_Init tests the initialization of the LocalFileSystem
func TestLocalFileSystem_Init(t *testing.T) {
    // Test with a valid directory
    tempDir, err := os.MkdirTemp("", "localfs-init-test-")
    if err != nil {
        t.Fatalf("Failed to create temp dir: %v", err)
    }
    defer os.RemoveAll(tempDir)
    
    _, err = NewLocalFileSystem(tempDir)
    if err != nil {
        t.Errorf("NewLocalFileSystem failed with valid dir: %v", err)
    }
    
    // Test with a non-existent directory
    _, err = NewLocalFileSystem("/path/that/does/not/exist")
    if err == nil {
        t.Error("NewLocalFileSystem should fail with non-existent directory")
    }
    
    // Test with a file instead of a directory
    testFile := filepath.Join(tempDir, "testfile")
    err = os.WriteFile(testFile, []byte("test"), 0644)
    if err != nil {
        t.Fatalf("Failed to create test file: %v", err)
    }
    
    _, err = NewLocalFileSystem(testFile)
    if err == nil {
        t.Error("NewLocalFileSystem should fail with a file path")
    }
}

// TestLocalFileSystem_FileHandles tests file handle generation and resolution
func TestLocalFileSystem_FileHandles(t *testing.T) {
    localFS, _, cleanup := setupTestFS(t)
    defer cleanup()
    
    testPaths := []string{
        "/",
        "/file.txt",
        "/dir/subfile.txt",
        "/dir with spaces/file with spaces.txt",
    }
    
    for _, path := range testPaths {
        // Test path to handle conversion
        handle, err := localFS.PathToFileHandle(path)
        if err != nil {
            t.Errorf("PathToFileHandle failed for %s: %v", path, err)
            continue
        }
        
        if len(handle) == 0 {
            t.Errorf("PathToFileHandle returned empty handle for %s", path)
            continue
        }
        
        // Test handle to path conversion
        resolvedPath, err := localFS.FileHandleToPath(handle)
        if err != nil {
            t.Errorf("FileHandleToPath failed for %s: %v", path, err)
            continue
        }
        
        if resolvedPath != path {
            t.Errorf("Path roundtrip failed: got %s, want %s", resolvedPath, path)
        }
    }
    
    // Test invalid handle
    _, err := localFS.FileHandleToPath([]byte("invalid-handle"))
    if err == nil {
        t.Error("FileHandleToPath should fail with invalid handle")
    }
}

// The following tests are placeholders for future implementation.
// Currently they just verify that the methods properly return "not supported" errors.

func TestLocalFileSystem_GetAttr(t *testing.T) {
    localFS, _, cleanup := setupTestFS(t)
    defer cleanup()
    
    _, err := localFS.GetAttr(context.Background(), "/some/path")
    if err == nil {
        t.Error("Expected error from stub implementation")
    }
}

func TestLocalFileSystem_ReadWrite(t *testing.T) {
    localFS, _, cleanup := setupTestFS(t)
    defer cleanup()
    
    // Test Read
    _, _, err := localFS.Read(context.Background(), "/some/file", 0, 100)
    if err == nil {
        t.Error("Expected error from stub Read implementation")
    }
    
    // Test Write
    _, err = localFS.Write(context.Background(), "/some/file", 0, []byte("test"), false)
    if err == nil {
        t.Error("Expected error from stub Write implementation")
    }
}

func TestLocalFileSystem_DirectoryOps(t *testing.T) {
    localFS, _, cleanup := setupTestFS(t)
    defer cleanup()
    
    // Test ReadDir
    _, _, err := localFS.ReadDir(context.Background(), "/some/dir", 0, 10)
    if err == nil {
        t.Error("Expected error from stub ReadDir implementation")
    }
    
    // Test Mkdir
    _, _, err = localFS.Mkdir(context.Background(), "/some", "dir", fs.FileAttr{})
    if err == nil {
        t.Error("Expected error from stub Mkdir implementation")
    }
    
    // Test Rmdir
    err = localFS.Rmdir(context.Background(), "/some/dir")
    if err == nil {
        t.Error("Expected error from stub Rmdir implementation")
    }
}