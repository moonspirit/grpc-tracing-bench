// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0(customized by red@2023-07-07)
// - protoc             v3.19.4
// source: pkg/proto/echopb/echo.proto

package echopb

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// EchoServiceClient is the client API for EchoService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type EchoServiceClient interface {
	Echo(ctx context.Context, in *Request, opts ...grpc.CallOption) (*Response, error)
	StreamEcho(ctx context.Context, opts ...grpc.CallOption) (EchoService_StreamEchoClient, error)
}

type echoServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewEchoServiceClient(cc grpc.ClientConnInterface) EchoServiceClient {
	return &echoServiceClient{cc}
}

func (c *echoServiceClient) Echo(ctx context.Context, in *Request, opts ...grpc.CallOption) (*Response, error) {
	out := new(Response)
	err := c.cc.Invoke(ctx, "/echopb.EchoService/Echo", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *echoServiceClient) StreamEcho(ctx context.Context, opts ...grpc.CallOption) (EchoService_StreamEchoClient, error) {
	stream, err := c.cc.NewStream(ctx, &EchoService_ServiceDesc.Streams[0], "/echopb.EchoService/StreamEcho", opts...)
	if err != nil {
		return nil, err
	}
	x := &echoServiceStreamEchoClient{stream}
	return x, nil
}

type EchoService_StreamEchoClient interface {
	Send(*Request) error
	Recv() (*Response, error)
	grpc.ClientStream
}

type echoServiceStreamEchoClient struct {
	grpc.ClientStream
}

func (x *echoServiceStreamEchoClient) Send(m *Request) error {
	return x.ClientStream.SendMsg(m)
}

func (x *echoServiceStreamEchoClient) Recv() (*Response, error) {
	m := new(Response)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// EchoServiceServer is the server API for EchoService service.
// All implementations must embed UnimplementedEchoServiceServer
// for forward compatibility
type EchoServiceServer interface {
	Echo(context.Context, *Request) (*Response, error)
	StreamEcho(EchoService_StreamEchoServer) error
	mustEmbedUnimplementedEchoServiceServer()
}

// UnimplementedEchoServiceServer must be embedded to have forward compatible implementations.
type UnimplementedEchoServiceServer struct {
}

func (UnimplementedEchoServiceServer) Echo(context.Context, *Request) (*Response, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Echo not implemented")
}
func (UnimplementedEchoServiceServer) StreamEcho(EchoService_StreamEchoServer) error {
	return status.Errorf(codes.Unimplemented, "method StreamEcho not implemented")
}
func (UnimplementedEchoServiceServer) mustEmbedUnimplementedEchoServiceServer() {}

// UnsafeEchoServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to EchoServiceServer will
// result in compilation errors.
type UnsafeEchoServiceServer interface {
	mustEmbedUnimplementedEchoServiceServer()
}

func RegisterEchoServiceServer(s grpc.ServiceRegistrar, srv EchoServiceServer) {
	s.RegisterService(&EchoService_ServiceDesc, srv)
}

func _EchoService_Echo_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Request)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(EchoServiceServer).Echo(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/echopb.EchoService/Echo",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(EchoServiceServer).Echo(ctx, req.(*Request))
	}
	return interceptor(ctx, in, info, handler)
}

func _EchoService_StreamEcho_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(EchoServiceServer).StreamEcho(&echoServiceStreamEchoServer{stream})
}

type EchoService_StreamEchoServer interface {
	Send(*Response) error
	Recv() (*Request, error)
	grpc.ServerStream
}

type echoServiceStreamEchoServer struct {
	grpc.ServerStream
}

func (x *echoServiceStreamEchoServer) Send(m *Response) error {
	return x.ServerStream.SendMsg(m)
}

func (x *echoServiceStreamEchoServer) Recv() (*Request, error) {
	m := new(Request)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// EchoService_ServiceDesc is the grpc.ServiceDesc for EchoService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var EchoService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "echopb.EchoService",
	HandlerType: (*EchoServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Echo",
			Handler:    _EchoService_Echo_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "StreamEcho",
			Handler:       _EchoService_StreamEcho_Handler,
			ServerStreams: true,
			ClientStreams: true,
		},
	},
	Metadata: "pkg/proto/echopb/echo.proto",
}