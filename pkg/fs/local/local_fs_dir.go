package local

import (
	"context"
	"fmt"
	"github.com/example/nfsserver/pkg/fs"
	"os"
	"path/filepath"
	"syscall"
)

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

	// Create all entries including "." and ".."
	var allEntries []fs.DirEntry

	// Get inode for current directory
	currentDirStat, ok := fileInfo.Sys().(*syscall.Stat_t)
	if !ok {
		return nil, 0, fs.NewError("ReadDir", dir, fmt.Errorf("unable to get system information"))
	}

	// Add "." entry (current directory)
	allEntries = append(allEntries, fs.DirEntry{
		Name:   ".",
		FileId: currentDirStat.Ino,
		Cookie: 1,
	})

	// Add ".." entry (parent directory)
	parentPath := filepath.Dir(fullPath)
	parentInfo, err := os.Stat(parentPath)
	var parentIno uint64
	if err == nil {
		if parentStat, ok := parentInfo.Sys().(*syscall.Stat_t); ok {
			parentIno = parentStat.Ino
		}
	}
	if parentIno == 0 {
		// If we couldn't get parent inode, use a derivative of current inode
		parentIno = currentDirStat.Ino ^ 0x1234
	}

	allEntries = append(allEntries, fs.DirEntry{
		Name:   "..",
		FileId: parentIno,
		Cookie: 2,
	})

	// Read regular directory entries
	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return nil, 0, fs.NewError("ReadDir", dir, mapOSError(err))
	}

	// Add regular entries
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

		// Set the cookie for this entry (i+3 because "." is 1 and ".." is 2)
		nextCookie := int64(i + 3)

		allEntries = append(allEntries, fs.DirEntry{
			Name:       entry.Name(),
			FileId:     fileId,
			Cookie:     nextCookie,
			Attributes: nil, // No attributes in basic ReadDir
		})
	}

	// Handle pagination using cookie
	var result []fs.DirEntry
	if cookie == 0 {
		// First page, include all entries up to count
		result = allEntries
	} else {
		// Find entries after the cookie
		for _, entry := range allEntries {
			if entry.Cookie > cookie {
				result = append(result, entry)
			}
		}
	}

	// Limit number of entries if count is specified
	if count > 0 && count < len(result) {
		result = result[:count]
	}

	// Calculate next cookie value
	var nextCookie int64
	if len(result) > 0 {
		// Next cookie is the cookie of the last entry plus 1
		nextCookie = result[len(result)-1].Cookie + 1
	} else {
		// No entries, use a cookie that indicates end of directory
		nextCookie = int64(len(allEntries) + 1)
	}

	// Update the inode map for "." and ".."
	l.updateInodeMap(dir, currentDirStat.Ino)

	// Parent directory path relative to root
	parentRelPath := filepath.Dir(dir)
	if parentRelPath == "." {
		parentRelPath = "/"
	}
	l.updateInodeMap(parentRelPath, parentIno)

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
