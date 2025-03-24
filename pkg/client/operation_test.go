// pkg/client/operations_test.go

package client

import (
    "context"
    "net"
    "testing"
    "time"
	"fmt"
	"bytes"

    "github.com/example/nfsserver/pkg/api"
    "google.golang.org/grpc"
    "google.golang.org/grpc/test/bufconn"
)

// 创建一个模拟的gRPC服务器
func setupMockServer(t *testing.T) (*bufconn.Listener, *mockNFSService, *Client) {
    // 创建一个带缓冲的监听器
    bufSize := 1024 * 1024
    listener := bufconn.Listen(bufSize)
    
    // 创建模拟服务
    mockService := &mockNFSService{
        readDirResponses: make(map[string]*api.ReadDirResponse),
        lookupResponses: make(map[string]*api.LookupResponse),
        readResponses: make(map[string]*api.ReadResponse), 
        writeResponses: make(map[string]*api.WriteResponse),
		createResponses: make(map[string]*api.CreateResponse),
    }
    
    // 启动gRPC服务器
    server := grpc.NewServer()
    api.RegisterNFSServiceServer(server, mockService)
    
    go func() {
        if err := server.Serve(listener); err != nil {
            t.Fatalf("Server exited with error: %v", err)
        }
    }()
    
    // 创建拨号器函数
    dialer := func(context.Context, string) (net.Conn, error) {
        return listener.Dial()
    }
    
    // 创建连接
    ctx := context.Background()
    conn, err := grpc.DialContext(ctx, "bufnet", 
        grpc.WithContextDialer(dialer),
        grpc.WithInsecure())
    if err != nil {
        t.Fatalf("Failed to dial bufnet: %v", err)
    }
    
    // 创建客户端
    client := &Client{
        conn: conn,
        nfsClient: api.NewNFSServiceClient(conn),
        config: &Config{
            Timeout: 5 * time.Second,
            MaxRetries: 1,
        },
        handleCache: NewHandleCache(100, 5*time.Minute),
    }
    
    return listener, mockService, client
}

type mockNFSService struct {
    api.UnimplementedNFSServiceServer
    readDirResponses map[string]*api.ReadDirResponse
    lookupResponses map[string]*api.LookupResponse
    rootHandleResponse *api.GetRootHandleResponse
    readResponses map[string]*api.ReadResponse   
    writeResponses map[string]*api.WriteResponse 
	createResponses map[string]*api.CreateResponse
}
// 添加Read实现
func (m *mockNFSService) Read(ctx context.Context, req *api.ReadRequest) (*api.ReadResponse, error) {
    // 生成键 - 使用句柄和偏移量的组合
    key := fmt.Sprintf("%s:%d", string(req.FileHandle), req.Offset)
    
    // 返回预设响应
    if resp, ok := m.readResponses[key]; ok {
        return resp, nil
    }
    
    // 默认响应 - 空数据
    return &api.ReadResponse{
        Status: api.Status_OK,
        Data: []byte{},
        Eof: true,
        Attributes: &api.FileAttributes{
            Type: api.FileType_REGULAR,
            Size: 0,
        },
    }, nil
}

// 添加Write实现
func (m *mockNFSService) Write(ctx context.Context, req *api.WriteRequest) (*api.WriteResponse, error) {
    // 生成键 - 使用句柄和偏移量的组合
    key := fmt.Sprintf("%s:%d", string(req.FileHandle), req.Offset)
    
    // 返回预设响应
    if resp, ok := m.writeResponses[key]; ok {
        return resp, nil
    }
    
    // 默认响应 - 写入所有数据
    return &api.WriteResponse{
        Status: api.Status_OK,
        Count: uint32(len(req.Data)),
        Stability: req.Stability,
        Verifier: 12345,
        Attributes: &api.FileAttributes{
            Type: api.FileType_REGULAR,
            Size: uint64(req.Offset) + uint64(len(req.Data)),
        },
    }, nil
}

// 模拟ReadDir方法
func (m *mockNFSService) ReadDir(ctx context.Context, req *api.ReadDirRequest) (*api.ReadDirResponse, error) {
    // 生成键
    key := string(req.DirectoryHandle)
    
    // 返回预设响应
    if resp, ok := m.readDirResponses[key]; ok {
        return resp, nil
    }
    
    // 默认响应
    return &api.ReadDirResponse{
        Status: api.Status_OK,
        CookieVerifier: 12345,
        Entries: []*api.DirEntry{},
        Eof: true,
    }, nil
}

