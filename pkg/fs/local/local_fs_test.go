// pkg/fs/local/local_fs_test.go
package local

import (
    "context"
    "errors"
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

// Additional test for structured file handle
func TestStructuredFileHandle(t *testing.T) {
    // Test serialization and deserialization
    original := &fs.FileHandle{
        FileSystemID: 123,
        Inode:        456789,
        Generation:   2,
    }
    
    // Serialize
    data := original.Serialize()
    
    // Deserialize
    parsed, err := fs.DeserializeFileHandle(data)
    if err != nil {
        t.Errorf("Failed to deserialize handle: %v", err)
    }
    
    // Compare
    if parsed.FileSystemID != original.FileSystemID {
        t.Errorf("FileSystemID mismatch: got %d, want %d", 
            parsed.FileSystemID, original.FileSystemID)
    }
    
    if parsed.Inode != original.Inode {
        t.Errorf("Inode mismatch: got %d, want %d", 
            parsed.Inode, original.Inode)
    }
    
    if parsed.Generation != original.Generation {
        t.Errorf("Generation mismatch: got %d, want %d", 
            parsed.Generation, original.Generation)
    }
}

// Also update TestLocalFileSystem_FileHandles to create real files
func TestLocalFileSystem_FileHandles(t *testing.T) {
    localFS, tempDir, cleanup := setupTestFS(t)
    defer cleanup()
    
    // Create test files
    testFiles := []string{
        "file1.txt",
        "dir1/file2.txt",
        "dir with spaces/file with spaces.txt",
    }
    
    for _, file := range testFiles {
        dir := filepath.Dir(file)
        if dir != "." {
            dirPath := filepath.Join(tempDir, dir)
            if err := os.MkdirAll(dirPath, 0755); err != nil {
                t.Fatalf("Failed to create directory %s: %v", dir, err)
            }
        }
        
        filePath := filepath.Join(tempDir, file)
        if err := os.WriteFile(filePath, []byte("test content"), 0644); err != nil {
            t.Fatalf("Failed to create test file %s: %v", file, err)
        }
    }
    
    for _, file := range testFiles {
        relPath := "/" + file
        
        // Test path to handle conversion
        handle, err := localFS.PathToFileHandle(relPath)
        if err != nil {
            t.Errorf("PathToFileHandle failed for %s: %v", relPath, err)
            continue
        }
        
        if len(handle) == 0 {
            t.Errorf("PathToFileHandle returned empty handle for %s", relPath)
            continue
        }
        
        // Parse and verify handle
        parsedHandle, err := fs.DeserializeFileHandle(handle)
        if err != nil {
            t.Errorf("Failed to parse handle for %s: %v", relPath, err)
            continue
        }
        
        if parsedHandle.FileSystemID != localFS.fsID {
            t.Errorf("FileSystemID mismatch for %s: got %d, want %d", 
                relPath, parsedHandle.FileSystemID, localFS.fsID)
        }
        
        // Test handle to path conversion
        resolvedPath, err := localFS.FileHandleToPath(handle)
        if err != nil {
            t.Errorf("FileHandleToPath failed for %s: %v", relPath, err)
            continue
        }
        
        if resolvedPath != relPath {
            t.Errorf("Path roundtrip failed: got %s, want %s", resolvedPath, relPath)
        }
    }
    
    // Test invalid handle
    _, err := localFS.FileHandleToPath([]byte("invalid-handle"))
    if err == nil {
        t.Error("FileHandleToPath should fail with invalid handle")
    }
}


// TestGetAttr tests the GetAttr method
func TestGetAttr(t *testing.T) {
    localFS, tempDir, cleanup := setupTestFS(t)
    defer cleanup()
    
    // Create test file
    testFile := createTestFile(t, tempDir, "test.txt", "test content")
    relPath := "/" + filepath.Base(testFile)
    
    // Get attributes
    info, err := localFS.GetAttr(context.Background(), relPath)
    if err != nil {
        t.Fatalf("GetAttr failed: %v", err)
    }
    
    // Verify file type
    if info.Type != fs.FileTypeRegular {
        t.Errorf("Wrong file type: got %v, want %v", info.Type, fs.FileTypeRegular)
    }
    
    // Verify size
    expectedSize := int64(len("test content"))
    if info.Size != expectedSize {
        t.Errorf("Wrong file size: got %d, want %d", info.Size, expectedSize)
    }
    
    // Create test directory
    testDir := createTestDir(t, tempDir, "testdir")
    relDirPath := "/" + filepath.Base(testDir)
    
    // Get attributes for directory
    dirInfo, err := localFS.GetAttr(context.Background(), relDirPath)
    if err != nil {
        t.Fatalf("GetAttr failed for directory: %v", err)
    }
    
    // Verify directory type
    if dirInfo.Type != fs.FileTypeDirectory {
        t.Errorf("Wrong file type for directory: got %v, want %v", 
            dirInfo.Type, fs.FileTypeDirectory)
    }
    
    // Test non-existent file
    _, err = localFS.GetAttr(context.Background(), "/nonexistent")
    if err == nil {
        t.Error("GetAttr should fail for non-existent file")
    } else if !errors.Is(err, fs.ErrNotExist) {
        t.Errorf("Wrong error type: got %v, want %v", err, fs.ErrNotExist)
    }
}

// TestConvertFileInfo tests the convertFileInfo method
func TestConvertFileInfo(t *testing.T) {
    localFS, tempDir, cleanup := setupTestFS(t)
    defer cleanup()
    
    // Create test file
    testFile := createTestFile(t, tempDir, "convert.txt", "content for conversion")
    relPath := "/" + filepath.Base(testFile)
    
    // Get os.FileInfo
    fullPath := filepath.Join(tempDir, filepath.Base(testFile))
    osInfo, err := os.Stat(fullPath)
    if err != nil {
        t.Fatalf("Failed to stat test file: %v", err)
    }
    
    // Convert to fs.FileInfo
    fsInfo, err := localFS.convertFileInfo(relPath, osInfo)
    if err != nil {
        t.Fatalf("convertFileInfo failed: %v", err)
    }
    
    // Verify basic properties
    if fsInfo.Size != osInfo.Size() {
        t.Errorf("Size mismatch: got %d, want %d", fsInfo.Size, osInfo.Size())
    }
    
    if !fsInfo.ModifyTime.Equal(osInfo.ModTime()) {
        t.Errorf("ModTime mismatch: got %v, want %v", fsInfo.ModifyTime, osInfo.ModTime())
    }
    
    // Verify type
    if fsInfo.Type != fs.FileTypeRegular {
        t.Errorf("Type mismatch: got %v, want %v", fsInfo.Type, fs.FileTypeRegular)
    }
    
    // Create directory and test conversion
    testDir := createTestDir(t, tempDir, "convertdir")
    relDirPath := "/" + filepath.Base(testDir)
    
    // Get os.FileInfo for directory
    osDirInfo, err := os.Stat(testDir)
    if err != nil {
        t.Fatalf("Failed to stat test directory: %v", err)
    }
    
    // Convert directory info
    fsDirInfo, err := localFS.convertFileInfo(relDirPath, osDirInfo)
    if err != nil {
        t.Fatalf("convertFileInfo failed for directory: %v", err)
    }
    
    // Verify directory type
    if fsDirInfo.Type != fs.FileTypeDirectory {
        t.Errorf("Type mismatch for directory: got %v, want %v", 
            fsDirInfo.Type, fs.FileTypeDirectory)
    }
}

// TestResolvePath tests the path resolution and security
func TestResolvePath(t *testing.T) {
    localFS, tempDir, cleanup := setupTestFS(t)
    defer cleanup()
    
    // Test normal path
    path := "/normal/path"
    expected := filepath.Join(tempDir, "normal", "path")
    result, err := localFS.resolvePath(path)
    if err != nil {
        t.Errorf("resolvePath failed for normal path: %v", err)
    } else if result != expected {
        t.Errorf("resolvePath result mismatch: got %q, want %q", result, expected)
    }
    
    // Test path with leading slash
    path = "no/leading/slash"
    expected = filepath.Join(tempDir, "no", "leading", "slash")
    result, err = localFS.resolvePath(path)
    if err != nil {
        t.Errorf("resolvePath failed for path without leading slash: %v", err)
    } else if result != expected {
        t.Errorf("resolvePath result mismatch: got %q, want %q", result, expected)
    }
    
    // Test directory traversal attempt
    path = "/../../../etc/passwd"
    _, err = localFS.resolvePath(path)
    if err == nil {
        t.Error("resolvePath should reject directory traversal attempts")
    }
    
    // Test path with dot segments
    path = "/a/./b/../c"
    expected = filepath.Join(tempDir, "a", "c")
    result, err = localFS.resolvePath(path)
    if err != nil {
        t.Errorf("resolvePath failed for path with dot segments: %v", err)
    } else if result != expected {
        t.Errorf("resolvePath result mismatch: got %q, want %q", result, expected)
    }
}


// TestLookup tests the Lookup method
func TestLookup(t *testing.T) {
    localFS, tempDir, cleanup := setupTestFS(t)
    defer cleanup()
    
    // Create test directory
    testDirName := "testdir"
    testDir := createTestDir(t, tempDir, testDirName)
    
    // Create test file inside directory
    testFileName := "testfile.txt"
    testContent := "test content"
    _ = createTestFile(t, testDir, testFileName, testContent)
    
    // Test looking up the file in the directory
    filePath, fileInfo, err := localFS.Lookup(context.Background(), "/"+testDirName, testFileName)
    if err != nil {
        t.Fatalf("Lookup failed: %v", err)
    }
    
    // Check path
    expectedPath := "/" + testDirName + "/" + testFileName
    if filePath != expectedPath {
        t.Errorf("Wrong path: got %q, want %q", filePath, expectedPath)
    }
    
    // Check file type
    if fileInfo.Type != fs.FileTypeRegular {
        t.Errorf("Wrong file type: got %v, want %v", fileInfo.Type, fs.FileTypeRegular)
    }
    
    // Check size
    if fileInfo.Size != int64(len(testContent)) {
        t.Errorf("Wrong size: got %d, want %d", fileInfo.Size, len(testContent))
    }
    
    // Test looking up non-existent file
    _, _, err = localFS.Lookup(context.Background(), "/"+testDirName, "nonexistent.txt")
    if err == nil {
        t.Error("Lookup should fail for non-existent file")
    }
    
    // Test looking up in non-existent directory
    _, _, err = localFS.Lookup(context.Background(), "/nonexistent", testFileName)
    if err == nil {
        t.Error("Lookup should fail for non-existent directory")
    }
    
    // Test looking up in a file (not a directory)
    _, _, err = localFS.Lookup(context.Background(), "/"+testDirName+"/"+testFileName, "anything")
    if err == nil {
        t.Error("Lookup should fail when directory is actually a file")
    }
}

// TestRead tests the Read method
func TestRead(t *testing.T) {
    localFS, tempDir, cleanup := setupTestFS(t)
    defer cleanup()
    
    // Create test file
    testFileName := "testfile.txt"
    testContent := "abcdefghijklmnopqrstuvwxyz"
    _ = createTestFile(t, tempDir, testFileName, testContent)
    
    // Test reading entire file
    data, eof, err := localFS.Read(context.Background(), "/"+testFileName, 0, len(testContent))
    if err != nil {
        t.Fatalf("Read failed: %v", err)
    }
    
    if string(data) != testContent {
        t.Errorf("Read content mismatch: got %q, want %q", string(data), testContent)
    }
    
    if !eof {
        t.Error("Expected EOF when reading to end of file")
    }
    
    // Test reading part of file
    data, eof, err = localFS.Read(context.Background(), "/"+testFileName, 3, 5)
    if err != nil {
        t.Fatalf("Partial read failed: %v", err)
    }
    
    if string(data) != "defgh" {
        t.Errorf("Partial read content mismatch: got %q, want %q", string(data), "defgh")
    }
    
    if eof {
        t.Error("Shouldn't get EOF when reading part of file")
    }
    
    // Test reading past end of file
    data, eof, err = localFS.Read(context.Background(), "/"+testFileName, 20, 10)
    if err != nil {
        t.Fatalf("Read past EOF failed: %v", err)
    }
    
    if string(data) != "uvwxyz" {
        t.Errorf("Read past EOF content mismatch: got %q, want %q", string(data), "uvwxyz")
    }
    
    if !eof {
        t.Error("Expected EOF when reading past end of file")
    }
    
    // Test reading from offset beyond file size
    data, eof, err = localFS.Read(context.Background(), "/"+testFileName, 30, 10)
    if err != nil {
        t.Fatalf("Read from beyond file size failed: %v", err)
    }
    
    if len(data) != 0 {
        t.Errorf("Read from beyond file size returned data: %q", string(data))
    }
    
    if !eof {
        t.Error("Expected EOF when reading from beyond file size")
    }
    
    // Test reading non-existent file
    _, _, err = localFS.Read(context.Background(), "/nonexistent.txt", 0, 10)
    if err == nil {
        t.Error("Read should fail for non-existent file")
    }
    
    // Test reading directory
    _ = createTestDir(t, tempDir, "testdir")
    _, _, err = localFS.Read(context.Background(), "/testdir", 0, 10)
    if err == nil || !errors.Is(err, fs.ErrIsDir) {
        t.Errorf("Read should fail with IsDir error for directory, got: %v", err)
    }
}

// TestWrite tests the Write method
func TestWrite(t *testing.T) {
    localFS, tempDir, cleanup := setupTestFS(t)
    defer cleanup()
    
    // Create empty test file
    testFileName := "writefile.txt"
    emptyFile := createTestFile(t, tempDir, testFileName, "")
    
    // Test writing to file
    testData := "Hello, World!"
    bytesWritten, err := localFS.Write(context.Background(), "/"+testFileName, 0, []byte(testData), false)
    if err != nil {
        t.Fatalf("Write failed: %v", err)
    }
    
    if bytesWritten != len(testData) {
        t.Errorf("Wrong bytes written: got %d, want %d", bytesWritten, len(testData))
    }
    
    // Read file content to verify
    content, err := os.ReadFile(emptyFile)
    if err != nil {
        t.Fatalf("Failed to read file content: %v", err)
    }
    
    if string(content) != testData {
        t.Errorf("File content mismatch: got %q, want %q", string(content), testData)
    }
    
    // Test writing at offset
    moreData := "Additional data"
    bytesWritten, err = localFS.Write(context.Background(), "/"+testFileName, 7, []byte(moreData), false)
    if err != nil {
        t.Fatalf("Write at offset failed: %v", err)
    }
    
    if bytesWritten != len(moreData) {
        t.Errorf("Wrong bytes written at offset: got %d, want %d", bytesWritten, len(moreData))
    }
    
    // Read updated content
    content, err = os.ReadFile(emptyFile)
    if err != nil {
        t.Fatalf("Failed to read updated file content: %v", err)
    }
    
    expectedContent := "Hello, Additional data"
    if string(content) != expectedContent {
        t.Errorf("Updated content mismatch: got %q, want %q", string(content), expectedContent)
    }
    
    // Test writing to non-existent file
    _, err = localFS.Write(context.Background(), "/nonexistent.txt", 0, []byte("test"), false)
    if err == nil {
        t.Error("Write should fail for non-existent file")
    }
    
    // Test writing to directory
    _ = createTestDir(t, tempDir, "writedir")
    _, err = localFS.Write(context.Background(), "/writedir", 0, []byte("test"), false)
    if err == nil || !errors.Is(err, fs.ErrIsDir) {
        t.Errorf("Write should fail with IsDir error for directory, got: %v", err)
    }
    
    // Test sync write (just check that it doesn't error)
    _, err = localFS.Write(context.Background(), "/"+testFileName, 0, []byte("Sync write"), true)
    if err != nil {
        t.Errorf("Sync write failed: %v", err)
    }
}