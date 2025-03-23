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
    "log"

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
    log.Printf("lookupPathByInode: 查找 inode=%d 的路径", inode)
    
    // 输出当前 inodeMap 的内容以便调试
    log.Printf("当前 inodeMap 内容:")
    count := 0
    l.inodeMap.Range(func(key, value interface{}) bool {
        count++
        if count <= 10 { // 限制输出数量，避免日志过大
            log.Printf("  - inode=%d -> path=%s", key, value)
        }
        return true
    })
    log.Printf("inodeMap 共有 %d 条记录", count)
    
    if path, ok := l.inodeMap.Load(inode); ok {
        pathStr := path.(string)
        log.Printf("lookupPathByInode: 找到路径: %s", pathStr)
        return pathStr, true
    }
    
    log.Printf("lookupPathByInode: 找不到 inode=%d 的路径", inode)
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

func (l *LocalFileSystem) FileHandleToPath(fh []byte) (string, error) {
    log.Printf("FileHandleToPath received handle: %x (length: %d)", fh, len(fh))
    
    handle, err := fs.DeserializeFileHandle(fh)
    if err != nil {
        log.Printf("DeserializeFileHandle error: %v", err)
        return "", fs.NewError("FileHandleToPath", "", fs.ErrInvalidHandle)
    }
    
    log.Printf("Deserialized handle: FS=%d, Inode=%d, Gen=%d", 
        handle.FileSystemID, handle.Inode, handle.Generation)
    
    // Verify filesystem ID
    if handle.FileSystemID != l.fsID {
        return "", fs.NewError("FileHandleToPath", "", fs.ErrStale)
    }
    
    // First try to find in the mapping table
    if path, ok := l.lookupPathByInode(handle.Inode); ok {
        return path, nil
    }
    
    // If not in the mapping table, try dynamic lookup
    log.Printf("No record in mapping table, attempting dynamic lookup for inode=%d", handle.Inode)
    path, err := l.findPathByInode(handle.Inode)
    if err != nil {
        log.Printf("Dynamic lookup failed: %v", err)
        return "", fs.NewError("FileHandleToPath", "", fs.ErrStale)
    }
    
    // After finding the path, update the mapping table
    log.Printf("Dynamic lookup successful: inode=%d -> path=%s", handle.Inode, path)
    l.updateInodeMap(path, handle.Inode)
    
    return path, nil
}