// 模拟Lookup方法
func (m *mockNFSService) Lookup(ctx context.Context, req *api.LookupRequest) (*api.LookupResponse, error) {
    // 生成键
    key := string(req.DirectoryHandle) + ":" + req.Name
    
    // 返回预设响应
    if resp, ok := m.lookupResponses[key]; ok {
        return resp, nil
    }
    
    // 默认响应（文件不存在）
    return &api.LookupResponse{
        Status: api.Status_ERR_NOENT,
    }, nil
}

// 测试ReadDir方法
func TestReadDir(t *testing.T) {
    // 设置模拟服务器
    _, mockService, client := setupMockServer(t)
    defer client.Close()
    
    // 预设测试数据
    dirHandle := []byte("test-dir-handle")
    mockService.readDirResponses[string(dirHandle)] = &api.ReadDirResponse{
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
    }
    
    // 测试成功情况
    ctx := context.Background()
    entries, err := client.ReadDir(ctx, dirHandle)
    if err != nil {
        t.Fatalf("ReadDir failed: %v", err)
    }
    
    // 验证结果
    if len(entries) != 5 {
        t.Errorf("Wrong number of entries: got %d, want 5", len(entries))
    }
    
    // 验证条目内容
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
    
    // 测试错误情况
    badHandle := []byte("bad-handle")
    mockService.readDirResponses[string(badHandle)] = &api.ReadDirResponse{
        Status: api.Status_ERR_BADHANDLE,
    }
    
    _, err = client.ReadDir(ctx, badHandle)
    if err == nil {
        t.Error("Expected error for bad handle, got nil")
    }
}


// 添加GetRootHandle实现
func (m *mockNFSService) GetRootHandle(ctx context.Context, req *api.GetRootHandleRequest) (*api.GetRootHandleResponse, error) {
    if m.rootHandleResponse != nil {
        return m.rootHandleResponse, nil
    }
    
    // 默认响应
    return &api.GetRootHandleResponse{
        Status: api.Status_OK,
        FileHandle: []byte("root-dir-handle"),
        Attributes: &api.FileAttributes{
            Type: api.FileType_DIRECTORY,
            Mode: 0755,
        },
    }, nil
}

// 添加测试函数
func TestGetRootFileHandle(t *testing.T) {
    // Setup mock server
    _, mockService, client := setupMockServer(t)
    defer client.Close()
    
    // Setup test data
    rootHandle := []byte("root-dir-handle")
    
    // Mock success response
    mockService.rootHandleResponse = &api.GetRootHandleResponse{
        Status: api.Status_OK,
        FileHandle: rootHandle,
        Attributes: &api.FileAttributes{
            Type: api.FileType_DIRECTORY,
            Mode: 0755,
        },
    }
    
    // Test successful case
    ctx := context.Background()
    handle, err := client.GetRootFileHandle(ctx)
    if err != nil {
        t.Fatalf("GetRootFileHandle failed: %v", err)
    }
    
    // Verify returned handle
    if string(handle) != string(rootHandle) {
        t.Errorf("Wrong root handle: got %v, want %v", handle, rootHandle)
    }
    
    // Test error case
    mockService.rootHandleResponse = &api.GetRootHandleResponse{
        Status: api.Status_ERR_SERVERFAULT,
    }
    
    _, err = client.GetRootFileHandle(ctx)
    if err == nil {
        t.Error("Expected error for server fault, got nil")
    }
}

