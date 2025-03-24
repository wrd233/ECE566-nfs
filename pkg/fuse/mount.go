package fuse

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
)

// MountOptions contains options for mounting the filesystem
type MountOptions struct {
	MountPoint   string
	ReadOnly     bool
	CacheTimeout time.Duration
	Debug        bool
}

// Mount mounts the NFS filesystem at the specified mount point
func Mount(options MountOptions) error {
	// Mount options
	mountOpts := []fuse.MountOption{
		fuse.FSName("nfs-fuse"),
		fuse.Subtype("nfs"),
		// LocalVolume option removed as it doesn't exist
	}

	if options.ReadOnly {
		mountOpts = append(mountOpts, fuse.ReadOnly())
	}

	if options.Debug {
		// Debug option fixed
		mountOpts = append(mountOpts, fuse.AllowOther())
	}

	// Mount the filesystem
	c, err := fuse.Mount(options.MountPoint, mountOpts...)
	if err != nil {
		return fmt.Errorf("failed to mount: %v", err)
	}
	defer c.Close()

	// Create the filesystem
	nfsFS := NewNFSFS()

	// Serve the filesystem until unmounted
	go func() {
		if err := fs.Serve(c, nfsFS); err != nil {
			fmt.Printf("Error serving filesystem: %v\n", err)
		}
	}()

	// Wait for SIGINT or SIGTERM to unmount
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	// Unmount
	if err := Unmount(options.MountPoint); err != nil {
		return fmt.Errorf("failed to unmount: %v", err)
	}

	return nil
}

// Unmount unmounts the filesystem
func Unmount(mountPoint string) error {
	return fuse.Unmount(mountPoint)
}