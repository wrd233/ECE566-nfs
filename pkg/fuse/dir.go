package fuse

import (
	"os"
	"time"
	"context"
	"log"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/example/nfsserver/pkg/api"
)

// Dir represents a directory in the filesystem
type Dir struct {
	fs     *NFSFS        // Reference to the file system
	handle []byte        // NFS file handle for this directory
	path   string        // Path for logging/debugging
}

// Attr sets the attributes of the directory
func (d *Dir) Attr(ctx context.Context, attr *fuse.Attr) error {
	// Get attributes from NFS server
	log.Printf("Getting attributes for directory: %s", d.path)
	
	// Default attributes in case of failure
	attr.Mode = os.ModeDir | 0755
	attr.Mtime = time.Now()
	
	return nil
}

// Lookup looks up a specific entry in the directory
func (d *Dir) Lookup(ctx context.Context, name string) (fs.Node, error) {
	log.Printf("Looking up %s in directory %s", name, d.path)
	
	// Use NFS client to lookup the file
	fileHandle, attrs, err := d.fs.client.Lookup(ctx, d.handle, name)
	if err != nil {
		log.Printf("Lookup failed: %v", err)
		return nil, fuse.ENOENT
	}
	
	// Determine if it's a file or directory
	if attrs.Type == api.FileType_DIRECTORY {
		return &Dir{
			fs:     d.fs,
			handle: fileHandle,
			path:   d.path + "/" + name,
		}, nil
	} else {
		return &File{
			fs:     d.fs,
			handle: fileHandle,
			path:   d.path + "/" + name,
			size:   int64(attrs.Size),
		}, nil
	}
}

// ReadDirAll returns all entries in the directory
func (d *Dir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	log.Printf("Reading directory: %s", d.path)
	
	// Use NFS client to read directory
	entries, err := d.fs.client.ReadDir(ctx, d.handle)
	if err != nil {
		log.Printf("ReadDir failed: %v", err)
		return nil, fuse.EIO
	}
	
	// Convert NFS entries to FUSE dirents
	result := make([]fuse.Dirent, 0, len(entries))
	for _, entry := range entries {
		var direntType fuse.DirentType
		
		// Try to determine the entry type
		// In NFS, entry itself doesn't contain type,
		// but we could do a Lookup to get attributes
		// For simplicity, we'll just set it to DT_Unknown
		direntType = fuse.DT_Unknown
		
		// Add the entry
		result = append(result, fuse.Dirent{
			Name:  entry.Name,
			Type:  direntType,
			Inode: entry.FileId,
		})
	}
	
	log.Printf("Returning %d directory entries", len(result))
	return result, nil
}