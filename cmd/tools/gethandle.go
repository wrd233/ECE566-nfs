package main

import (
	"context"
	"encoding/hex"
	"flag"
	"fmt"
	"log"

	"github.com/example/nfsserver/pkg/fs/local"
)

func main() {
	// Parse command line flags
	rootPath := flag.String("root", "./exports", "Root directory to export")
	path := flag.String("path", "/", "Path relative to root")
	
	flag.Parse()
	
	// Create the filesystem
	fileSystem, err := local.NewLocalFileSystem(*rootPath)
	if err != nil {
		log.Fatalf("Failed to initialize filesystem: %v", err)
	}
	
	// Get the file handle
	handle, err := fileSystem.PathToFileHandle(*path)
	if err != nil {
		log.Fatalf("Failed to get file handle: %v", err)
	}
	
	// Display the handle in hex format
	fmt.Printf("File handle for '%s': %s\n", *path, hex.EncodeToString(handle))
	
	// Test validating the handle
	resolvedPath, err := fileSystem.FileHandleToPath(handle)
	if err != nil {
		log.Fatalf("Failed to validate handle: %v", err)
	}
	
	// Get file info
	info, err := fileSystem.GetAttr(context.Background(), resolvedPath)
	if err != nil {
		log.Fatalf("Failed to get file info: %v", err)
	}
	
	// Display file info
	fmt.Printf("File type: %s\n", info.Type)
	fmt.Printf("Size: %d bytes\n", info.Size)
	fmt.Printf("Mode: %o\n", info.Mode)
	fmt.Printf("Owner: %d:%d\n", info.Uid, info.Gid)
	fmt.Printf("Modified: %s\n", info.ModifyTime)

	fmt.Printf("File handle for '%s':\n%s\n", *path, hex.EncodeToString(handle))
	fmt.Printf("Use this exact string with client:\n")
	fmt.Printf("./bin/nfsclient -handle %s -op getattr\n", hex.EncodeToString(handle))
}