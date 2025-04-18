package local

import (
	"context"
	"fmt"
	"github.com/example/nfsserver/pkg/fs"
	"os"
	"syscall"
)

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
