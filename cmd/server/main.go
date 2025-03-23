package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/example/nfsserver/pkg/fs/local"
	"github.com/example/nfsserver/pkg/server"
)

func main() {
	// Parse command line flags
	listenAddr := flag.String("listen", ":2049", "Network address to listen on")
	rootPath := flag.String("root", "./exports", "Root directory to export")
	maxConcurrent := flag.Int("max-concurrent", 100, "Maximum concurrent requests")
	maxReadSize := flag.Int("max-read", 1024*1024, "Maximum read size in bytes")
	maxWriteSize := flag.Int("max-write", 1024*1024, "Maximum write size in bytes")
	enableRootSquash := flag.Bool("root-squash", true, "Enable root squashing")
	anonUID := flag.Uint("anon-uid", 65534, "Anonymous user ID")
	anonGID := flag.Uint("anon-gid", 65534, "Anonymous group ID")
	requestTimeout := flag.Int("timeout", 30, "Request timeout in seconds")
	
	flag.Parse()
	
	// Create the server configuration
	config := &server.Config{
		ListenAddress:    *listenAddr,
		MaxConcurrent:    *maxConcurrent,
		MaxReadSize:      *maxReadSize,
		MaxWriteSize:     *maxWriteSize,
		EnableRootSquash: *enableRootSquash,
		AnonUID:          uint32(*anonUID),
		AnonGID:          uint32(*anonGID),
		RequestTimeout:   *requestTimeout,
	}
	
	// Ensure export directory exists
	if err := os.MkdirAll(*rootPath, 0755); err != nil {
		log.Fatalf("Failed to create export directory: %v", err)
	}
	
	// Create the filesystem
	fileSystem, err := local.NewLocalFileSystem(*rootPath)
	if err != nil {
		log.Fatalf("Failed to initialize filesystem: %v", err)
	}
	
	// Create and start the NFS server
	nfsServer, err := server.NewNFSServer(config, fileSystem)
	if err != nil {
		log.Fatalf("Failed to create NFS server: %v", err)
	}
	
	// Start the server in a goroutine
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- nfsServer.Start()
	}()
	
	// Wait for signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	// Wait for either the server to error or a signal
	select {
	case err := <-serverErr:
		if err != nil {
			log.Fatalf("Server error: %v", err)
		}
	case sig := <-sigChan:
		log.Printf("Received signal %v, shutting down...", sig)
	}
	
	log.Println("NFS server stopped")
}