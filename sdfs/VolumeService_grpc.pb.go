// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v3.12.4
// source: VolumeService.proto

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

// VolumeServiceClient is the client API for VolumeService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type VolumeServiceClient interface {
	AuthenticateUser(ctx context.Context, in *AuthenticationRequest, opts ...grpc.CallOption) (*AuthenticationResponse, error)
	GetVolumeInfo(ctx context.Context, in *VolumeInfoRequest, opts ...grpc.CallOption) (*VolumeInfoResponse, error)
	ShutdownVolume(ctx context.Context, in *ShutdownRequest, opts ...grpc.CallOption) (*ShutdownResponse, error)
	CleanStore(ctx context.Context, in *CleanStoreRequest, opts ...grpc.CallOption) (*CleanStoreResponse, error)
	DeleteCloudVolume(ctx context.Context, in *DeleteCloudVolumeRequest, opts ...grpc.CallOption) (*DeleteCloudVolumeResponse, error)
	DSEInfo(ctx context.Context, in *DSERequest, opts ...grpc.CallOption) (*DSEResponse, error)
	SystemInfo(ctx context.Context, in *SystemInfoRequest, opts ...grpc.CallOption) (*SystemInfoResponse, error)
	SetVolumeCapacity(ctx context.Context, in *SetVolumeCapacityRequest, opts ...grpc.CallOption) (*SetVolumeCapacityResponse, error)
	GetConnectedVolumes(ctx context.Context, in *CloudVolumesRequest, opts ...grpc.CallOption) (*CloudVolumesResponse, error)
	GetGCSchedule(ctx context.Context, in *GCScheduleRequest, opts ...grpc.CallOption) (*GCScheduleResponse, error)
	SetCacheSize(ctx context.Context, in *SetCacheSizeRequest, opts ...grpc.CallOption) (*SetCacheSizeResponse, error)
	SetPassword(ctx context.Context, in *SetPasswordRequest, opts ...grpc.CallOption) (*SetPasswordResponse, error)
	SetReadSpeed(ctx context.Context, in *SpeedRequest, opts ...grpc.CallOption) (*SpeedResponse, error)
	SetWriteSpeed(ctx context.Context, in *SpeedRequest, opts ...grpc.CallOption) (*SpeedResponse, error)
	SyncFromCloudVolume(ctx context.Context, in *SyncFromVolRequest, opts ...grpc.CallOption) (*SyncFromVolResponse, error)
	SyncCloudVolume(ctx context.Context, in *SyncVolRequest, opts ...grpc.CallOption) (*SyncVolResponse, error)
	SetMaxAge(ctx context.Context, in *SetMaxAgeRequest, opts ...grpc.CallOption) (*SetMaxAgeResponse, error)
	ReconcileCloudMetadata(ctx context.Context, in *ReconcileCloudMetadataRequest, opts ...grpc.CallOption) (*ReconcileCloudMetadataResponse, error)
}

type volumeServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewVolumeServiceClient(cc grpc.ClientConnInterface) VolumeServiceClient {
	return &volumeServiceClient{cc}
}

