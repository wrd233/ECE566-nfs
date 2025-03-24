package main

import (
	"context"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/example/nfsserver/pkg/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	// Parse command line flags
	serverAddr := flag.String("server", "localhost:2049", "NFS server address")
	handleHex := flag.String("handle", "", "File handle in hex format")
	operation := flag.String("op", "getattr", "Operation to perform (getattr, lookup)")
	uid := flag.Uint("uid", 1000, "User ID")
	gid := flag.Uint("gid", 1000, "Group ID")
	name := flag.String("name", "", "Name to look up (for lookup operation)")
	
	flag.Parse()
	
	// Connect to the server
	conn, err := grpc.Dial(*serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()
	
	// Create client
	client := api.NewNFSServiceClient(conn)
	
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	// Create credentials
	creds := &api.Credentials{
		Uid:       uint32(*uid),
		Gid:       uint32(*gid),
		Groups:    []uint32{uint32(*gid)},
	}
	
	// Parse or get the file handle
	var fileHandle []byte
	if *handleHex != "" {
		// Parse hex string to bytes
		var err error
		fileHandle, err = hex.DecodeString(*handleHex)
		if err != nil {
			log.Fatalf("Invalid handle format: %v", err)
		}
		log.Printf("Using handle: %x (length: %d bytes)", fileHandle, len(fileHandle))
	} else {
		log.Fatalf("File handle is required")
	}
	
	// Perform the requested operation
	switch *operation {
	case "getattr":
		resp, err := client.GetAttr(ctx, &api.GetAttrRequest{
			FileHandle:  fileHandle,
			Credentials: creds,
		})
		
		if err != nil {
			log.Fatalf("GetAttr failed: %v", err)
		}
		
		// Display the result
		fmt.Printf("Status: %s\n", resp.Status)
		if resp.Status == api.Status_OK {
			fmt.Printf("File Type: %s\n", resp.Attributes.Type)
			fmt.Printf("Mode: %o\n", resp.Attributes.Mode)
			fmt.Printf("Size: %d bytes\n", resp.Attributes.Size)
			fmt.Printf("Owner: %d:%d\n", resp.Attributes.Uid, resp.Attributes.Gid)
			fmt.Printf("Last Modified: %s\n", 
				time.Unix(resp.Attributes.Mtime.Seconds, int64(resp.Attributes.Mtime.Nano)))
		}
		
	case "lookup":
		if *name == "" {
			log.Fatalf("Name is required for lookup operation")
		}
		
		resp, err := client.Lookup(ctx, &api.LookupRequest{
			DirectoryHandle: fileHandle,
			Name:            *name,
			Credentials:     creds,
		})
		
		if err != nil {
			log.Fatalf("Lookup failed: %v", err)
		}
		
		// Display the result
		fmt.Printf("Status: %s\n", resp.Status)
		if resp.Status == api.Status_OK {
			fmt.Printf("File Handle: %x\n", resp.FileHandle)
			fmt.Printf("File Type: %s\n", resp.Attributes.Type)
			fmt.Printf("Mode: %o\n", resp.Attributes.Mode)
			fmt.Printf("Size: %d bytes\n", resp.Attributes.Size)
			fmt.Printf("Owner: %d:%d\n", resp.Attributes.Uid, resp.Attributes.Gid)
			fmt.Printf("Last Modified: %s\n", 
				time.Unix(resp.Attributes.Mtime.Seconds, int64(resp.Attributes.Mtime.Nano)))
		}
		
	default:
		fmt.Printf("Unsupported operation: %s\n", *operation)
	}
}