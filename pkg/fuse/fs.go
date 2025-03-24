package fuse

import (
	"bazil.org/fuse/fs"
	"github.com/example/nfsserver/pkg/client"
)

// NFSFS implements the FUSE filesystem interface
type NFSFS struct {
	// NFS client connection
	client client.NFSClient
	
	// Root directory handle
	rootHandle []byte
}

// NewNFSFS creates a new NFS filesystem
func NewNFSFS(nfsClient client.NFSClient, rootHandle []byte) *NFSFS {
	return &NFSFS{
		client:     nfsClient,
		rootHandle: rootHandle,
	}
}

// Root returns the root directory of the filesystem
func (nfs *NFSFS) Root() (fs.Node, error) {
	return &Dir{
		fs:     nfs,
		handle: nfs.rootHandle,
		path:   "/",
	}, nil
}