// pkg/fs/handle.go
package fs

import (
    "encoding/binary"
    "errors"
    "fmt"
)

// FileHandle is a structured representation of a file handle.
// It contains information to uniquely identify a file in the system.
type FileHandle struct {
    // FileSystemID identifies the specific filesystem
    FileSystemID uint32
    
    // Inode uniquely identifies a file within a filesystem
    Inode uint64
    
    // Generation prevents reuse of inode numbers
    // It's incremented when an inode is reused for a new file
    Generation uint32
}

// Size returns the size of serialized file handle in bytes
func (fh *FileHandle) Size() int {
    return 16 // 4 + 8 + 4 bytes
}

// Serialize converts the file handle to a byte slice
func (fh *FileHandle) Serialize() []byte {
    data := make([]byte, fh.Size())
    
    binary.BigEndian.PutUint32(data[0:4], fh.FileSystemID)
    binary.BigEndian.PutUint64(data[4:12], fh.Inode)
    binary.BigEndian.PutUint32(data[12:16], fh.Generation)
    
    return data
}

// Deserialize parses a byte slice into a file handle
func DeserializeFileHandle(data []byte) (*FileHandle, error) {
    if len(data) < 16 {
        return nil, errors.New("handle data too short")
    }
    
    fh := &FileHandle{
        FileSystemID: binary.BigEndian.Uint32(data[0:4]),
        Inode:        binary.BigEndian.Uint64(data[4:12]),
        Generation:   binary.BigEndian.Uint32(data[12:16]),
    }
    
    return fh, nil
}

// String returns a string representation of the file handle
func (fh *FileHandle) String() string {
    return fmt.Sprintf("FileHandle{FS:%d, Inode:%d, Gen:%d}", 
        fh.FileSystemID, fh.Inode, fh.Generation)
}