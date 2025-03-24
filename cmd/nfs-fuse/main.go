package main

import (
	"flag"
	"fmt"
	"os"
	"time"
	"os/signal"
    "syscall"
    "os/exec"

	"github.com/example/nfsserver/pkg/fuse"
)

func main() {
	// Parse command line arguments
	mountPoint := flag.String("mount", "", "Mount point for NFS filesystem")
	readOnly := flag.Bool("readonly", false, "Mount filesystem as read-only")
	debug := flag.Bool("debug", false, "Enable debug logging")
	flag.Parse()

	// Check if mount point is provided
	if *mountPoint == "" {
		fmt.Println("Error: Mount point is required")
		flag.Usage()
		os.Exit(1)
	}

	// Create mount options
	options := fuse.MountOptions{
		MountPoint:   *mountPoint,
		ReadOnly:     *readOnly,
		CacheTimeout: 1 * time.Minute,
		Debug:        *debug,
	}

    c := make(chan os.Signal, 1)
    signal.Notify(c, os.Interrupt, syscall.SIGTERM)
    go func() {
        <-c
        fmt.Println("\nReceived interrupt, exiting...")
        exec.Command("fusermount", "-uz", *mountPoint).Run()
        os.Exit(0)
    }()

	// Mount filesystem
	fmt.Printf("Mounting NFS filesystem at %s\n", *mountPoint)
	if err := fuse.Mount(options); err != nil {
		fmt.Printf("Error mounting filesystem: %v\n", err)
		os.Exit(1)
	}
}
