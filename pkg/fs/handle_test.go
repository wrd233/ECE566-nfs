package fs

import (
	"testing"
)

func TestFileHandleSerializeDeserialize(t *testing.T) {
	// Create sample file handle
	original := &FileHandle{
		FileSystemID: 12345,
		Inode:        67890,
		Generation:   42,
	}

	// Serialize
	data := original.Serialize()

	// Check length
	if len(data) != 16 {
		t.Errorf("Serialized handle length wrong: got %d, want 16", len(data))
	}

	// Deserialize
	recovered, err := DeserializeFileHandle(data)
	if err != nil {
		t.Fatalf("Failed to deserialize: %v", err)
	}

	// Check fields match
	if recovered.FileSystemID != original.FileSystemID {
		t.Errorf("FileSystemID mismatch: got %d, want %d", 
			recovered.FileSystemID, original.FileSystemID)
	}
	if recovered.Inode != original.Inode {
		t.Errorf("Inode mismatch: got %d, want %d", 
			recovered.Inode, original.Inode)
	}
	if recovered.Generation != original.Generation {
		t.Errorf("Generation mismatch: got %d, want %d", 
			recovered.Generation, original.Generation)
	}
}

func TestDeserializeInvalidHandle(t *testing.T) {
	// Test with too short data
	_, err := DeserializeFileHandle([]byte{1, 2, 3})
	if err == nil {
		t.Error("Expected error for too short data, got nil")
	}
}