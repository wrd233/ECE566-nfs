package local

import (
	"context"
	"fmt"
	"github.com/example/nfsserver/pkg/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
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
		BlockSize:  uint32(512),                         // Default block size
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

// Commit ensures that all data for the specified file has been flushed to stable storage
func (l *LocalFileSystem) Commit(ctx context.Context, path string) error {
	// Resolve and validate path
	fullPath, err := l.resolvePath(path)
	if err != nil {
		return fs.NewError("Commit", path, err)
	}

	// Open the file for syncing
	file, err := os.OpenFile(fullPath, os.O_RDWR, 0)
	if err != nil {
		return fs.NewError("Commit", path, mapOSError(err))
	}
	defer file.Close()

	// Sync to disk
	if err := file.Sync(); err != nil {
		return fs.NewError("Commit", path, mapOSError(err))
	}

	return nil
}
