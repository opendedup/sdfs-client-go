// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v3.12.4
// source: PortRedirector.proto

package sdfs

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

// PortRedirectorServiceClient is the client API for PortRedirectorService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type PortRedirectorServiceClient interface {
	GetProxyVolumes(ctx context.Context, in *ProxyVolumeInfoRequest, opts ...grpc.CallOption) (*ProxyVolumeInfoResponse, error)
	ReloadConfig(ctx context.Context, in *ReloadConfigRequest, opts ...grpc.CallOption) (*ReloadConfigResponse, error)
}

type portRedirectorServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewPortRedirectorServiceClient(cc grpc.ClientConnInterface) PortRedirectorServiceClient {
	return &portRedirectorServiceClient{cc}
}

func (c *portRedirectorServiceClient) GetProxyVolumes(ctx context.Context, in *ProxyVolumeInfoRequest, opts ...grpc.CallOption) (*ProxyVolumeInfoResponse, error) {
	out := new(ProxyVolumeInfoResponse)
	err := c.cc.Invoke(ctx, "/org.opendedup.grpc.PortRedirectorService/GetProxyVolumes", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *portRedirectorServiceClient) ReloadConfig(ctx context.Context, in *ReloadConfigRequest, opts ...grpc.CallOption) (*ReloadConfigResponse, error) {
	out := new(ReloadConfigResponse)
	err := c.cc.Invoke(ctx, "/org.opendedup.grpc.PortRedirectorService/ReloadConfig", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// PortRedirectorServiceServer is the server API for PortRedirectorService service.
// All implementations must embed UnimplementedPortRedirectorServiceServer
// for forward compatibility
type PortRedirectorServiceServer interface {
	GetProxyVolumes(context.Context, *ProxyVolumeInfoRequest) (*ProxyVolumeInfoResponse, error)
	ReloadConfig(context.Context, *ReloadConfigRequest) (*ReloadConfigResponse, error)
	mustEmbedUnimplementedPortRedirectorServiceServer()
}

// UnimplementedPortRedirectorServiceServer must be embedded to have forward compatible implementations.
type UnimplementedPortRedirectorServiceServer struct {
}

func (UnimplementedPortRedirectorServiceServer) GetProxyVolumes(context.Context, *ProxyVolumeInfoRequest) (*ProxyVolumeInfoResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetProxyVolumes not implemented")
}
func (UnimplementedPortRedirectorServiceServer) ReloadConfig(context.Context, *ReloadConfigRequest) (*ReloadConfigResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ReloadConfig not implemented")
}
func (UnimplementedPortRedirectorServiceServer) mustEmbedUnimplementedPortRedirectorServiceServer() {}

// UnsafePortRedirectorServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to PortRedirectorServiceServer will
// result in compilation errors.
type UnsafePortRedirectorServiceServer interface {
	mustEmbedUnimplementedPortRedirectorServiceServer()
}

func RegisterPortRedirectorServiceServer(s grpc.ServiceRegistrar, srv PortRedirectorServiceServer) {
	s.RegisterService(&PortRedirectorService_ServiceDesc, srv)
}

func _PortRedirectorService_GetProxyVolumes_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ProxyVolumeInfoRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PortRedirectorServiceServer).GetProxyVolumes(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/org.opendedup.grpc.PortRedirectorService/GetProxyVolumes",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PortRedirectorServiceServer).GetProxyVolumes(ctx, req.(*ProxyVolumeInfoRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _PortRedirectorService_ReloadConfig_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ReloadConfigRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PortRedirectorServiceServer).ReloadConfig(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/org.opendedup.grpc.PortRedirectorService/ReloadConfig",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PortRedirectorServiceServer).ReloadConfig(ctx, req.(*ReloadConfigRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// PortRedirectorService_ServiceDesc is the grpc.ServiceDesc for PortRedirectorService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var PortRedirectorService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "org.opendedup.grpc.PortRedirectorService",
	HandlerType: (*PortRedirectorServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetProxyVolumes",
			Handler:    _PortRedirectorService_GetProxyVolumes_Handler,
		},
		{
			MethodName: "ReloadConfig",
			Handler:    _PortRedirectorService_ReloadConfig_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "PortRedirector.proto",
}
