package local

import (
	"os"
	"path/filepath"
	"testing"
	"syscall"
)

func TestFindPathByInode(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "nfs-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test filesystem
	fs, err := NewLocalFileSystem(tempDir)
	if err != nil {
		t.Fatalf("Failed to create filesystem: %v", err)
	}

	// Create test file
	testFilePath := filepath.Join(tempDir, "testfile.txt")
	if err := os.WriteFile(testFilePath, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Get inode of test file
	fileInfo, err := os.Stat(testFilePath)
	if err != nil {
		t.Fatalf("Failed to stat test file: %v", err)
	}
	stat, ok := fileInfo.Sys().(*syscall.Stat_t)
	if !ok {
		t.Skip("Could not get syscall.Stat_t from FileInfo")
	}
	inode := stat.Ino

	// Test findPathByInode
	path, err := fs.findPathByInode(inode)
	if err != nil {
		t.Fatalf("findPathByInode failed: %v", err)
	}

	// Check result
	expectedPath := "/testfile.txt"
	if path != expectedPath {
		t.Errorf("Path mismatch: got %q, want %q", path, expectedPath)
	}

	// Test with nonexistent inode
	_, err = fs.findPathByInode(999999999)
	if err == nil {
		t.Error("Expected error for nonexistent inode, got nil")
	}
}