func (c *volumeServiceClient) AuthenticateUser(ctx context.Context, in *AuthenticationRequest, opts ...grpc.CallOption) (*AuthenticationResponse, error) {
	out := new(AuthenticationResponse)
	err := c.cc.Invoke(ctx, "/org.opendedup.grpc.VolumeService/AuthenticateUser", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *volumeServiceClient) GetVolumeInfo(ctx context.Context, in *VolumeInfoRequest, opts ...grpc.CallOption) (*VolumeInfoResponse, error) {
	out := new(VolumeInfoResponse)
	err := c.cc.Invoke(ctx, "/org.opendedup.grpc.VolumeService/GetVolumeInfo", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *volumeServiceClient) ShutdownVolume(ctx context.Context, in *ShutdownRequest, opts ...grpc.CallOption) (*ShutdownResponse, error) {
	out := new(ShutdownResponse)
	err := c.cc.Invoke(ctx, "/org.opendedup.grpc.VolumeService/ShutdownVolume", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *volumeServiceClient) CleanStore(ctx context.Context, in *CleanStoreRequest, opts ...grpc.CallOption) (*CleanStoreResponse, error) {
	out := new(CleanStoreResponse)
	err := c.cc.Invoke(ctx, "/org.opendedup.grpc.VolumeService/CleanStore", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *volumeServiceClient) DeleteCloudVolume(ctx context.Context, in *DeleteCloudVolumeRequest, opts ...grpc.CallOption) (*DeleteCloudVolumeResponse, error) {
	out := new(DeleteCloudVolumeResponse)
	err := c.cc.Invoke(ctx, "/org.opendedup.grpc.VolumeService/DeleteCloudVolume", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *volumeServiceClient) DSEInfo(ctx context.Context, in *DSERequest, opts ...grpc.CallOption) (*DSEResponse, error) {
	out := new(DSEResponse)
	err := c.cc.Invoke(ctx, "/org.opendedup.grpc.VolumeService/DSEInfo", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *volumeServiceClient) SystemInfo(ctx context.Context, in *SystemInfoRequest, opts ...grpc.CallOption) (*SystemInfoResponse, error) {
	out := new(SystemInfoResponse)
	err := c.cc.Invoke(ctx, "/org.opendedup.grpc.VolumeService/SystemInfo", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *volumeServiceClient) SetVolumeCapacity(ctx context.Context, in *SetVolumeCapacityRequest, opts ...grpc.CallOption) (*SetVolumeCapacityResponse, error) {
	out := new(SetVolumeCapacityResponse)
	err := c.cc.Invoke(ctx, "/org.opendedup.grpc.VolumeService/SetVolumeCapacity", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *volumeServiceClient) GetConnectedVolumes(ctx context.Context, in *CloudVolumesRequest, opts ...grpc.CallOption) (*CloudVolumesResponse, error) {
	out := new(CloudVolumesResponse)
	err := c.cc.Invoke(ctx, "/org.opendedup.grpc.VolumeService/GetConnectedVolumes", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *volumeServiceClient) GetGCSchedule(ctx context.Context, in *GCScheduleRequest, opts ...grpc.CallOption) (*GCScheduleResponse, error) {
	out := new(GCScheduleResponse)
	err := c.cc.Invoke(ctx, "/org.opendedup.grpc.VolumeService/GetGCSchedule", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *volumeServiceClient) SetCacheSize(ctx context.Context, in *SetCacheSizeRequest, opts ...grpc.CallOption) (*SetCacheSizeResponse, error) {
	out := new(SetCacheSizeResponse)
	err := c.cc.Invoke(ctx, "/org.opendedup.grpc.VolumeService/SetCacheSize", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *volumeServiceClient) SetPassword(ctx context.Context, in *SetPasswordRequest, opts ...grpc.CallOption) (*SetPasswordResponse, error) {
	out := new(SetPasswordResponse)
	err := c.cc.Invoke(ctx, "/org.opendedup.grpc.VolumeService/SetPassword", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *volumeServiceClient) SetReadSpeed(ctx context.Context, in *SpeedRequest, opts ...grpc.CallOption) (*SpeedResponse, error) {
	out := new(SpeedResponse)
	err := c.cc.Invoke(ctx, "/org.opendedup.grpc.VolumeService/SetReadSpeed", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *volumeServiceClient) SetWriteSpeed(ctx context.Context, in *SpeedRequest, opts ...grpc.CallOption) (*SpeedResponse, error) {
	out := new(SpeedResponse)
	err := c.cc.Invoke(ctx, "/org.opendedup.grpc.VolumeService/SetWriteSpeed", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *volumeServiceClient) SyncFromCloudVolume(ctx context.Context, in *SyncFromVolRequest, opts ...grpc.CallOption) (*SyncFromVolResponse, error) {
	out := new(SyncFromVolResponse)
	err := c.cc.Invoke(ctx, "/org.opendedup.grpc.VolumeService/SyncFromCloudVolume", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *volumeServiceClient) SyncCloudVolume(ctx context.Context, in *SyncVolRequest, opts ...grpc.CallOption) (*SyncVolResponse, error) {
	out := new(SyncVolResponse)
	err := c.cc.Invoke(ctx, "/org.opendedup.grpc.VolumeService/SyncCloudVolume", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *volumeServiceClient) SetMaxAge(ctx context.Context, in *SetMaxAgeRequest, opts ...grpc.CallOption) (*SetMaxAgeResponse, error) {
	out := new(SetMaxAgeResponse)
	err := c.cc.Invoke(ctx, "/org.opendedup.grpc.VolumeService/SetMaxAge", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *volumeServiceClient) ReconcileCloudMetadata(ctx context.Context, in *ReconcileCloudMetadataRequest, opts ...grpc.CallOption) (*ReconcileCloudMetadataResponse, error) {
	out := new(ReconcileCloudMetadataResponse)
	err := c.cc.Invoke(ctx, "/org.opendedup.grpc.VolumeService/ReconcileCloudMetadata", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// VolumeServiceServer is the server API for VolumeService service.
// All implementations must embed UnimplementedVolumeServiceServer
// for forward compatibility
type VolumeServiceServer interface {
	AuthenticateUser(context.Context, *AuthenticationRequest) (*AuthenticationResponse, error)
	GetVolumeInfo(context.Context, *VolumeInfoRequest) (*VolumeInfoResponse, error)
	ShutdownVolume(context.Context, *ShutdownRequest) (*ShutdownResponse, error)
	CleanStore(context.Context, *CleanStoreRequest) (*CleanStoreResponse, error)
	DeleteCloudVolume(context.Context, *DeleteCloudVolumeRequest) (*DeleteCloudVolumeResponse, error)
	DSEInfo(context.Context, *DSERequest) (*DSEResponse, error)
	SystemInfo(context.Context, *SystemInfoRequest) (*SystemInfoResponse, error)
	SetVolumeCapacity(context.Context, *SetVolumeCapacityRequest) (*SetVolumeCapacityResponse, error)
	GetConnectedVolumes(context.Context, *CloudVolumesRequest) (*CloudVolumesResponse, error)
	GetGCSchedule(context.Context, *GCScheduleRequest) (*GCScheduleResponse, error)
	SetCacheSize(context.Context, *SetCacheSizeRequest) (*SetCacheSizeResponse, error)
	SetPassword(context.Context, *SetPasswordRequest) (*SetPasswordResponse, error)
	SetReadSpeed(context.Context, *SpeedRequest) (*SpeedResponse, error)
	SetWriteSpeed(context.Context, *SpeedRequest) (*SpeedResponse, error)
	SyncFromCloudVolume(context.Context, *SyncFromVolRequest) (*SyncFromVolResponse, error)
	SyncCloudVolume(context.Context, *SyncVolRequest) (*SyncVolResponse, error)
	SetMaxAge(context.Context, *SetMaxAgeRequest) (*SetMaxAgeResponse, error)
	ReconcileCloudMetadata(context.Context, *ReconcileCloudMetadataRequest) (*ReconcileCloudMetadataResponse, error)
	mustEmbedUnimplementedVolumeServiceServer()
}

// UnimplementedVolumeServiceServer must be embedded to have forward compatible implementations.
type UnimplementedVolumeServiceServer struct {
}

func (UnimplementedVolumeServiceServer) AuthenticateUser(context.Context, *AuthenticationRequest) (*AuthenticationResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AuthenticateUser not implemented")
}
func (UnimplementedVolumeServiceServer) GetVolumeInfo(context.Context, *VolumeInfoRequest) (*VolumeInfoResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetVolumeInfo not implemented")
}
func (UnimplementedVolumeServiceServer) ShutdownVolume(context.Context, *ShutdownRequest) (*ShutdownResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ShutdownVolume not implemented")
}
func (UnimplementedVolumeServiceServer) CleanStore(context.Context, *CleanStoreRequest) (*CleanStoreResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CleanStore not implemented")
}
func (UnimplementedVolumeServiceServer) DeleteCloudVolume(context.Context, *DeleteCloudVolumeRequest) (*DeleteCloudVolumeResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DeleteCloudVolume not implemented")
}
func (UnimplementedVolumeServiceServer) DSEInfo(context.Context, *DSERequest) (*DSEResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DSEInfo not implemented")
}
func (UnimplementedVolumeServiceServer) SystemInfo(context.Context, *SystemInfoRequest) (*SystemInfoResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SystemInfo not implemented")
}
func (UnimplementedVolumeServiceServer) SetVolumeCapacity(context.Context, *SetVolumeCapacityRequest) (*SetVolumeCapacityResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SetVolumeCapacity not implemented")
}
func (UnimplementedVolumeServiceServer) GetConnectedVolumes(context.Context, *CloudVolumesRequest) (*CloudVolumesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetConnectedVolumes not implemented")
}
func (UnimplementedVolumeServiceServer) GetGCSchedule(context.Context, *GCScheduleRequest) (*GCScheduleResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetGCSchedule not implemented")
}
func (UnimplementedVolumeServiceServer) SetCacheSize(context.Context, *SetCacheSizeRequest) (*SetCacheSizeResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SetCacheSize not implemented")
}
func (UnimplementedVolumeServiceServer) SetPassword(context.Context, *SetPasswordRequest) (*SetPasswordResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SetPassword not implemented")
}
func (UnimplementedVolumeServiceServer) SetReadSpeed(context.Context, *SpeedRequest) (*SpeedResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SetReadSpeed not implemented")
}
func (UnimplementedVolumeServiceServer) SetWriteSpeed(context.Context, *SpeedRequest) (*SpeedResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SetWriteSpeed not implemented")
}
func (UnimplementedVolumeServiceServer) SyncFromCloudVolume(context.Context, *SyncFromVolRequest) (*SyncFromVolResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SyncFromCloudVolume not implemented")
}
func (UnimplementedVolumeServiceServer) SyncCloudVolume(context.Context, *SyncVolRequest) (*SyncVolResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SyncCloudVolume not implemented")
}
func (UnimplementedVolumeServiceServer) SetMaxAge(context.Context, *SetMaxAgeRequest) (*SetMaxAgeResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SetMaxAge not implemented")
}
func (UnimplementedVolumeServiceServer) ReconcileCloudMetadata(context.Context, *ReconcileCloudMetadataRequest) (*ReconcileCloudMetadataResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ReconcileCloudMetadata not implemented")
}
func (UnimplementedVolumeServiceServer) mustEmbedUnimplementedVolumeServiceServer() {}

// UnsafeVolumeServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to VolumeServiceServer will
// result in compilation errors.
type UnsafeVolumeServiceServer interface {
	mustEmbedUnimplementedVolumeServiceServer()
}

func RegisterVolumeServiceServer(s grpc.ServiceRegistrar, srv VolumeServiceServer) {
	s.RegisterService(&VolumeService_ServiceDesc, srv)
}

func _VolumeService_AuthenticateUser_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AuthenticationRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(VolumeServiceServer).AuthenticateUser(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/org.opendedup.grpc.VolumeService/AuthenticateUser",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(VolumeServiceServer).AuthenticateUser(ctx, req.(*AuthenticationRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _VolumeService_GetVolumeInfo_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(VolumeInfoRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(VolumeServiceServer).GetVolumeInfo(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/org.opendedup.grpc.VolumeService/GetVolumeInfo",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(VolumeServiceServer).GetVolumeInfo(ctx, req.(*VolumeInfoRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _VolumeService_ShutdownVolume_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ShutdownRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(VolumeServiceServer).ShutdownVolume(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/org.opendedup.grpc.VolumeService/ShutdownVolume",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(VolumeServiceServer).ShutdownVolume(ctx, req.(*ShutdownRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _VolumeService_CleanStore_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CleanStoreRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(VolumeServiceServer).CleanStore(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/org.opendedup.grpc.VolumeService/CleanStore",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(VolumeServiceServer).CleanStore(ctx, req.(*CleanStoreRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _VolumeService_DeleteCloudVolume_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DeleteCloudVolumeRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(VolumeServiceServer).DeleteCloudVolume(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/org.opendedup.grpc.VolumeService/DeleteCloudVolume",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(VolumeServiceServer).DeleteCloudVolume(ctx, req.(*DeleteCloudVolumeRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _VolumeService_DSEInfo_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DSERequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(VolumeServiceServer).DSEInfo(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/org.opendedup.grpc.VolumeService/DSEInfo",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(VolumeServiceServer).DSEInfo(ctx, req.(*DSERequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _VolumeService_SystemInfo_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SystemInfoRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(VolumeServiceServer).SystemInfo(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/org.opendedup.grpc.VolumeService/SystemInfo",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(VolumeServiceServer).SystemInfo(ctx, req.(*SystemInfoRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _VolumeService_SetVolumeCapacity_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SetVolumeCapacityRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(VolumeServiceServer).SetVolumeCapacity(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/org.opendedup.grpc.VolumeService/SetVolumeCapacity",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(VolumeServiceServer).SetVolumeCapacity(ctx, req.(*SetVolumeCapacityRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _VolumeService_GetConnectedVolumes_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CloudVolumesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(VolumeServiceServer).GetConnectedVolumes(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/org.opendedup.grpc.VolumeService/GetConnectedVolumes",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(VolumeServiceServer).GetConnectedVolumes(ctx, req.(*CloudVolumesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _VolumeService_GetGCSchedule_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GCScheduleRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(VolumeServiceServer).GetGCSchedule(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/org.opendedup.grpc.VolumeService/GetGCSchedule",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(VolumeServiceServer).GetGCSchedule(ctx, req.(*GCScheduleRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _VolumeService_SetCacheSize_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SetCacheSizeRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(VolumeServiceServer).SetCacheSize(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/org.opendedup.grpc.VolumeService/SetCacheSize",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(VolumeServiceServer).SetCacheSize(ctx, req.(*SetCacheSizeRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _VolumeService_SetPassword_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SetPasswordRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(VolumeServiceServer).SetPassword(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/org.opendedup.grpc.VolumeService/SetPassword",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(VolumeServiceServer).SetPassword(ctx, req.(*SetPasswordRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _VolumeService_SetReadSpeed_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SpeedRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(VolumeServiceServer).SetReadSpeed(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/org.opendedup.grpc.VolumeService/SetReadSpeed",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(VolumeServiceServer).SetReadSpeed(ctx, req.(*SpeedRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _VolumeService_SetWriteSpeed_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SpeedRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(VolumeServiceServer).SetWriteSpeed(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/org.opendedup.grpc.VolumeService/SetWriteSpeed",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(VolumeServiceServer).SetWriteSpeed(ctx, req.(*SpeedRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _VolumeService_SyncFromCloudVolume_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SyncFromVolRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(VolumeServiceServer).SyncFromCloudVolume(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/org.opendedup.grpc.VolumeService/SyncFromCloudVolume",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(VolumeServiceServer).SyncFromCloudVolume(ctx, req.(*SyncFromVolRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _VolumeService_SyncCloudVolume_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SyncVolRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(VolumeServiceServer).SyncCloudVolume(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/org.opendedup.grpc.VolumeService/SyncCloudVolume",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(VolumeServiceServer).SyncCloudVolume(ctx, req.(*SyncVolRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _VolumeService_SetMaxAge_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SetMaxAgeRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(VolumeServiceServer).SetMaxAge(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/org.opendedup.grpc.VolumeService/SetMaxAge",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(VolumeServiceServer).SetMaxAge(ctx, req.(*SetMaxAgeRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _VolumeService_ReconcileCloudMetadata_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ReconcileCloudMetadataRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(VolumeServiceServer).ReconcileCloudMetadata(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/org.opendedup.grpc.VolumeService/ReconcileCloudMetadata",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(VolumeServiceServer).ReconcileCloudMetadata(ctx, req.(*ReconcileCloudMetadataRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// VolumeService_ServiceDesc is the grpc.ServiceDesc for VolumeService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var VolumeService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "org.opendedup.grpc.VolumeService",
	HandlerType: (*VolumeServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "AuthenticateUser",
			Handler:    _VolumeService_AuthenticateUser_Handler,
		},
		{
			MethodName: "GetVolumeInfo",
			Handler:    _VolumeService_GetVolumeInfo_Handler,
		},
		{
			MethodName: "ShutdownVolume",
			Handler:    _VolumeService_ShutdownVolume_Handler,
		},
		{
			MethodName: "CleanStore",
			Handler:    _VolumeService_CleanStore_Handler,
		},
		{
			MethodName: "DeleteCloudVolume",
			Handler:    _VolumeService_DeleteCloudVolume_Handler,
		},
		{
			MethodName: "DSEInfo",
			Handler:    _VolumeService_DSEInfo_Handler,
		},
		{
			MethodName: "SystemInfo",
			Handler:    _VolumeService_SystemInfo_Handler,
		},
		{
			MethodName: "SetVolumeCapacity",
			Handler:    _VolumeService_SetVolumeCapacity_Handler,
		},
		{
			MethodName: "GetConnectedVolumes",
			Handler:    _VolumeService_GetConnectedVolumes_Handler,
		},
		{
			MethodName: "GetGCSchedule",
			Handler:    _VolumeService_GetGCSchedule_Handler,
		},
		{
			MethodName: "SetCacheSize",
			Handler:    _VolumeService_SetCacheSize_Handler,
		},
		{
			MethodName: "SetPassword",
			Handler:    _VolumeService_SetPassword_Handler,
		},
		{
			MethodName: "SetReadSpeed",
			Handler:    _VolumeService_SetReadSpeed_Handler,
		},
		{
			MethodName: "SetWriteSpeed",
			Handler:    _VolumeService_SetWriteSpeed_Handler,
		},
		{
			MethodName: "SyncFromCloudVolume",
			Handler:    _VolumeService_SyncFromCloudVolume_Handler,
		},
		{
			MethodName: "SyncCloudVolume",
			Handler:    _VolumeService_SyncCloudVolume_Handler,
		},
		{
			MethodName: "SetMaxAge",
			Handler:    _VolumeService_SetMaxAge_Handler,
		},
		{
			MethodName: "ReconcileCloudMetadata",
			Handler:    _VolumeService_ReconcileCloudMetadata_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "VolumeService.proto",
}
