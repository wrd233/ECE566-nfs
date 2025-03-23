package fs

import (
    "context"
)

// FileSystem defines the interface that NFS operations will use to interact
// with the underlying storage system. It abstracts away storage implementation
// details to allow different backends to be used with the same NFS protocol layer.
type FileSystem interface {
    // GetAttr retrieves attributes for the file at the specified path.
    // Returns file information including type, size, permissions, timestamps, etc.
    GetAttr(ctx context.Context, path string) (FileInfo, error)
    
    // SetAttr modifies attributes for the file at the specified path.
    // Only attributes that are non-nil in the attr parameter will be modified.
    SetAttr(ctx context.Context, path string, attr FileAttr) (FileInfo, error)
    
    // Lookup finds a file by name within a directory.
    // Returns the full path to the file and its attributes.
    Lookup(ctx context.Context, dir string, name string) (string, FileInfo, error)
    
    // Access checks if the given credentials can access the file with the requested permission.
    // Returns nil if access is allowed, otherwise an error.
    Access(ctx context.Context, path string, mode FileMode, creds Credentials) error
    
    // Read reads data from a file at the specified offset.
    // Returns the data read, whether the end of file was reached, and any error.
    Read(ctx context.Context, path string, offset int64, length int) ([]byte, bool, error)
    
    // Write writes data to a file at the specified offset.
    // If sync is true, the data should be committed to stable storage before returning.
    // Returns the number of bytes written and any error.
    Write(ctx context.Context, path string, offset int64, data []byte, sync bool) (int, error)
    
    // Create creates a new file in the specified directory.
    // Returns the path to the new file and its attributes.
    // If excl is true, the operation will fail if the file already exists.
    Create(ctx context.Context, dir string, name string, attr FileAttr, excl bool) (string, FileInfo, error)
    
    // Remove removes the specified file.
    // Returns an error if the file doesn't exist or cannot be removed.
    Remove(ctx context.Context, path string) error
    
    // Mkdir creates a new directory.
    // Returns the path to the new directory and its attributes.
    Mkdir(ctx context.Context, dir string, name string, attr FileAttr) (string, FileInfo, error)
    
    // Rmdir removes the specified directory.
    // Returns an error if the directory doesn't exist, is not empty, or cannot be removed.
    Rmdir(ctx context.Context, path string) error
    
    // ReadDir reads the contents of a directory.
    // cookie can be used for pagination, and should be the cookie of the last entry
    // from a previous call.
    // count specifies the maximum number of entries to return.
    // Returns directory entries, the next cookie to use, and any error.
    ReadDir(ctx context.Context, dir string, cookie int64, count int) ([]DirEntry, int64, error)
    
    // ReadDirPlus is like ReadDir, but also returns file attributes for each entry.
    // This can be significantly more efficient than separate Lookup calls for each entry.
    ReadDirPlus(ctx context.Context, dir string, cookie int64, count int) ([]DirEntry, int64, error)
    
    // Rename renames a file or directory.
    // Returns an error if the source doesn't exist or the operation fails.
    Rename(ctx context.Context, oldPath string, newPath string) error
    
    // Symlink creates a symbolic link.
    // Returns the path to the new symlink and its attributes.
    Symlink(ctx context.Context, dir string, name string, target string, attr FileAttr) (string, FileInfo, error)
    
    // Readlink reads the target of a symbolic link.
    // Returns the target path and any error.
    Readlink(ctx context.Context, path string) (string, error)
    
    // StatFS retrieves file system statistics.
    // Returns information about total space, free space, etc.
    StatFS(ctx context.Context) (FSStat, error)
    
    // FileHandleToPath converts a file handle to a file system path.
    // Returns the path and any error.
    FileHandleToPath(fh []byte) (string, error)
    
    // PathToFileHandle converts a file system path to a file handle.
    // Returns the file handle and any error.
    PathToFileHandle(path string) ([]byte, error)
}

// Credentials represents the authentication information for a user.
type Credentials struct {
    // UID is the user ID
    UID uint32
    
    // GID is the primary group ID
    GID uint32
    
    // Groups is the list of supplementary group IDs
    Groups []uint32
}