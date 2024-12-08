// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.5.1
// - protoc             v5.29.1
// source: client.proto

package proto

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
	SpiderService_GetTask_FullMethodName          = "/spiders.SpiderService/GetTask"
	SpiderService_HandleTask_FullMethodName       = "/spiders.SpiderService/HandleTask"
	SpiderService_ControlSpiders_FullMethodName   = "/spiders.SpiderService/ControlSpiders"
	SpiderService_GetSpidersStatus_FullMethodName = "/spiders.SpiderService/GetSpidersStatus"
)

// SpiderServiceClient is the client API for SpiderService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type SpiderServiceClient interface {
	GetTask(ctx context.Context, in *TaskRequest, opts ...grpc.CallOption) (*CrawlerTask, error)
	HandleTask(ctx context.Context, in *CrawlerResult, opts ...grpc.CallOption) (*HandleResponse, error)
	ControlSpiders(ctx context.Context, in *ControlRequest, opts ...grpc.CallOption) (*ControlResponse, error)
	GetSpidersStatus(ctx context.Context, in *StatusRequest, opts ...grpc.CallOption) (*StatusResponse, error)
}

type spiderServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewSpiderServiceClient(cc grpc.ClientConnInterface) SpiderServiceClient {
	return &spiderServiceClient{cc}
}

func (c *spiderServiceClient) GetTask(ctx context.Context, in *TaskRequest, opts ...grpc.CallOption) (*CrawlerTask, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(CrawlerTask)
	err := c.cc.Invoke(ctx, SpiderService_GetTask_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *spiderServiceClient) HandleTask(ctx context.Context, in *CrawlerResult, opts ...grpc.CallOption) (*HandleResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(HandleResponse)
	err := c.cc.Invoke(ctx, SpiderService_HandleTask_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *spiderServiceClient) ControlSpiders(ctx context.Context, in *ControlRequest, opts ...grpc.CallOption) (*ControlResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(ControlResponse)
	err := c.cc.Invoke(ctx, SpiderService_ControlSpiders_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *spiderServiceClient) GetSpidersStatus(ctx context.Context, in *StatusRequest, opts ...grpc.CallOption) (*StatusResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(StatusResponse)
	err := c.cc.Invoke(ctx, SpiderService_GetSpidersStatus_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// SpiderServiceServer is the server API for SpiderService service.
// All implementations must embed UnimplementedSpiderServiceServer
// for forward compatibility.
type SpiderServiceServer interface {
	GetTask(context.Context, *TaskRequest) (*CrawlerTask, error)
	HandleTask(context.Context, *CrawlerResult) (*HandleResponse, error)
	ControlSpiders(context.Context, *ControlRequest) (*ControlResponse, error)
	GetSpidersStatus(context.Context, *StatusRequest) (*StatusResponse, error)
	mustEmbedUnimplementedSpiderServiceServer()
}

// UnimplementedSpiderServiceServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedSpiderServiceServer struct{}

func (UnimplementedSpiderServiceServer) GetTask(context.Context, *TaskRequest) (*CrawlerTask, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetTask not implemented")
}
func (UnimplementedSpiderServiceServer) HandleTask(context.Context, *CrawlerResult) (*HandleResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method HandleTask not implemented")
}
func (UnimplementedSpiderServiceServer) ControlSpiders(context.Context, *ControlRequest) (*ControlResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ControlSpiders not implemented")
}
func (UnimplementedSpiderServiceServer) GetSpidersStatus(context.Context, *StatusRequest) (*StatusResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetSpidersStatus not implemented")
}
func (UnimplementedSpiderServiceServer) mustEmbedUnimplementedSpiderServiceServer() {}
func (UnimplementedSpiderServiceServer) testEmbeddedByValue()                       {}

// UnsafeSpiderServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to SpiderServiceServer will
// result in compilation errors.
type UnsafeSpiderServiceServer interface {
	mustEmbedUnimplementedSpiderServiceServer()
}

func RegisterSpiderServiceServer(s grpc.ServiceRegistrar, srv SpiderServiceServer) {
	// If the following call pancis, it indicates UnimplementedSpiderServiceServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&SpiderService_ServiceDesc, srv)
}

func _SpiderService_GetTask_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(TaskRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SpiderServiceServer).GetTask(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: SpiderService_GetTask_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SpiderServiceServer).GetTask(ctx, req.(*TaskRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _SpiderService_HandleTask_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CrawlerResult)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SpiderServiceServer).HandleTask(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: SpiderService_HandleTask_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SpiderServiceServer).HandleTask(ctx, req.(*CrawlerResult))
	}
	return interceptor(ctx, in, info, handler)
}

func _SpiderService_ControlSpiders_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ControlRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SpiderServiceServer).ControlSpiders(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: SpiderService_ControlSpiders_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SpiderServiceServer).ControlSpiders(ctx, req.(*ControlRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _SpiderService_GetSpidersStatus_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(StatusRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SpiderServiceServer).GetSpidersStatus(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: SpiderService_GetSpidersStatus_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SpiderServiceServer).GetSpidersStatus(ctx, req.(*StatusRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// SpiderService_ServiceDesc is the grpc.ServiceDesc for SpiderService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var SpiderService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "spiders.SpiderService",
	HandlerType: (*SpiderServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetTask",
			Handler:    _SpiderService_GetTask_Handler,
		},
		{
			MethodName: "HandleTask",
			Handler:    _SpiderService_HandleTask_Handler,
		},
		{
			MethodName: "ControlSpiders",
			Handler:    _SpiderService_ControlSpiders_Handler,
		},
		{
			MethodName: "GetSpidersStatus",
			Handler:    _SpiderService_GetSpidersStatus_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "client.proto",
}