// 测试Lookup方法
func TestLookup(t *testing.T) {
    // 设置模拟服务器
    _, mockService, client := setupMockServer(t)
    defer client.Close()
    
    // 预设测试数据
    dirHandle := []byte("test-dir-handle")
    fileName := "test-file.txt"
    fileHandle := []byte("test-file-handle")
    
    mockService.lookupResponses[string(dirHandle) + ":" + fileName] = &api.LookupResponse{
        Status: api.Status_OK,
        FileHandle: fileHandle,
        Attributes: &api.FileAttributes{
            Type: api.FileType_REGULAR,
            Mode: 0644,
            Size: 1024,
            Uid: 1000,
            Gid: 1000,
        },
    }
    
    // 测试成功情况
    ctx := context.Background()
    handle, attrs, err := client.Lookup(ctx, dirHandle, fileName)
    if err != nil {
        t.Fatalf("Lookup failed: %v", err)
    }
    
    // 验证返回的句柄
    if string(handle) != string(fileHandle) {
        t.Errorf("Wrong file handle: got %v, want %v", handle, fileHandle)
    }
    
    // 验证文件属性
    if attrs.Type != api.FileType_REGULAR {
        t.Errorf("Wrong file type: got %v, want %v", attrs.Type, api.FileType_REGULAR)
    }
    
    // 测试文件不存在情况
    nonExistentFile := "non-existent.txt"
    mockService.lookupResponses[string(dirHandle) + ":" + nonExistentFile] = &api.LookupResponse{
        Status: api.Status_ERR_NOENT,
    }
    
    _, _, err = client.Lookup(ctx, dirHandle, nonExistentFile)
    if err == nil {
        t.Error("Expected error for non-existent file, got nil")
    }
}

func TestLookupPath(t *testing.T) {
    // Setup mock server
    _, mockService, client := setupMockServer(t)
    defer client.Close()
    
    // Setup test data
    rootHandle := []byte("root-dir-handle")
    dir1Handle := []byte("dir1-handle")
    dir2Handle := []byte("dir2-handle")
    fileHandle := []byte("file-handle")
    
    // Mock root handle response
    mockService.rootHandleResponse = &api.GetRootHandleResponse{
        Status: api.Status_OK,
        FileHandle: rootHandle,
        Attributes: &api.FileAttributes{
            Type: api.FileType_DIRECTORY,
            Mode: 0755,
        },
    }
    
    // Mock other lookup responses
    // dir1 lookup
    mockService.lookupResponses[string(rootHandle) + ":dir1" ] = &api.LookupResponse{
        Status: api.Status_OK,
        FileHandle: dir1Handle,
        Attributes: &api.FileAttributes{
            Type: api.FileType_DIRECTORY,
            Mode: 0755,
        },
    }
    
    // dir2 lookup
    mockService.lookupResponses[string(dir1Handle) + ":dir2" ] = &api.LookupResponse{
        Status: api.Status_OK,
        FileHandle: dir2Handle,
        Attributes: &api.FileAttributes{
            Type: api.FileType_DIRECTORY,
            Mode: 0755,
        },
    }
    
    // file.txt lookup
    mockService.lookupResponses[string(dir2Handle) + ":file.txt" ] = &api.LookupResponse{
        Status: api.Status_OK,
        FileHandle: fileHandle,
        Attributes: &api.FileAttributes{
            Type: api.FileType_REGULAR,
            Mode: 0644,
            Size: 1024,
        },
    }
    
    // Test cases
    testCases := []struct {
        name        string
        path        string
        wantHandle  []byte
        wantErr     bool
    }{
        {"Root path", "/", rootHandle, false},
        {"Single level", "/dir1", dir1Handle, false},
        {"Two levels", "/dir1/dir2", dir2Handle, false},
        {"Full path", "/dir1/dir2/file.txt", fileHandle, false},
        {"Non-existent", "/not-exists", nil, true},
    }
    
    ctx := context.Background()
    
    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            // For non-existent path test
            if tc.name == "Non-existent" {
                mockService.lookupResponses[string(rootHandle) + ":not-exists" ] = &api.LookupResponse{
                    Status: api.Status_ERR_NOENT,
                }
            }
            
            handle, err := client.LookupPath(ctx, tc.path)
            
            // Check error expectation
            if (err != nil) != tc.wantErr {
                t.Errorf("LookupPath() error = %v, wantErr %v", err, tc.wantErr)
                return
            }
            
            // Skip further checks if error was expected
            if tc.wantErr {
                return
            }
            
            // Verify handle
            if string(handle) != string(tc.wantHandle) {
                t.Errorf("Wrong handle for path %s: got %v, want %v", 
                    tc.path, handle, tc.wantHandle)
            }
        })
    }

}

