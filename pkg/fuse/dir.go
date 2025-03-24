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


// Create implements the Create method for FUSE directories
func (d *Dir) Create(ctx context.Context, req *fuse.CreateRequest, resp *fuse.CreateResponse) (fs.Node, fs.Handle, error) {
    log.Printf("Creating file %s in directory %s (flags: %v, mode: %v)", 
        req.Name, d.path, req.Flags, req.Mode)
    
    // Convert FUSE mode to NFS attributes
    // TODO:忽略传入的权限，总是使用0666
    attrs := &api.FileAttributes{
        Mode: 0666,
    }
    
    // Use NFS client to create the file
    // Use GUARDED mode to prevent overwrite if exists
    fileHandle, fileAttrs, err := d.fs.client.Create(ctx, d.handle, req.Name, attrs, api.CreateMode_GUARDED)
    if err != nil {
        log.Printf("Create failed: %v", err)
        return nil, nil, fuse.EIO
    }
    
    // Get file size from attributes or default to 0
    var fileSize int64 = 0
    if fileAttrs != nil {
        fileSize = int64(fileAttrs.Size)
    }
    
    // Create file node
    file := &File{
        fs:     d.fs,
        handle: fileHandle,
        path:   d.path + "/" + req.Name,
        size:   fileSize,
    }
    
    // Set appropriate flags in the response
    resp.Flags |= fuse.OpenDirectIO
    
    // Return both node and handle (they're the same in our implementation)
    return file, file, nil
}

// Mkdir implements the Mkdir method for FUSE directories
func (d *Dir) Mkdir(ctx context.Context, req *fuse.MkdirRequest) (fs.Node, error) {
    log.Printf("Creating directory %s in directory %s (mode: %o)", 
        req.Name, d.path, req.Mode)
    
    // Always use full permissions for directories, ignoring potential umask effects
    attrs := &api.FileAttributes{
        Mode: 0777, // rwxrwxrwx
    }
    
    // Use NFS client to create the directory
    dirHandle, _, err := d.fs.client.Mkdir(ctx, d.handle, req.Name, attrs)
    if err != nil {
        log.Printf("Mkdir failed: %v", err)
        return nil, fuse.EIO
    }
    
    // Create directory node
    dir := &Dir{
        fs:     d.fs,
        handle: dirHandle,
        path:   d.path + "/" + req.Name,
    }
    
    return dir, nil
}