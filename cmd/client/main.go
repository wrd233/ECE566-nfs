package main

import (
	"context"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"time"
	"os"

	"github.com/example/nfsserver/pkg/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	// Parse command line flags
	serverAddr := flag.String("server", "localhost:2049", "NFS server address")
	handleHex := flag.String("handle", "", "File handle in hex format")
	operation := flag.String("op", "getattr", "Operation to perform (getattr, lookup, read, write)")
	uid := flag.Uint("uid", 1000, "User ID")
	gid := flag.Uint("gid", 1000, "Group ID")
	name := flag.String("name", "", "Name to look up (for lookup operation)")
	offset := flag.Uint64("offset", 0, "Offset in file (for read/write operation)")
	count := flag.Uint("count", 1024, "Number of bytes to read (for read operation)")
	dataStr := flag.String("data", "", "Data to write (for write operation)")
	dataFile := flag.String("file", "", "File containing data to write (for write operation)")
	stability := flag.Uint("stability", 0, "Stability level: 0=UNSTABLE, 1=DATA_SYNC, 2=FILE_SYNC (for write operation)")
	
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
	case "read":
        resp, err := client.Read(ctx, &api.ReadRequest{
            FileHandle:  fileHandle,
            Credentials: creds,
            Offset:      *offset,
            Count:       uint32(*count),
        })
		
		if err != nil {
			log.Fatalf("Read failed: %v", err)
		}
		
		// Display the result
		fmt.Printf("Status: %s\n", resp.Status)
		if resp.Status == api.Status_OK {
			fmt.Printf("Data length: %d bytes\n", len(resp.Data))
			fmt.Printf("EOF: %v\n", resp.Eof)
			
			// Print first 100 bytes of data (or less if data is smaller)
			dataPreview := resp.Data
			if len(dataPreview) > 100 {
				dataPreview = dataPreview[:100]
			}
			fmt.Printf("Data preview: %s\n", string(dataPreview))
			
			// Print file attributes
			fmt.Printf("File Size: %d bytes\n", resp.Attributes.Size)
			fmt.Printf("Last Modified: %s\n", 
				time.Unix(resp.Attributes.Mtime.Seconds, int64(resp.Attributes.Mtime.Nano)))
		}
	case "write":
		// Prepare data to write
		var data []byte
		if *dataStr != "" {
			data = []byte(*dataStr)
		} else if *dataFile != "" {
			var err error
			data, err = os.ReadFile(*dataFile)
			if err != nil {
				log.Fatalf("Failed to read data file: %v", err)
			}
		} else {
			log.Fatalf("Either -data or -file must be specified for write operation")
		}
		
		// Validate stability value
		if *stability > 2 {
			log.Fatalf("Invalid stability value: %d (must be 0-2)", *stability)
		}
		
		resp, err := client.Write(ctx, &api.WriteRequest{
			FileHandle:  fileHandle,
			Credentials: creds,
			Offset:      *offset,
			Data:        data,
			Stability:   uint32(*stability),
		})
		
		if err != nil {
			log.Fatalf("Write failed: %v", err)
		}
		
		// Display the result
		fmt.Printf("Status: %s\n", resp.Status)
		if resp.Status == api.Status_OK {
			fmt.Printf("Bytes written: %d\n", resp.Count)
			fmt.Printf("Stability level: %d\n", resp.Stability)
			fmt.Printf("Write verifier: %d\n", resp.Verifier)
			
			// Print file attributes
			fmt.Printf("New file size: %d bytes\n", resp.Attributes.Size)
			fmt.Printf("Last Modified: %s\n", 
				time.Unix(resp.Attributes.Mtime.Seconds, int64(resp.Attributes.Mtime.Nano)))
		}
	case "readdir":
		resp, err := client.ReadDir(ctx, &api.ReadDirRequest{
			DirectoryHandle: fileHandle,
			Credentials:     creds,
			Cookie:          0, 
			CookieVerifier:  0,
			Count:           100,
		})
		
		if err != nil {
			log.Fatalf("ReadDir failed: %v", err)
		}
		
		fmt.Printf("Status: %s\n", resp.Status)
		if resp.Status == api.Status_OK {
			fmt.Printf("Directory entries (%d):\n", len(resp.Entries))
			fmt.Printf("%-20s %-20s %-10s\n", "Name", "FileID", "Cookie")
			fmt.Printf("%-20s %-20s %-10s\n", "----", "------", "------")
			for _, entry := range resp.Entries {
				fmt.Printf("%-20s %-20d %-10d\n", entry.Name, entry.FileId, entry.Cookie)
			}
			fmt.Printf("EOF: %v\n", resp.Eof)
			fmt.Printf("CookieVerifier: %d\n", resp.CookieVerifier)
		}
		
	default:
		fmt.Printf("Unsupported operation: %s\n", *operation)
	}
}