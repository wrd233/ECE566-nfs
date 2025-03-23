// pkg/fs/local/local_fs.go
package local

import (
    "context"
    "fmt"
    "os"
    "path/filepath"
    "strings"
    "sync"
    "syscall"
    "io"

    "github.com/example/nfsserver/pkg/fs"
)

// LocalFileSystem implements fs.FileSystem using the local operating system's
// filesystem.
type LocalFileSystem struct {
    // rootPath is the base directory in the local filesystem
    rootPath string
    
    // fsID is a unique identifier for this filesystem instance
    fsID uint32
    
    // inodeMap maintains a mapping from inode numbers to paths
    inodeMap sync.Map // map[uint64]string
    
    // generationMap tracks the generation number for each inode
    generationMap sync.Map // map[uint64]uint32
}

// NewLocalFileSystem creates a new local filesystem implementation.
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
    
    // Generate a filesystem ID based on the root path
    fsID := generateFsID(absPath)
    
    return &LocalFileSystem{
        rootPath: absPath,
        fsID:     fsID,
    }, nil
}

// generateFsID creates a filesystem ID from a path
func generateFsID(path string) uint32 {
    var h uint32 = 0
    for _, c := range path {
        h = h*31 + uint32(c)
    }
    return h
}

// resolvePath converts a path relative to the filesystem to an absolute OS path
// with security checks to prevent directory traversal
func (l *LocalFileSystem) resolvePath(path string) (string, error) {
    // Remove leading slash if present for consistency
    path = strings.TrimPrefix(path, "/")
    
    // Clean the path to remove any '..' components
    cleanPath := filepath.Clean(path)
    
    // Join with the root path
    fullPath := filepath.Join(l.rootPath, cleanPath)
    
    // Verify the path is still under the root path (prevent directory traversal)
    if !strings.HasPrefix(fullPath, l.rootPath) {
        return "", fs.ErrInvalidName
    }
    
    return fullPath, nil
}

// getInode retrieves the inode number for a file
func (l *LocalFileSystem) getInode(path string) (uint64, error) {
    fullPath, err := l.resolvePath(path)
    if err != nil {
        return 0, err
    }
    
    info, err := os.Stat(fullPath)
    if err != nil {
        return 0, mapOSError(err)
    }
    
    stat, ok := info.Sys().(*syscall.Stat_t)
    if !ok {
        return 0, fmt.Errorf("unable to get system information for file")
    }
    
    return stat.Ino, nil
}

// getGeneration gets or creates a generation number for an inode
func (l *LocalFileSystem) getGeneration(inode uint64) uint32 {
    if gen, ok := l.generationMap.Load(inode); ok {
        return gen.(uint32)
    }
    
    // For simplicity, we start with generation 1
    l.generationMap.Store(inode, uint32(1))
    return 1
}

// updateInodeMap adds or updates the inode to path mapping
func (l *LocalFileSystem) updateInodeMap(path string, inode uint64) {
    l.inodeMap.Store(inode, path)
}

// lookupPathByInode finds a path by inode number
func (l *LocalFileSystem) lookupPathByInode(inode uint64) (string, bool) {
    if path, ok := l.inodeMap.Load(inode); ok {
        return path.(string), true
    }
    return "", false
}

// openFile safely opens a file with proper error mapping
func (l *LocalFileSystem) openFile(path string, flag int, perm os.FileMode) (*os.File, error) {
    fullPath, err := l.resolvePath(path)
    if err != nil {
        return nil, fs.NewError("open", path, err)
    }
    
    file, err := os.OpenFile(fullPath, flag, perm)
    if err != nil {
        return nil, fs.NewError("open", path, mapOSError(err))
    }
    
    return file, nil
}

// getFileInfo gets the os.FileInfo for a path
func (l *LocalFileSystem) getFileInfo(path string) (os.FileInfo, error) {
    fullPath, err := l.resolvePath(path)
    if err != nil {
        return nil, err
    }
    
    info, err := os.Stat(fullPath)
    if err != nil {
        return nil, mapOSError(err)
    }
    
    return info, nil
}