// Add dynamic lookup method
func (l *LocalFileSystem) findPathByInode(targetInode uint64) (string, error) {
    var result string
    var found bool
    
    err := filepath.Walk(l.rootPath, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return nil // Continue traversal
        }
        
        stat, ok := info.Sys().(*syscall.Stat_t)
        if !ok {
            return nil
        }
        
        if stat.Ino == targetInode {
            // Found matching inode
            relPath, err := filepath.Rel(l.rootPath, path)
            if err != nil {
                return nil
            }
            
            // Handle root directory
            if relPath == "." {
                result = "/"
            } else {
                result = "/" + relPath
            }
            
            found = true
            return filepath.SkipAll // Stop traversal after finding
        }
        
        return nil
    })
    
    if err != nil {
        return "", err
    }
    
    if !found {
        return "", fmt.Errorf("could not find file with inode=%d", targetInode)
    }
    
    return result, nil
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
    // Resolve and validate path
    fullPath, err := l.resolvePath(path)
    if err != nil {
        return fs.FileInfo{}, fs.NewError("SetAttr", path, err)
    }
    
    // Get current file info
    fileInfo, err := os.Stat(fullPath)
    if err != nil {
        return fs.FileInfo{}, fs.NewError("SetAttr", path, mapOSError(err))
    }
    
    // Apply attribute changes
    
    // Change mode if specified
    if attr.Mode != nil {
        err = os.Chmod(fullPath, os.FileMode(*attr.Mode))
        if err != nil {
            return fs.FileInfo{}, fs.NewError("SetAttr", path, mapOSError(err))
        }
    }
    
    // Change ownership if specified
    if attr.Uid != nil || attr.Gid != nil {
        // Get current ownership if only one is specified
        stat, ok := fileInfo.Sys().(*syscall.Stat_t)
        if !ok {
            return fs.FileInfo{}, fs.NewError("SetAttr", path, fmt.Errorf("unable to get file system info"))
        }
        
        uid := int(stat.Uid)
        gid := int(stat.Gid)
        
        if attr.Uid != nil {
            uid = int(*attr.Uid)
        }
        
        if attr.Gid != nil {
            gid = int(*attr.Gid)
        }
        
        err = os.Chown(fullPath, uid, gid)
        if err != nil {
            return fs.FileInfo{}, fs.NewError("SetAttr", path, mapOSError(err))
        }
    }
    
    // Change size if specified (truncate file)
    if attr.Size != nil {
        err = os.Truncate(fullPath, *attr.Size)
        if err != nil {
            return fs.FileInfo{}, fs.NewError("SetAttr", path, mapOSError(err))
        }
    }
    
    // Change access/modification times if specified
    if attr.AccessTime != nil || attr.ModifyTime != nil {
        atime := fileInfo.ModTime() // Use current by default
        mtime := fileInfo.ModTime()
        
        if attr.AccessTime != nil {
            atime = *attr.AccessTime
        }
        
        if attr.ModifyTime != nil {
            mtime = *attr.ModifyTime
        }
        
        err = os.Chtimes(fullPath, atime, mtime)
        if err != nil {
            return fs.FileInfo{}, fs.NewError("SetAttr", path, mapOSError(err))
        }
    }
    
    // Get updated file info
    newFileInfo, err := os.Stat(fullPath)
    if err != nil {
        return fs.FileInfo{}, fs.NewError("SetAttr", path, mapOSError(err))
    }
    
    // Convert to fs.FileInfo
    fsInfo, err := l.convertFileInfo(path, newFileInfo)
    if err != nil {
        return fs.FileInfo{}, fs.NewError("SetAttr", path, err)
    }
    
    return fsInfo, nil
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
    // Resolve and validate path
    fullPath, err := l.resolvePath(path)
    if err != nil {
        return fs.NewError("Access", path, err)
    }
    
    // Check if path exists
    fileInfo, err := os.Stat(fullPath)
    if err != nil {
        return fs.NewError("Access", path, mapOSError(err))
    }
    
    // Get system-specific information
    stat, ok := fileInfo.Sys().(*syscall.Stat_t)
    if !ok {
        return fs.NewError("Access", path, fmt.Errorf("unable to get system information"))
    }
    
    // Convert file mode to a permission mask
    requiredPerm := mode & 7 // Keep only the rwx bits
    
    // Check if user is owner, in group, or other
    var checkPerm fs.FileMode
    fileMode := fs.FileMode(fileInfo.Mode() & 0777) // Get permission bits
    
    if stat.Uid == creds.UID {
        // User is owner, check owner permission bits
        checkPerm = (fileMode >> 6) & 7
    } else if stat.Gid == creds.GID || containsGroup(creds.Groups, stat.Gid) {
        // User is in group, check group permission bits
        checkPerm = (fileMode >> 3) & 7
    } else {
        // User is other, check other permission bits
        checkPerm = fileMode & 7
    }
    
    // Check if required permissions are granted
    if (requiredPerm & checkPerm) != requiredPerm {
        return fs.NewError("Access", path, fs.ErrPermission)
    }
    
    return nil
}

// Helper function to check if a GID is in a list of groups
func containsGroup(groups []uint32, gid uint32) bool {
    for _, g := range groups {
        if g == gid {
            return true
        }
    }
    return false
}

