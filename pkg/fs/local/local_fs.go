// pkg/fs/local/local_fs.go
package local

import (
    "context"
    "crypto/sha256"
    "encoding/binary"
    "encoding/hex"
    "fmt"
    "os"
    "path/filepath"
    "sync"
    "time"

    "github.com/example/nfsserver/pkg/fs"
)

// LocalFileSystem implements fs.FileSystem using the local operating system's
// filesystem. This provides direct access to files on the host machine.
type LocalFileSystem struct {
    // rootPath is the base directory in the local filesystem
    rootPath string
    
    // handleCache maps paths to generated handles
    handleCache sync.Map // map[string][]byte
    
    // pathCache maps handle hashes to paths
    pathCache sync.Map // map[string]string
    
    // handleLock protects handle generation
    handleLock sync.Mutex
}

// NewLocalFileSystem creates a new local filesystem implementation.
// rootPath is the base directory that all operations will be relative to.
func NewLocalFileSystem(rootPath string) (*LocalFileSystem, error) {
    // Ensure rootPath exists and is a directory
    fi, err := os.Stat(rootPath)
    if err != nil {
        return nil, fs.NewError("init", rootPath, err)
    }
    
    if !fi.IsDir() {
        return nil, fs.NewError("init", rootPath, fs.ErrNotDir)
    }
    
    // Get absolute path to ensure consistency
    absPath, err := filepath.Abs(rootPath)
    if err != nil {
        return nil, fs.NewError("init", rootPath, err)
    }
    
    return &LocalFileSystem{
        rootPath: absPath,
    }, nil
}

// resolvePath converts a path relative to the filesystem to an absolute OS path
func (l *LocalFileSystem) resolvePath(path string) string {
    // Clean the path to remove any '..' components
    cleanPath := filepath.Clean(path)
    
    // Join with the root path
    return filepath.Join(l.rootPath, cleanPath)
}

// GetAttr retrieves attributes for the file at the specified path.
func (l *LocalFileSystem) GetAttr(ctx context.Context, path string) (fs.FileInfo, error) {
    // This is a stub implementation - in a real implementation, this would:
    // 1. Resolve the path to an absolute path
    // 2. Use os.Stat to get file info
    // 3. Convert os.FileInfo to fs.FileInfo
    // 4. Return the result
    
    return fs.FileInfo{}, fs.NewError("GetAttr", path, fs.ErrNotSupported)
}

// SetAttr modifies attributes for the file at the specified path.
func (l *LocalFileSystem) SetAttr(ctx context.Context, path string, attr fs.FileAttr) (fs.FileInfo, error) {
    return fs.FileInfo{}, fs.NewError("SetAttr", path, fs.ErrNotSupported)
}

// Lookup finds a file by name within a directory.
func (l *LocalFileSystem) Lookup(ctx context.Context, dir string, name string) (string, fs.FileInfo, error) {
    return "", fs.FileInfo{}, fs.NewError("Lookup", filepath.Join(dir, name), fs.ErrNotSupported)
}

// Access checks if the given credentials can access the file with the requested permission.
func (l *LocalFileSystem) Access(ctx context.Context, path string, mode fs.FileMode, creds fs.Credentials) error {
    return fs.NewError("Access", path, fs.ErrNotSupported)
}

// Read reads data from a file at the specified offset.
func (l *LocalFileSystem) Read(ctx context.Context, path string, offset int64, length int) ([]byte, bool, error) {
    return nil, false, fs.NewError("Read", path, fs.ErrNotSupported)
}

// Write writes data to a file at the specified offset.
func (l *LocalFileSystem) Write(ctx context.Context, path string, offset int64, data []byte, sync bool) (int, error) {
    return 0, fs.NewError("Write", path, fs.ErrNotSupported)
}

// Create creates a new file in the specified directory.
func (l *LocalFileSystem) Create(ctx context.Context, dir string, name string, attr fs.FileAttr, excl bool) (string, fs.FileInfo, error) {
    return "", fs.FileInfo{}, fs.NewError("Create", filepath.Join(dir, name), fs.ErrNotSupported)
}

// Remove removes the specified file.
func (l *LocalFileSystem) Remove(ctx context.Context, path string) error {
    return fs.NewError("Remove", path, fs.ErrNotSupported)
}