// Test Read method
func TestRead(t *testing.T) {
    // Setup mock server
    _, mockService, client := setupMockServer(t)
    defer client.Close()
    
    // Setup test data
    fileHandle := []byte("test-file-handle")
    fileData := []byte("This is test file content for reading operations")
    
    // Mock responses for different read scenarios
    
    // 1. Normal read in middle of file
    mockService.readResponses[fmt.Sprintf("%s:%d", string(fileHandle), 5)] = &api.ReadResponse{
        Status: api.Status_OK,
        Data: fileData[5:15], // "is test fi"
        Eof: false,
        Attributes: &api.FileAttributes{
            Type: api.FileType_REGULAR,
            Size: uint64(len(fileData)),
        },
    }
    
    // 2. Read to end of file
    mockService.readResponses[fmt.Sprintf("%s:%d", string(fileHandle), 30)] = &api.ReadResponse{
        Status: api.Status_OK,
        Data: fileData[30:], // "reading operations"
        Eof: true,
        Attributes: &api.FileAttributes{
            Type: api.FileType_REGULAR,
            Size: uint64(len(fileData)),
        },
    }
    
    // 3. Read past end of file
    mockService.readResponses[fmt.Sprintf("%s:%d", string(fileHandle), 100)] = &api.ReadResponse{
        Status: api.Status_OK,
        Data: []byte{},
        Eof: true,
        Attributes: &api.FileAttributes{
            Type: api.FileType_REGULAR,
            Size: uint64(len(fileData)),
        },
    }
    
    // 4. Error case
    mockService.readResponses[fmt.Sprintf("%s:%d", string(fileHandle), 999)] = &api.ReadResponse{
        Status: api.Status_ERR_IO,
    }
    
    // Test cases
    testCases := []struct {
        name      string
        offset    int64
        count     int
        wantData  []byte
        wantEof   bool
        wantErr   bool
    }{
        {"Normal read", 5, 10, fileData[5:15], false, false},
        {"Read to EOF", 30, 100, fileData[30:], true, false},
        {"Read past EOF", 100, 10, []byte{}, true, false},
        {"Error case", 999, 10, nil, false, true},
    }
    
    ctx := context.Background()
    
    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            data, eof, err := client.Read(ctx, fileHandle, tc.offset, tc.count)
            
            // Check error expectation
            if (err != nil) != tc.wantErr {
                t.Errorf("Read() error = %v, wantErr %v", err, tc.wantErr)
                return
            }
            
            // Skip further checks if error was expected
            if tc.wantErr {
                return
            }
            
            // Check EOF flag
            if eof != tc.wantEof {
                t.Errorf("Read() eof = %v, want %v", eof, tc.wantEof)
            }
            
            // Check returned data
            if !bytes.Equal(data, tc.wantData) {
                t.Errorf("Read() data = %v, want %v", data, tc.wantData)
            }
        })
    }
}

// Test Write method
func TestWrite(t *testing.T) {
    // Setup mock server
    _, mockService, client := setupMockServer(t)
    defer client.Close()
    
    // Setup test data
    fileHandle := []byte("test-file-handle")
    writeData := []byte("This is test data to write")
    
    // Mock responses for different write scenarios
    
    // 1. Normal write with UNSTABLE stability
    mockService.writeResponses[fmt.Sprintf("%s:%d", string(fileHandle), 0)] = &api.WriteResponse{
        Status: api.Status_OK,
        Count: uint32(len(writeData)),
        Stability: 0, // UNSTABLE
        Verifier: 12345,
        Attributes: &api.FileAttributes{
            Type: api.FileType_REGULAR,
            Size: uint64(len(writeData)),
        },
    }
    
    // 2. Write with DATA_SYNC stability
    mockService.writeResponses[fmt.Sprintf("%s:%d", string(fileHandle), 10)] = &api.WriteResponse{
        Status: api.Status_OK,
        Count: uint32(len(writeData)),
        Stability: 1, // DATA_SYNC
        Verifier: 12346,
        Attributes: &api.FileAttributes{
            Type: api.FileType_REGULAR,
            Size: uint64(10 + len(writeData)),
        },
    }
    
    // 3. Partial write
    mockService.writeResponses[fmt.Sprintf("%s:%d", string(fileHandle), 20)] = &api.WriteResponse{
        Status: api.Status_OK,
        Count: 10, // Only write first 10 bytes
        Stability: 2, // FILE_SYNC
        Verifier: 12347,
        Attributes: &api.FileAttributes{
            Type: api.FileType_REGULAR,
            Size: uint64(20 + 10),
        },
    }
    
    // 4. Write error
    mockService.writeResponses[fmt.Sprintf("%s:%d", string(fileHandle), 999)] = &api.WriteResponse{
        Status: api.Status_ERR_NOSPC,
    }
    
    // Test cases
    testCases := []struct {
        name       string
        offset     int64
        data       []byte
        stability  int
        wantCount  int
        wantErr    bool
    }{
        {"UNSTABLE write", 0, writeData, 0, len(writeData), false},
        {"DATA_SYNC write", 10, writeData, 1, len(writeData), false},
        {"FILE_SYNC write", 20, writeData, 2, 10, false}, // Partial write
        {"Error case", 999, writeData, 0, 0, true},
    }
    
    ctx := context.Background()
    
    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            count, err := client.Write(ctx, fileHandle, tc.offset, tc.data, tc.stability)
            
            // Check error expectation
            if (err != nil) != tc.wantErr {
                t.Errorf("Write() error = %v, wantErr %v", err, tc.wantErr)
                return
            }
            
            // Skip further checks if error was expected
            if tc.wantErr {
                return
            }
            
            // Check written bytes count
            if count != tc.wantCount {
                t.Errorf("Write() count = %v, want %v", count, tc.wantCount)
            }
        })
    }
}



