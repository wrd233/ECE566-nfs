package fuse

import (
	"time"
	"context"

	"bazil.org/fuse"
)

// File represents a file in the filesystem
type File struct{}

// Attr sets the attributes of the file
func (f *File) Attr(ctx context.Context, attr *fuse.Attr) error {
	content := "Hello, World!\n"
	attr.Mode = 0644
	attr.Size = uint64(len(content))
	attr.Mtime = time.Now()
	return nil
}

// ReadAll reads all content from the file
func (f *File) ReadAll(ctx context.Context) ([]byte, error) {
	return []byte("Hello, World!\n"), nil
}