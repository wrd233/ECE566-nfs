package fuse

import (
	"os"
	"time"
	"context"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
)

// Dir represents a directory in the filesystem
type Dir struct{}

// Attr sets the attributes of the directory
func (d *Dir) Attr(ctx context.Context, attr *fuse.Attr) error {
	attr.Mode = os.ModeDir | 0755
	attr.Mtime = time.Now()
	return nil
}

// Lookup looks up a specific entry in the directory
func (d *Dir) Lookup(ctx context.Context, name string) (fs.Node, error) {
	if name == "hello.txt" {
		return &File{}, nil
	}
	return nil, fuse.ENOENT
}

// ReadDirAll returns all entries in the directory
func (d *Dir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	return []fuse.Dirent{
		{Name: "hello.txt", Type: fuse.DT_File},
	}, nil
}