// Create creates a new file in the specified directory.
func (l *LocalFileSystem) Create(ctx context.Context, dir string, name string, attr fs.FileAttr, excl bool) (string, fs.FileInfo, error) {
    // Resolve parent directory path
    parentPath, err := l.resolvePath(dir)
    if err != nil {
        return "", fs.FileInfo{}, fs.NewError("Create", dir, err)
    }
    
    // Check if parent is a directory
    parentInfo, err := os.Stat(parentPath)
    if err != nil {
        return "", fs.FileInfo{}, fs.NewError("Create", dir, mapOSError(err))
    }
    
    if !parentInfo.IsDir() {
        return "", fs.FileInfo{}, fs.NewError("Create", dir, fs.ErrNotDir)
    }
    
    // Create full path for new file
    newFilePath := filepath.Join(parentPath, name)
    
    // Check for exclusive create
    if excl {
        _, err := os.Stat(newFilePath)
        if err == nil {
            // File already exists
            return "", fs.FileInfo{}, fs.NewError("Create", filepath.Join(dir, name), fs.ErrExist)
        } else if !os.IsNotExist(err) {
            // Some other error occurred
            return "", fs.FileInfo{}, fs.NewError("Create", filepath.Join(dir, name), mapOSError(err))
        }
    }
    
    // Determine permissions (use default if not specified)
    perm := os.FileMode(0644) // Default permission
    if attr.Mode != nil {
        perm = os.FileMode(*attr.Mode)
    }
    
    // Create the file
    file, err := os.OpenFile(newFilePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, perm)
    if err != nil {
        return "", fs.FileInfo{}, fs.NewError("Create", filepath.Join(dir, name), mapOSError(err))
    }
    defer file.Close()
    
    // Apply other attributes if specified
    if attr.Size != nil || attr.Uid != nil || attr.Gid != nil || attr.AccessTime != nil || attr.ModifyTime != nil {
        newPath := filepath.Join(dir, name)
        _, err = l.SetAttr(ctx, newPath, attr)
        if err != nil {
            return "", fs.FileInfo{}, fs.NewError("Create", newPath, err)
        }
    }
    
    // Get information about the new file
    newFileInfo, err := os.Stat(newFilePath)
    if err != nil {
        return "", fs.FileInfo{}, fs.NewError("Create", filepath.Join(dir, name), mapOSError(err))
    }
    
    // Convert to fs.FileInfo
    newFileRelPath := filepath.Join(dir, name)
    fsInfo, err := l.convertFileInfo(newFileRelPath, newFileInfo)
    if err != nil {
        return "", fs.FileInfo{}, fs.NewError("Create", newFileRelPath, err)
    }
    
    return newFileRelPath, fsInfo, nil
}

// Remove removes the specified file.
func (l *LocalFileSystem) Remove(ctx context.Context, path string) error {
    // Resolve and validate path
    fullPath, err := l.resolvePath(path)
    if err != nil {
        return fs.NewError("Remove", path, err)
    }
    
    // Check if path exists
    fileInfo, err := os.Stat(fullPath)
    if err != nil {
        return fs.NewError("Remove", path, mapOSError(err))
    }
    
    // Check if it's a directory (use Rmdir for directories)
    if fileInfo.IsDir() {
        return fs.NewError("Remove", path, fs.ErrIsDir)
    }
    
    // Remove the file
    err = os.Remove(fullPath)
    if err != nil {
        return fs.NewError("Remove", path, mapOSError(err))
    }
    
    return nil
}

// ReadDir reads the contents of a directory.
func (l *LocalFileSystem) ReadDir(ctx context.Context, dir string, cookie int64, count int) ([]fs.DirEntry, int64, error) {
    // Resolve and validate path
    fullPath, err := l.resolvePath(dir)
    if err != nil {
        return nil, 0, fs.NewError("ReadDir", dir, err)
    }
    
    // Check if path is a directory
    fileInfo, err := os.Stat(fullPath)
    if err != nil {
        return nil, 0, fs.NewError("ReadDir", dir, mapOSError(err))
    }
    
    if !fileInfo.IsDir() {
        return nil, 0, fs.NewError("ReadDir", dir, fs.ErrNotDir)
    }
    
    // Read all directory entries
    entries, err := os.ReadDir(fullPath)
    if err != nil {
        return nil, 0, fs.NewError("ReadDir", dir, mapOSError(err))
    }
    
    // Skip entries before cookie
    if cookie > 0 && cookie < int64(len(entries)) {
        entries = entries[cookie:]
    } else if cookie > 0 {
        // If cookie is beyond the end, return empty result
        return []fs.DirEntry{}, int64(len(entries)), nil
    }
    
    // Limit number of entries if count is specified
    if count > 0 && count < len(entries) {
        entries = entries[:count]
    }
    
    // Convert to fs.DirEntry format
    result := make([]fs.DirEntry, len(entries))
    for i, entry := range entries {
        // Generate a unique file ID (using inode number if possible)
        var fileId uint64
        info, err := entry.Info()
        if err == nil {
            if stat, ok := info.Sys().(*syscall.Stat_t); ok {
                fileId = stat.Ino
            }
        }
        
        // If we couldn't get inode, use a simple hash of the name
        if fileId == 0 {
            h := uint64(0)
            for _, c := range entry.Name() {
                h = h*31 + uint64(c)
            }
            fileId = h
        }
        
        // Set the cookie for this entry (simple index-based approach)
        nextCookie := cookie + int64(i) + 1
        
        result[i] = fs.DirEntry{
            Name:       entry.Name(),
            FileId:     fileId,
            Cookie:     nextCookie,
            Attributes: nil, // No attributes in basic ReadDir
        }
    }
    
    // Return the next cookie value
    nextCookie := cookie + int64(len(result))
    
    return result, nextCookie, nil
}