// 添加Create实现
func (m *mockNFSService) Create(ctx context.Context, req *api.CreateRequest) (*api.CreateResponse, error) {
    // 生成键 - 使用目录句柄和文件名的组合
    key := fmt.Sprintf("%s:%s", string(req.DirectoryHandle), req.Name)
    
    // 返回预设响应
    if resp, ok := m.createResponses[key]; ok {
        return resp, nil
    }
    
    // 默认响应 - 创建成功
    return &api.CreateResponse{
        Status: api.Status_OK,
        FileHandle: []byte(fmt.Sprintf("new-file-handle-%s", req.Name)),
        Attributes: &api.FileAttributes{
            Type: api.FileType_REGULAR,
            Mode: req.Attributes.Mode,
            Size: 0,
        },
    }, nil
}

// 添加TestCreate函数
func TestCreate(t *testing.T) {
    // Setup mock server
    _, mockService, client := setupMockServer(t)
    defer client.Close()
    
    // Setup test data
    dirHandle := []byte("test-dir-handle")
    fileName := "newfile.txt"
    fileHandle := []byte("new-file-handle")
    
    // Mock success response
    mockService.createResponses[fmt.Sprintf("%s:%s", string(dirHandle), fileName)] = &api.CreateResponse{
        Status: api.Status_OK,
        FileHandle: fileHandle,
        Attributes: &api.FileAttributes{
            Type: api.FileType_REGULAR,
            Mode: 0644,
            Size: 0,
        },
    }
    
    // Mock error response for existing file
    mockService.createResponses[fmt.Sprintf("%s:%s", string(dirHandle), "exists.txt")] = &api.CreateResponse{
        Status: api.Status_ERR_EXIST,
    }
    
    // Test cases
    testCases := []struct {
        name      string
        dirHandle []byte
        fileName  string
        attrs     *api.FileAttributes
        mode      api.CreateMode
        wantErr   bool
    }{
        {"Create new file", dirHandle, fileName, nil, api.CreateMode_GUARDED, false},
        {"Create with attributes", dirHandle, fileName, &api.FileAttributes{Mode: 0755}, api.CreateMode_UNCHECKED, false},
        {"File exists", dirHandle, "exists.txt", nil, api.CreateMode_GUARDED, true},
    }
    
    ctx := context.Background()
    
    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            handle, attrs, err := client.Create(ctx, tc.dirHandle, tc.fileName, tc.attrs, tc.mode)
            
            // Check error expectation
            if (err != nil) != tc.wantErr {
                t.Errorf("Create() error = %v, wantErr %v", err, tc.wantErr)
                return
            }
            
            // Skip further checks if error was expected
            if tc.wantErr {
                return
            }
            
            // Verify handle is not empty
            if len(handle) == 0 {
                t.Errorf("Create() returned empty handle")
            }
            
            // Verify attributes
            if attrs == nil {
                t.Errorf("Create() returned nil attributes")
            } else if attrs.Type != api.FileType_REGULAR {
                t.Errorf("Create() wrong file type: got %v, want REGULAR", attrs.Type)
            }
        })
    }
}