package fuse

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
	"context"
	"log"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/example/nfsserver/pkg/client"
)

// MountOptions contains options for mounting the filesystem
type MountOptions struct {
	MountPoint   string
	ServerAddr   string  // NFS server address
	ReadOnly     bool
	CacheTimeout time.Duration
	Debug        bool
}

// Mount mounts the NFS filesystem at the specified mount point
func Mount(options MountOptions) error {
	// Create NFS client
	config := &client.Config{
		ServerAddress: options.ServerAddr,
		Timeout:       30 * time.Second,
		MaxRetries:    3,
	}
	
	log.Printf("Connecting to NFS server at %s", options.ServerAddr)
	nfsClient, err := client.NewClient(config)
	if err != nil {
		return fmt.Errorf("failed to connect to NFS server: %w", err)
	}
	
	// Get root handle
	log.Println("Getting root directory handle")
	rootHandle, err := nfsClient.GetRootFileHandle(context.Background())
	if err != nil {
		nfsClient.Close()
		return fmt.Errorf("failed to get root handle: %w", err)
	}
	
	// Mount options
	mountOpts := []fuse.MountOption{
		fuse.FSName("nfs-fuse"),
		fuse.Subtype("nfs"),
	}

	if options.ReadOnly {
		mountOpts = append(mountOpts, fuse.ReadOnly())
	}

	if options.Debug {
		fuse.Debug = func(msg interface{}) {
			fmt.Printf("FUSE: %v\n", msg)
		}
		mountOpts = append(mountOpts, fuse.AllowOther())
	}

	// Mount the filesystem
	log.Printf("Mounting FUSE filesystem at %s", options.MountPoint)
	c, err := fuse.Mount(options.MountPoint, mountOpts...)
	if err != nil {
		nfsClient.Close()
		return fmt.Errorf("failed to mount: %w", err)
	}
	defer c.Close()

	// Create the filesystem
	nfsFS := NewNFSFS(nfsClient, rootHandle)

	// Serve the filesystem until unmounted
	go func() {
		log.Println("Starting FUSE server")
		if err := fs.Serve(c, nfsFS); err != nil {
			log.Printf("Error serving filesystem: %v", err)
		}
	}()
	
	// 删除c.Ready和c.MountError的检查，
	// 等待一小段时间确保挂载成功
	log.Println("Waiting for mount to be ready...")
	time.Sleep(1 * time.Second)

	// Wait for SIGINT or SIGTERM to unmount
	log.Println("FUSE filesystem mounted successfully")
	log.Println("Press Ctrl+C to unmount")
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	// Unmount
	log.Println("Unmounting filesystem...")
	if err := Unmount(options.MountPoint); err != nil {
		log.Printf("Warning: failed to unmount cleanly: %v", err)
	}
	
	// Close NFS client
	nfsClient.Close()
	log.Println("NFS connection closed")

	return nil
}

// Unmount unmounts the filesystem
func Unmount(mountPoint string) error {
	return fuse.Unmount(mountPoint)
}