// Mkdir creates a new directory.
func (l *LocalFileSystem) Mkdir(ctx context.Context, dir string, name string, attr fs.FileAttr) (string, fs.FileInfo, error) {
    // Resolve parent directory path
    parentPath, err := l.resolvePath(dir)
    if err != nil {
        return "", fs.FileInfo{}, fs.NewError("Mkdir", dir, err)
    }
    
    // Check if parent is a directory
    parentInfo, err := os.Stat(parentPath)
    if err != nil {
        return "", fs.FileInfo{}, fs.NewError("Mkdir", dir, mapOSError(err))
    }
    
    if !parentInfo.IsDir() {
        return "", fs.FileInfo{}, fs.NewError("Mkdir", dir, fs.ErrNotDir)
    }
    
    // Create full path for new directory
    newDirPath := filepath.Join(parentPath, name)
    
    // Determine permissions (use default if not specified)
    perm := os.FileMode(0755) // Default permission
    if attr.Mode != nil {
        perm = os.FileMode(*attr.Mode)
    }
    
    // Create the directory
    err = os.Mkdir(newDirPath, perm)
    if err != nil {
        return "", fs.FileInfo{}, fs.NewError("Mkdir", filepath.Join(dir, name), mapOSError(err))
    }
    
    // Get information about the new directory
    newDirInfo, err := os.Stat(newDirPath)
    if err != nil {
        return "", fs.FileInfo{}, fs.NewError("Mkdir", filepath.Join(dir, name), mapOSError(err))
    }
    
    // Convert to fs.FileInfo
    newDirRelPath := filepath.Join(dir, name)
    fsInfo, err := l.convertFileInfo(newDirRelPath, newDirInfo)
    if err != nil {
        return "", fs.FileInfo{}, fs.NewError("Mkdir", newDirRelPath, err)
    }
    
    return newDirRelPath, fsInfo, nil
}

// Rmdir removes the specified directory.
func (l *LocalFileSystem) Rmdir(ctx context.Context, path string) error {
    // Resolve and validate path
    fullPath, err := l.resolvePath(path)
    if err != nil {
        return fs.NewError("Rmdir", path, err)
    }
    
    // Check if path exists and is a directory
    fileInfo, err := os.Stat(fullPath)
    if err != nil {
        return fs.NewError("Rmdir", path, mapOSError(err))
    }
    
    if !fileInfo.IsDir() {
        return fs.NewError("Rmdir", path, fs.ErrNotDir)
    }
    
    // Check if directory is empty
    entries, err := os.ReadDir(fullPath)
    if err != nil {
        return fs.NewError("Rmdir", path, mapOSError(err))
    }
    
    if len(entries) > 0 {
        return fs.NewError("Rmdir", path, fs.ErrNotEmpty)
    }
    
    // Remove the directory
    err = os.Remove(fullPath)
    if err != nil {
        return fs.NewError("Rmdir", path, mapOSError(err))
    }
    
    return nil
}

// ReadDirPlus is like ReadDir, but also returns file attributes for each entry.
func (l *LocalFileSystem) ReadDirPlus(ctx context.Context, dir string, cookie int64, count int) ([]fs.DirEntry, int64, error) {
    return nil, 0, fs.NewError("ReadDirPlus", dir, fs.ErrNotSupported)
}

// Rename renames a file or directory.
func (l *LocalFileSystem) Rename(ctx context.Context, oldPath string, newPath string) error {
    // Resolve and validate both paths
    oldFullPath, err := l.resolvePath(oldPath)
    if err != nil {
        return fs.NewError("Rename", oldPath, err)
    }
    
    newFullPath, err := l.resolvePath(newPath)
    if err != nil {
        return fs.NewError("Rename", newPath, err)
    }
    
    // Check if source exists
    _, err = os.Stat(oldFullPath)
    if err != nil {
        return fs.NewError("Rename", oldPath, mapOSError(err))
    }
    
    // Check if destination parent directory exists
    newParent := filepath.Dir(newFullPath)
    parentInfo, err := os.Stat(newParent)
    if err != nil {
        return fs.NewError("Rename", newPath, mapOSError(err))
    }
    
    if !parentInfo.IsDir() {
        return fs.NewError("Rename", newPath, fs.ErrNotDir)
    }
    
    // Perform the rename operation
    err = os.Rename(oldFullPath, newFullPath)
    if err != nil {
        return fs.NewError("Rename", oldPath+" to "+newPath, mapOSError(err))
    }
    
    return nil
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