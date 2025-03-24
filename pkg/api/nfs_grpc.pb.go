// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.5.1
// - protoc             v3.12.4
// source: proto/nfs.proto

package api

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.64.0 or later.
const _ = grpc.SupportPackageIsVersion9

const (
	NFSService_GetAttr_FullMethodName = "/nfs.NFSService/GetAttr"
	NFSService_Lookup_FullMethodName  = "/nfs.NFSService/Lookup"
	NFSService_Read_FullMethodName    = "/nfs.NFSService/Read"
	NFSService_Write_FullMethodName   = "/nfs.NFSService/Write"
	NFSService_ReadDir_FullMethodName = "/nfs.NFSService/ReadDir"
	NFSService_Create_FullMethodName  = "/nfs.NFSService/Create"
	NFSService_Mkdir_FullMethodName   = "/nfs.NFSService/Mkdir"
)

// NFSServiceClient is the client API for NFSService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
//
// NFSService defines the NFS operations supported by the server
type NFSServiceClient interface {
	// Get file attributes
	GetAttr(ctx context.Context, in *GetAttrRequest, opts ...grpc.CallOption) (*GetAttrResponse, error)
	// Look up a file name in a directory
	Lookup(ctx context.Context, in *LookupRequest, opts ...grpc.CallOption) (*LookupResponse, error)
	// Read from a file
	Read(ctx context.Context, in *ReadRequest, opts ...grpc.CallOption) (*ReadResponse, error)
	// Write to a file
	Write(ctx context.Context, in *WriteRequest, opts ...grpc.CallOption) (*WriteResponse, error)
	// Read Directory
	ReadDir(ctx context.Context, in *ReadDirRequest, opts ...grpc.CallOption) (*ReadDirResponse, error)
	// Create a new file
	Create(ctx context.Context, in *CreateRequest, opts ...grpc.CallOption) (*CreateResponse, error)
	// Create a new directory
	Mkdir(ctx context.Context, in *MkdirRequest, opts ...grpc.CallOption) (*MkdirResponse, error)
}

type nFSServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewNFSServiceClient(cc grpc.ClientConnInterface) NFSServiceClient {
	return &nFSServiceClient{cc}
}