// convertFileInfo converts os.FileInfo to fs.FileInfo
func (l *LocalFileSystem) convertFileInfo(path string, osInfo os.FileInfo) (fs.FileInfo, error) {
    if osInfo == nil {
        return fs.FileInfo{}, fmt.Errorf("nil FileInfo")
    }
    
    // Get system-specific info
    stat, ok := osInfo.Sys().(*syscall.Stat_t)
    if !ok {
        return fs.FileInfo{}, fmt.Errorf("unable to get system information")
    }
    
    // Determine file type
    fileType := fs.FileTypeRegular
    mode := osInfo.Mode()
    
    if mode.IsDir() {
        fileType = fs.FileTypeDirectory
    } else if mode&os.ModeSymlink != 0 {
        fileType = fs.FileTypeSymlink
    } else if mode&os.ModeDevice != 0 {
        if mode&os.ModeCharDevice != 0 {
            fileType = fs.FileTypeChar
        } else {
            fileType = fs.FileTypeBlock
        }
    } else if mode&os.ModeNamedPipe != 0 {
        fileType = fs.FileTypeFIFO
    } else if mode&os.ModeSocket != 0 {
        fileType = fs.FileTypeSocket
    }
    
    // Convert permission bits
    fsMode := fs.FileMode(mode.Perm())
    
    // Handle special bits (simplified)
    if mode&os.ModeSetuid != 0 {
        fsMode |= fs.ModeSetUID
    }
    if mode&os.ModeSetgid != 0 {
        fsMode |= fs.ModeSetGID
    }
    if mode&os.ModeSticky != 0 {
        fsMode |= fs.ModeSticky
    }
    
    // Use ModTime for all time fields for simplicity and cross-platform compatibility
    modTime := osInfo.ModTime()
    
    // Create FileInfo
    fsInfo := fs.FileInfo{
        Type:       fileType,
        Mode:       fsMode,
        Size:       osInfo.Size(),
        Uid:        stat.Uid,
        Gid:        stat.Gid,
        Nlink:      uint32(stat.Nlink),
        Rdev:       uint64(stat.Rdev),
        BlockSize:  uint32(512), // Default block size
        Blocks:     uint64((osInfo.Size() + 511) / 512), // Approximate blocks from size
        ModifyTime: modTime,
        AccessTime: modTime, // Use ModTime as a fallback
        ChangeTime: modTime, // Use ModTime as a fallback
    }
    
    // Update the inode map
    l.updateInodeMap(path, stat.Ino)
    
    return fsInfo, nil
}

// mapOSError maps os errors to fs errors
func mapOSError(err error) error {
    if os.IsNotExist(err) {
        return fs.ErrNotExist
    } else if os.IsPermission(err) {
        return fs.ErrPermission
    } else if os.IsExist(err) {
        return fs.ErrExist
    }
    
    // Handle more specific errors
    if pathErr, ok := err.(*os.PathError); ok {
        switch pathErr.Err {
        case syscall.ENOTEMPTY:
            return fs.ErrNotEmpty
        case syscall.EINVAL:
            return fs.ErrInvalidName
        case syscall.ENOSPC:
            return fs.ErrNoSpace
        }
    }
    
    // Default to IO error
    return fs.ErrIO
}

// GetAttr retrieves attributes for the file at the specified path.
func (l *LocalFileSystem) GetAttr(ctx context.Context, path string) (fs.FileInfo, error) {
    // Get file info
    osInfo, err := l.getFileInfo(path)
    if err != nil {
        return fs.FileInfo{}, fs.NewError("GetAttr", path, err)
    }
    
    // Convert to fs.FileInfo
    info, err := l.convertFileInfo(path, osInfo)
    if err != nil {
        return fs.FileInfo{}, fs.NewError("GetAttr", path, err)
    }
    
    return info, nil
}

// FileHandleToPath converts a file handle to a file system path.
func (l *LocalFileSystem) FileHandleToPath(fh []byte) (string, error) {
    handle, err := fs.DeserializeFileHandle(fh)
    if err != nil {
        return "", fs.NewError("FileHandleToPath", "", fs.ErrInvalidHandle)
    }
    
    // Verify it's for our filesystem
    if handle.FileSystemID != l.fsID {
        return "", fs.NewError("FileHandleToPath", "", fs.ErrInvalidHandle)
    }
    
    // Try to find path in inode map
    path, ok := l.lookupPathByInode(handle.Inode)
    if !ok {
        return "", fs.NewError("FileHandleToPath", "", fs.ErrStale)
    }
    
    // Verify the generation number
    if gen, ok := l.generationMap.Load(handle.Inode); ok {
        if gen.(uint32) != handle.Generation {
            return "", fs.NewError("FileHandleToPath", "", fs.ErrStale)
        }
    }
    
    return path, nil
}

// PathToFileHandle converts a file system path to a file handle.
func (l *LocalFileSystem) PathToFileHandle(path string) ([]byte, error) {
    // Get inode for the path
    inode, err := l.getInode(path)
    if err != nil {
        return nil, fs.NewError("PathToFileHandle", path, err)
    }
    
    // Get or create generation number
    generation := l.getGeneration(inode)
    
    // Update inode map
    l.updateInodeMap(path, inode)
    
    // Create and serialize the file handle
    handle := &fs.FileHandle{
        FileSystemID: l.fsID,
        Inode:        inode,
        Generation:   generation,
    }
    
    return handle.Serialize(), nil
}

// SetAttr modifies attributes for the file at the specified path.
func (l *LocalFileSystem) SetAttr(ctx context.Context, path string, attr fs.FileAttr) (fs.FileInfo, error) {
    return fs.FileInfo{}, fs.NewError("SetAttr", path, fs.ErrNotSupported)
}

