package fuse

import (
	"bazil.org/fuse/fs"
)

// NFSFS implements the FUSE filesystem interface
type NFSFS struct{}

// NewNFSFS creates a new NFS filesystem
func NewNFSFS() *NFSFS {
	return &NFSFS{}
}

// Root returns the root directory of the filesystem
func (nfs *NFSFS) Root() (fs.Node, error) {
	return &Dir{}, nil
}