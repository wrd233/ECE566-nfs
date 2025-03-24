package fuse

import (
	"time"
	"context"
	"log"

	"bazil.org/fuse"
)

// File represents a file in the filesystem
type File struct {
	fs     *NFSFS  // Reference to the file system
	handle []byte  // NFS file handle for this file
	path   string  // Path for logging/debugging
	size   int64   // File size
}

// Attr sets the attributes of the file
func (f *File) Attr(ctx context.Context, attr *fuse.Attr) error {
	// Get attributes from NFS server
	log.Printf("Getting attributes for file: %s", f.path)
	
	// Default attributes in case of failure
	attr.Mode = 0644
	attr.Size = uint64(f.size)
	attr.Mtime = time.Now()
	
	return nil
}

// ReadAll reads all content from the file
func (f *File) ReadAll(ctx context.Context) ([]byte, error) {
	log.Printf("Reading file: %s", f.path)
	
	// Use NFS client to read the file
	data, _, err := f.fs.client.Read(ctx, f.handle, 0, int(f.size))
	if err != nil {
		log.Printf("Read failed: %v", err)
		return nil, fuse.EIO
	}
	
	return data, nil
}