// Lookup finds a file by name within a directory.
func (l *LocalFileSystem) Lookup(ctx context.Context, dir string, name string) (string, fs.FileInfo, error) {
    // Ensure dir is actually a directory
    dirPath, err := l.resolvePath(dir)
    if err != nil {
        return "", fs.FileInfo{}, fs.NewError("Lookup", dir, err)
    }
    
    dirInfo, err := os.Stat(dirPath)
    if err != nil {
        return "", fs.FileInfo{}, fs.NewError("Lookup", dir, mapOSError(err))
    }
    
    if !dirInfo.IsDir() {
        return "", fs.FileInfo{}, fs.NewError("Lookup", dir, fs.ErrNotDir)
    }
    
    // Create the full path for the target file/directory
    targetName := filepath.Join(dir, name)
    
    // Get file info for the target
    fileInfo, err := l.getFileInfo(targetName)
    if err != nil {
        return "", fs.FileInfo{}, fs.NewError("Lookup", targetName, err)
    }
    
    // Convert to fs.FileInfo
    fsInfo, err := l.convertFileInfo(targetName, fileInfo)
    if err != nil {
        return "", fs.FileInfo{}, fs.NewError("Lookup", targetName, err)
    }
    
    return targetName, fsInfo, nil
}

// Read reads data from a file at the specified offset.
func (l *LocalFileSystem) Read(ctx context.Context, path string, offset int64, length int) ([]byte, bool, error) {
    // Resolve and validate path
    fullPath, err := l.resolvePath(path)
    if err != nil {
        return nil, false, fs.NewError("Read", path, err)
    }
    
    // Get file info to check if it's a regular file and get size
    fileInfo, err := os.Stat(fullPath)
    if err != nil {
        return nil, false, fs.NewError("Read", path, mapOSError(err))
    }
    
    if fileInfo.IsDir() {
        return nil, false, fs.NewError("Read", path, fs.ErrIsDir)
    }
    
    fileSize := fileInfo.Size()
    
    // Check if offset is beyond or at file size
    if offset >= fileSize {
        return []byte{}, true, nil // Empty data with EOF flag
    }
    
    // Open the file for reading
    file, err := os.Open(fullPath)
    if err != nil {
        return nil, false, fs.NewError("Read", path, mapOSError(err))
    }
    defer file.Close()
    
    // Seek to the specified offset
    _, err = file.Seek(offset, io.SeekStart)
    if err != nil {
        return nil, false, fs.NewError("Read", path, mapOSError(err))
    }
    
    // Determine how many bytes we can actually read
    // If reading would go beyond EOF, limit to file size
    bytesToRead := length
    if offset + int64(length) > fileSize {
        bytesToRead = int(fileSize - offset)
    }
    
    // Create buffer for reading
    buffer := make([]byte, bytesToRead)
    
    // Read data
    bytesRead, err := io.ReadFull(file, buffer)
    
    // Adjust buffer to actual bytes read
    buffer = buffer[:bytesRead]
    
    // Determine EOF: we're at EOF if the current position after reading is at or past the file size
    eof := (offset + int64(bytesRead) >= fileSize)
    
    // If we got an error other than EOF, return it
    if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
        return buffer, eof, fs.NewError("Read", path, mapOSError(err))
    }
    
    return buffer, eof, nil
}

// Write writes data to a file at the specified offset.
func (l *LocalFileSystem) Write(ctx context.Context, path string, offset int64, data []byte, sync bool) (int, error) {
    // Resolve and validate path
    fullPath, err := l.resolvePath(path)
    if err != nil {
        return 0, fs.NewError("Write", path, err)
    }
    
    // Get file info to check if it's a regular file
    fileInfo, err := os.Stat(fullPath)
    if err != nil {
        return 0, fs.NewError("Write", path, mapOSError(err))
    }
    
    if fileInfo.IsDir() {
        return 0, fs.NewError("Write", path, fs.ErrIsDir)
    }
    
    // Open file for writing
    file, err := os.OpenFile(fullPath, os.O_RDWR, 0)
    if err != nil {
        return 0, fs.NewError("Write", path, mapOSError(err))
    }
    defer file.Close()
    
    // Seek to the specified offset
    _, err = file.Seek(offset, io.SeekStart)
    if err != nil {
        return 0, fs.NewError("Write", path, mapOSError(err))
    }
    
    // Write data
    bytesWritten, err := file.Write(data)
    if err != nil {
        return 0, fs.NewError("Write", path, mapOSError(err))
    }
    
    // Sync to disk if requested
    if sync && bytesWritten > 0 {
        err = file.Sync()
        if err != nil {
            return bytesWritten, fs.NewError("Write", path, mapOSError(err))
        }
    }
    
    return bytesWritten, nil
}

// Access checks if the given credentials can access the file with the requested permission.
func (l *LocalFileSystem) Access(ctx context.Context, path string, mode fs.FileMode, creds fs.Credentials) error {
    return fs.NewError("Access", path, fs.ErrNotSupported)
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