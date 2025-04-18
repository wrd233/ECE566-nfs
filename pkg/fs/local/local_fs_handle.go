package local

import (
	"fmt"
	"github.com/example/nfsserver/pkg/fs"
	"log"
	"os"
	"path/filepath"
	"syscall"
)

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
