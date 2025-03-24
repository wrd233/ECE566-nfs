package fuse

import (
	"time"
	"context"
	"log"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
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
	log.Printf("Reading file: %s (size: %d bytes)", f.path, f.size)
	
	// Use NFS client to read the file
	data, _, err := f.fs.client.Read(ctx, f.handle, 0, int(f.size))
	if err != nil {
		log.Printf("Read failed: %v", err)
		return nil, fuse.EIO
	}
	
	return data, nil
}

// Open implements the Open method for FUSE files
func (f *File) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (fs.Handle, error) {
    log.Printf("Opening file: %s (flags: %v)", f.path, req.Flags)
    
    // Set direct IO flag to avoid kernel caching
    // This ensures we don't have to implement Fsync() method
    resp.Flags |= fuse.OpenDirectIO
    
    // Return the file as its own handle
    return f, nil
}

// Read implements the Read method for FUSE files
func (f *File) Read(ctx context.Context, req *fuse.ReadRequest, resp *fuse.ReadResponse) error {
    log.Printf("Reading file: %s (offset: %d, size: %d)", f.path, req.Offset, req.Size)
    
    // Use NFS client to read the file at the requested offset
    data, _, err := f.fs.client.Read(ctx, f.handle, req.Offset, req.Size)
    if err != nil {
        log.Printf("Read failed: %v", err)
        return fuse.EIO
    }
    
    resp.Data = data
    return nil
}

// Write implements the Write method for FUSE files
func (f *File) Write(ctx context.Context, req *fuse.WriteRequest, resp *fuse.WriteResponse) error {
    log.Printf("Writing to file: %s (offset: %d, size: %d bytes)", f.path, req.Offset, len(req.Data))
    
    // Use NFS client to write the data
    // Use FILE_SYNC stability level (2) for safety
    count, err := f.fs.client.Write(ctx, f.handle, req.Offset, req.Data, 2)
    if err != nil {
        log.Printf("Write failed: %v", err)
        return fuse.EIO
    }
    
    // Update response with bytes written
    resp.Size = count
    
    // Update file size if needed
    newSize := req.Offset + int64(count)
    if newSize > f.size {
        f.size = newSize
    }
    
    return nil
}

// Flush implements the Flush method for FUSE files
func (f *File) Flush(ctx context.Context, req *fuse.FlushRequest) error {
    log.Printf("Flushing file: %s", f.path)
    // In our implementation, writes are already synced to the server
    // with FILE_SYNC stability, so we don't need additional action here
    return nil
}