func (c *nFSServiceClient) GetAttr(ctx context.Context, in *GetAttrRequest, opts ...grpc.CallOption) (*GetAttrResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(GetAttrResponse)
	err := c.cc.Invoke(ctx, NFSService_GetAttr_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *nFSServiceClient) Lookup(ctx context.Context, in *LookupRequest, opts ...grpc.CallOption) (*LookupResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(LookupResponse)
	err := c.cc.Invoke(ctx, NFSService_Lookup_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *nFSServiceClient) Read(ctx context.Context, in *ReadRequest, opts ...grpc.CallOption) (*ReadResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(ReadResponse)
	err := c.cc.Invoke(ctx, NFSService_Read_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *nFSServiceClient) Write(ctx context.Context, in *WriteRequest, opts ...grpc.CallOption) (*WriteResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(WriteResponse)
	err := c.cc.Invoke(ctx, NFSService_Write_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *nFSServiceClient) ReadDir(ctx context.Context, in *ReadDirRequest, opts ...grpc.CallOption) (*ReadDirResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(ReadDirResponse)
	err := c.cc.Invoke(ctx, NFSService_ReadDir_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *nFSServiceClient) Create(ctx context.Context, in *CreateRequest, opts ...grpc.CallOption) (*CreateResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(CreateResponse)
	err := c.cc.Invoke(ctx, NFSService_Create_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *nFSServiceClient) Mkdir(ctx context.Context, in *MkdirRequest, opts ...grpc.CallOption) (*MkdirResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(MkdirResponse)
	err := c.cc.Invoke(ctx, NFSService_Mkdir_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// NFSServiceServer is the server API for NFSService service.
// All implementations must embed UnimplementedNFSServiceServer
// for forward compatibility.
//
// NFSService defines the NFS operations supported by the server
type NFSServiceServer interface {
	// Get file attributes
	GetAttr(context.Context, *GetAttrRequest) (*GetAttrResponse, error)
	// Look up a file name in a directory
	Lookup(context.Context, *LookupRequest) (*LookupResponse, error)
	// Read from a file
	Read(context.Context, *ReadRequest) (*ReadResponse, error)
	// Write to a file
	Write(context.Context, *WriteRequest) (*WriteResponse, error)
	// Read Directory
	ReadDir(context.Context, *ReadDirRequest) (*ReadDirResponse, error)
	// Create a new file
	Create(context.Context, *CreateRequest) (*CreateResponse, error)
	// Create a new directory
	Mkdir(context.Context, *MkdirRequest) (*MkdirResponse, error)
	mustEmbedUnimplementedNFSServiceServer()
}

// UnimplementedNFSServiceServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedNFSServiceServer struct{}

func (UnimplementedNFSServiceServer) GetAttr(context.Context, *GetAttrRequest) (*GetAttrResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetAttr not implemented")
}
func (UnimplementedNFSServiceServer) Lookup(context.Context, *LookupRequest) (*LookupResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Lookup not implemented")
}
func (UnimplementedNFSServiceServer) Read(context.Context, *ReadRequest) (*ReadResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Read not implemented")
}
func (UnimplementedNFSServiceServer) Write(context.Context, *WriteRequest) (*WriteResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Write not implemented")
}
func (UnimplementedNFSServiceServer) ReadDir(context.Context, *ReadDirRequest) (*ReadDirResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ReadDir not implemented")
}
func (UnimplementedNFSServiceServer) Create(context.Context, *CreateRequest) (*CreateResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Create not implemented")
}
func (UnimplementedNFSServiceServer) Mkdir(context.Context, *MkdirRequest) (*MkdirResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Mkdir not implemented")
}
func (UnimplementedNFSServiceServer) mustEmbedUnimplementedNFSServiceServer() {}
func (UnimplementedNFSServiceServer) testEmbeddedByValue()                    {}

// UnsafeNFSServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to NFSServiceServer will
// result in compilation errors.
type UnsafeNFSServiceServer interface {
	mustEmbedUnimplementedNFSServiceServer()
}

func RegisterNFSServiceServer(s grpc.ServiceRegistrar, srv NFSServiceServer) {
	// If the following call pancis, it indicates UnimplementedNFSServiceServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&NFSService_ServiceDesc, srv)
}

func _NFSService_GetAttr_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetAttrRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(NFSServiceServer).GetAttr(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: NFSService_GetAttr_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(NFSServiceServer).GetAttr(ctx, req.(*GetAttrRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _NFSService_Lookup_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(LookupRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(NFSServiceServer).Lookup(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: NFSService_Lookup_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(NFSServiceServer).Lookup(ctx, req.(*LookupRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _NFSService_Read_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ReadRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(NFSServiceServer).Read(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: NFSService_Read_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(NFSServiceServer).Read(ctx, req.(*ReadRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _NFSService_Write_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(WriteRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(NFSServiceServer).Write(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: NFSService_Write_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(NFSServiceServer).Write(ctx, req.(*WriteRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _NFSService_ReadDir_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ReadDirRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(NFSServiceServer).ReadDir(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: NFSService_ReadDir_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(NFSServiceServer).ReadDir(ctx, req.(*ReadDirRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _NFSService_Create_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CreateRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(NFSServiceServer).Create(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: NFSService_Create_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(NFSServiceServer).Create(ctx, req.(*CreateRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _NFSService_Mkdir_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MkdirRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(NFSServiceServer).Mkdir(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: NFSService_Mkdir_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(NFSServiceServer).Mkdir(ctx, req.(*MkdirRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// NFSService_ServiceDesc is the grpc.ServiceDesc for NFSService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var NFSService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "nfs.NFSService",
	HandlerType: (*NFSServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetAttr",
			Handler:    _NFSService_GetAttr_Handler,
		},
		{
			MethodName: "Lookup",
			Handler:    _NFSService_Lookup_Handler,
		},
		{
			MethodName: "Read",
			Handler:    _NFSService_Read_Handler,
		},
		{
			MethodName: "Write",
			Handler:    _NFSService_Write_Handler,
		},
		{
			MethodName: "ReadDir",
			Handler:    _NFSService_ReadDir_Handler,
		},
		{
			MethodName: "Create",
			Handler:    _NFSService_Create_Handler,
		},
		{
			MethodName: "Mkdir",
			Handler:    _NFSService_Mkdir_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "proto/nfs.proto",
}
