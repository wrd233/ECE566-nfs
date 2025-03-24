package main

import (
	"context"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/example/nfsserver/pkg/client"
)

func main() {
	// 解析命令行参数
	serverAddr := flag.String("server", "localhost:2049", "NFS server address")
	handleHex := flag.String("handle", "", "Directory handle in hex format")
	flag.Parse()
	
	// 验证参数
	if *handleHex == "" {
		fmt.Println("Error: Directory handle is required")
		fmt.Println("Usage: test-readdir -server <addr> -handle <hex>")
		fmt.Println("\nTo get a directory handle, use the gethandle tool:")
		fmt.Println("  ./bin/gethandle -path /some/dir")
		os.Exit(1)
	}
	
	// 解析目录句柄
	dirHandle, err := hex.DecodeString(*handleHex)
	if err != nil {
		fmt.Printf("Error: Invalid handle format: %v\n", err)
		os.Exit(1)
	}
	
	// 创建客户端
	config := &client.Config{
		ServerAddress: *serverAddr,
		Timeout:       5 * time.Second,
	}
	
	nfsClient, err := client.NewClient(config)
	if err != nil {
		fmt.Printf("Error: Failed to connect to server: %v\n", err)
		os.Exit(1)
	}
	defer nfsClient.Close()
	
	// 执行ReadDir操作
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	fmt.Printf("Reading directory with handle: %x\n", dirHandle)
	entries, err := nfsClient.ReadDir(ctx, dirHandle)
	if err != nil {
		fmt.Printf("Error: ReadDir failed: %v\n", err)
		os.Exit(1)
	}
	
	// 显示结果
	fmt.Printf("Directory contains %d entries:\n", len(entries))
	fmt.Printf("%-20s %-20s %-10s\n", "Name", "FileID", "Cookie")
	fmt.Printf("%-20s %-20s %-10s\n", "----", "------", "------")
	
	for _, entry := range entries {
		fmt.Printf("%-20s %-20d %-10d\n", entry.Name, entry.FileId, entry.Cookie)
	}
}