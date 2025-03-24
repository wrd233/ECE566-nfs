package client

import (
	"context"
	"testing"
	"time"
	"net"

	"github.com/example/nfsserver/pkg/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

// 创建一个模拟的gRPC服务器
func setupMockServer(t *testing.T) (*bufconn.Listener, *mockNFSService) {
	bufSize := 1024 * 1024
	listener := bufconn.Listen(bufSize)
	
	mockService := &mockNFSService{}
	server := grpc.NewServer()
	api.RegisterNFSServiceServer(server, mockService)
	
	go func() {
		if err := server.Serve(listener); err != nil {
			t.Fatalf("Server exited with error: %v", err)
		}
	}()
	
	return listener, mockService
}

// bufconn拨号器
func bufDialer(listener *bufconn.Listener) func(context.Context, string) (net.Conn, error) {
	return func(ctx context.Context, s string) (net.Conn, error) {
		return listener.Dial()
	}
}

// 模拟的NFS服务
type mockNFSService struct {
	api.UnimplementedNFSServiceServer
	readDirResponses map[string]*api.ReadDirResponse // 存储预设的目录内容响应
}

// 实现ReadDir RPC方法
func (m *mockNFSService) ReadDir(ctx context.Context, req *api.ReadDirRequest) (*api.ReadDirResponse, error) {
	// 生成一个键，基于目录句柄
	key := string(req.DirectoryHandle)
	
	// 如果有预设响应，返回它
	if resp, ok := m.readDirResponses[key]; ok {
		return resp, nil
	}
	
	// 默认响应，空目录
	return &api.ReadDirResponse{
		Status: api.Status_OK,
		CookieVerifier: 12345,
		Entries: []*api.DirEntry{},
		Eof: true,
	}, nil
}

// 测试ReadDir方法基本功能
func TestReadDir(t *testing.T) {
	// 设置模拟服务器
	listener, mockService := setupMockServer(t)
	
	// 预设目录内容
	dirHandle := []byte("test-dir-handle")
	mockService.readDirResponses = map[string]*api.ReadDirResponse{
		string(dirHandle): {
			Status: api.Status_OK,
			CookieVerifier: 12345,
			Entries: []*api.DirEntry{
				{FileId: 1, Name: ".", Cookie: 1},
				{FileId: 2, Name: "..", Cookie: 2},
				{FileId: 3, Name: "file1.txt", Cookie: 3},
				{FileId: 4, Name: "file2.txt", Cookie: 4},
				{FileId: 5, Name: "subdir", Cookie: 5},
			},
			Eof: true,
		},
	}
	
	// 创建带有自定义拨号器的客户端连接
	ctx := context.Background()
	conn, err := grpc.DialContext(ctx, "bufnet", 
		grpc.WithContextDialer(bufDialer(listener)),
		grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()
	
	// 创建客户端并替换gRPC客户端
	client := &Client{
		conn: conn,
		nfsClient: api.NewNFSServiceClient(conn),
		config: &Config{
			Timeout: 5 * time.Second,
		},
		handleCache: NewHandleCache(100, 5*time.Minute),
	}
	
	// 测试ReadDir方法
	entries, err := client.ReadDir(ctx, dirHandle)
	if err != nil {
		t.Fatalf("ReadDir failed: %v", err)
	}
	
	// 验证结果
	if len(entries) != 5 {
		t.Errorf("Wrong number of entries: got %d, want 5", len(entries))
	}
	
	// 验证具体条目
	expectedNames := map[string]bool{
		".": true,
		"..": true,
		"file1.txt": true,
		"file2.txt": true,
		"subdir": true,
	}
	
	for _, entry := range entries {
		if !expectedNames[entry.Name] {
			t.Errorf("Unexpected entry name: %s", entry.Name)
		}
	}
}
