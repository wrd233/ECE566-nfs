package server

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/example/nfsserver/pkg/api"
	"github.com/example/nfsserver/pkg/fs/local"
)

func TestGetAttr(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "nfs-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test file
	testFilePath := filepath.Join(tempDir, "testfile.txt")
	if err := os.WriteFile(testFilePath, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create filesystem
	fs, err := local.NewLocalFileSystem(tempDir)
	if err != nil {
		t.Fatalf("Failed to create filesystem: %v", err)
	}

	// Create server
	server, err := NewNFSServer(DefaultConfig(), fs)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Get file handle
	fileHandle, err := fs.PathToFileHandle("/testfile.txt")
	if err != nil {
		t.Fatalf("Failed to get file handle: %v", err)
	}

	// Create credentials
	creds := &api.Credentials{
		Uid:    1000,
		Gid:    1000,
		Groups: []uint32{1000},
	}

	// Create request
	req := &api.GetAttrRequest{
		FileHandle:  fileHandle,
		Credentials: creds,
	}

	// Call GetAttr
	resp, err := server.GetAttr(context.Background(), req)
	if err != nil {
		t.Fatalf("GetAttr failed: %v", err)
	}

	// Check response
	if resp.Status != api.Status_OK {
		t.Errorf("Unexpected status: got %v, want OK", resp.Status)
	}
	if resp.Attributes == nil {
		t.Fatal("Attributes is nil")
	}
	if resp.Attributes.Type != api.FileType_REGULAR {
		t.Errorf("Wrong file type: got %v, want REGULAR", resp.Attributes.Type)
	}
	if resp.Attributes.Size != 12 { // "test content" = 12 bytes
		t.Errorf("Wrong size: got %d, want 12", resp.Attributes.Size)
	}
}