// Mkdir creates a new directory.
func (l *LocalFileSystem) Mkdir(ctx context.Context, dir string, name string, attr fs.FileAttr) (string, fs.FileInfo, error) {
    return "", fs.FileInfo{}, fs.NewError("Mkdir", filepath.Join(dir, name), fs.ErrNotSupported)
}

// Rmdir removes the specified directory.
func (l *LocalFileSystem) Rmdir(ctx context.Context, path string) error {
    return fs.NewError("Rmdir", path, fs.ErrNotSupported)
}

// ReadDir reads the contents of a directory.
func (l *LocalFileSystem) ReadDir(ctx context.Context, dir string, cookie int64, count int) ([]fs.DirEntry, int64, error) {
    return nil, 0, fs.NewError("ReadDir", dir, fs.ErrNotSupported)
}

// ReadDirPlus is like ReadDir, but also returns file attributes for each entry.
func (l *LocalFileSystem) ReadDirPlus(ctx context.Context, dir string, cookie int64, count int) ([]fs.DirEntry, int64, error) {
    return nil, 0, fs.NewError("ReadDirPlus", dir, fs.ErrNotSupported)
}

// Rename renames a file or directory.
func (l *LocalFileSystem) Rename(ctx context.Context, oldPath string, newPath string) error {
    return fs.NewError("Rename", oldPath, fs.ErrNotSupported)
}

// Symlink creates a symbolic link.
func (l *LocalFileSystem) Symlink(ctx context.Context, dir string, name string, target string, attr fs.FileAttr) (string, fs.FileInfo, error) {
    return "", fs.FileInfo{}, fs.NewError("Symlink", filepath.Join(dir, name), fs.ErrNotSupported)
}

// Readlink reads the target of a symbolic link.
func (l *LocalFileSystem) Readlink(ctx context.Context, path string) (string, error) {
    return "", fs.NewError("Readlink", path, fs.ErrNotSupported)
}

// StatFS retrieves file system statistics.
func (l *LocalFileSystem) StatFS(ctx context.Context) (fs.FSStat, error) {
    return fs.FSStat{}, fs.NewError("StatFS", "", fs.ErrNotSupported)
}

// FileHandleToPath converts a file handle to a file system path.
func (l *LocalFileSystem) FileHandleToPath(fh []byte) (string, error) {
    if len(fh) == 0 {
        return "", fs.ErrInvalidHandle
    }
    
    // Convert handle to string for map lookup
    handleHex := hex.EncodeToString(fh)
    
    // Look up in cache
    if pathObj, ok := l.pathCache.Load(handleHex); ok {
        if path, ok := pathObj.(string); ok {
            return path, nil
        }
    }
    
    return "", fs.NewError("FileHandleToPath", "", fs.ErrInvalidHandle)
}

// PathToFileHandle converts a file system path to a file handle.
func (l *LocalFileSystem) PathToFileHandle(path string) ([]byte, error) {
    // Look up in cache first
    if handleObj, ok := l.handleCache.Load(path); ok {
        if handle, ok := handleObj.([]byte); ok {
            return handle, nil
        }
    }
    
    l.handleLock.Lock()
    defer l.handleLock.Unlock()
    
    // Check again in case it was added while waiting for lock
    if handleObj, ok := l.handleCache.Load(path); ok {
        if handle, ok := handleObj.([]byte); ok {
            return handle, nil
        }
    }
    
    // Generate a new handle
    // Format: SHA-256(rootPath + ":" + path + ":" + timestamp)
    timestamp := time.Now().UnixNano()
    data := fmt.Sprintf("%s:%s:%d", l.rootPath, path, timestamp)
    hash := sha256.Sum256([]byte(data))
    
    // Add a timestamp prefix to ensure uniqueness
    handle := make([]byte, 8+32)
    binary.BigEndian.PutUint64(handle[:8], uint64(timestamp))
    copy(handle[8:], hash[:])
    
    // Store in cache
    handleHex := hex.EncodeToString(handle)
    l.handleCache.Store(path, handle)
    l.pathCache.Store(handleHex, path)
    
    return handle, nil
}