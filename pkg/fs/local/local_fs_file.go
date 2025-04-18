package local

import (
	"context"
	"github.com/example/nfsserver/pkg/fs"
	"io"
	"os"
	"path/filepath"
)

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
	if offset+int64(length) > fileSize {
		bytesToRead = int(fileSize - offset)
	}

	// Create buffer for reading
	buffer := make([]byte, bytesToRead)

	// Read data
	bytesRead, err := io.ReadFull(file, buffer)

	// Adjust buffer to actual bytes read
	buffer = buffer[:bytesRead]

	// Determine EOF: we're at EOF if the current position after reading is at or past the file size
	eof := (offset+int64(bytesRead) >= fileSize)

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
