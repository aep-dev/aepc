// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             (unknown)
// source: example/bookstore/v1/bookstore.proto

package bookstore

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

const (
	Bookstore_CreateBook_FullMethodName = "/example.bookstore.v1.Bookstore/CreateBook"
	Bookstore_GetBook_FullMethodName    = "/example.bookstore.v1.Bookstore/GetBook"
	Bookstore_UpdateBook_FullMethodName = "/example.bookstore.v1.Bookstore/UpdateBook"
	Bookstore_DeleteBook_FullMethodName = "/example.bookstore.v1.Bookstore/DeleteBook"
	Bookstore_ListBook_FullMethodName   = "/example.bookstore.v1.Bookstore/ListBook"
	Bookstore_ApplyBook_FullMethodName  = "/example.bookstore.v1.Bookstore/ApplyBook"
)

// BookstoreClient is the client API for Bookstore service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type BookstoreClient interface {
	// An aep-compliant Create method for Book.
	CreateBook(ctx context.Context, in *CreateBookRequest, opts ...grpc.CallOption) (*Book, error)
	// An aep-compliant Get method for Book.
	GetBook(ctx context.Context, in *GetBookRequest, opts ...grpc.CallOption) (*Book, error)
	// An aep-compliant Update method for Book.
	UpdateBook(ctx context.Context, in *UpdateBookRequest, opts ...grpc.CallOption) (*Book, error)
	// An aep-compliant Delete method for Book.
	DeleteBook(ctx context.Context, in *DeleteBookRequest, opts ...grpc.CallOption) (*emptypb.Empty, error)
	// An aep-compliant List method for books.
	ListBook(ctx context.Context, in *ListBookRequest, opts ...grpc.CallOption) (*ListBookResponse, error)
	// An aep-compliant Apply method for books.
	ApplyBook(ctx context.Context, in *ApplyBookRequest, opts ...grpc.CallOption) (*Book, error)
}

type bookstoreClient struct {
	cc grpc.ClientConnInterface
}

func NewBookstoreClient(cc grpc.ClientConnInterface) BookstoreClient {
	return &bookstoreClient{cc}
}

func (c *bookstoreClient) CreateBook(ctx context.Context, in *CreateBookRequest, opts ...grpc.CallOption) (*Book, error) {
	out := new(Book)
	err := c.cc.Invoke(ctx, Bookstore_CreateBook_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *bookstoreClient) GetBook(ctx context.Context, in *GetBookRequest, opts ...grpc.CallOption) (*Book, error) {
	out := new(Book)
	err := c.cc.Invoke(ctx, Bookstore_GetBook_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *bookstoreClient) UpdateBook(ctx context.Context, in *UpdateBookRequest, opts ...grpc.CallOption) (*Book, error) {
	out := new(Book)
	err := c.cc.Invoke(ctx, Bookstore_UpdateBook_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *bookstoreClient) DeleteBook(ctx context.Context, in *DeleteBookRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, Bookstore_DeleteBook_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *bookstoreClient) ListBook(ctx context.Context, in *ListBookRequest, opts ...grpc.CallOption) (*ListBookResponse, error) {
	out := new(ListBookResponse)
	err := c.cc.Invoke(ctx, Bookstore_ListBook_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *bookstoreClient) ApplyBook(ctx context.Context, in *ApplyBookRequest, opts ...grpc.CallOption) (*Book, error) {
	out := new(Book)
	err := c.cc.Invoke(ctx, Bookstore_ApplyBook_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// BookstoreServer is the server API for Bookstore service.
// All implementations must embed UnimplementedBookstoreServer
// for forward compatibility
type BookstoreServer interface {
	// An aep-compliant Create method for Book.
	CreateBook(context.Context, *CreateBookRequest) (*Book, error)
	// An aep-compliant Get method for Book.
	GetBook(context.Context, *GetBookRequest) (*Book, error)
	// An aep-compliant Update method for Book.
	UpdateBook(context.Context, *UpdateBookRequest) (*Book, error)
	// An aep-compliant Delete method for Book.
	DeleteBook(context.Context, *DeleteBookRequest) (*emptypb.Empty, error)
	// An aep-compliant List method for books.
	ListBook(context.Context, *ListBookRequest) (*ListBookResponse, error)
	// An aep-compliant Apply method for books.
	ApplyBook(context.Context, *ApplyBookRequest) (*Book, error)
	mustEmbedUnimplementedBookstoreServer()
}

// UnimplementedBookstoreServer must be embedded to have forward compatible implementations.
type UnimplementedBookstoreServer struct {
}

func (UnimplementedBookstoreServer) CreateBook(context.Context, *CreateBookRequest) (*Book, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateBook not implemented")
}
func (UnimplementedBookstoreServer) GetBook(context.Context, *GetBookRequest) (*Book, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetBook not implemented")
}
func (UnimplementedBookstoreServer) UpdateBook(context.Context, *UpdateBookRequest) (*Book, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateBook not implemented")
}
func (UnimplementedBookstoreServer) DeleteBook(context.Context, *DeleteBookRequest) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DeleteBook not implemented")
}
func (UnimplementedBookstoreServer) ListBook(context.Context, *ListBookRequest) (*ListBookResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListBook not implemented")
}
func (UnimplementedBookstoreServer) ApplyBook(context.Context, *ApplyBookRequest) (*Book, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ApplyBook not implemented")
}
func (UnimplementedBookstoreServer) mustEmbedUnimplementedBookstoreServer() {}

// UnsafeBookstoreServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to BookstoreServer will
// result in compilation errors.
type UnsafeBookstoreServer interface {
	mustEmbedUnimplementedBookstoreServer()
}

func RegisterBookstoreServer(s grpc.ServiceRegistrar, srv BookstoreServer) {
	s.RegisterService(&Bookstore_ServiceDesc, srv)
}

func _Bookstore_CreateBook_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CreateBookRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BookstoreServer).CreateBook(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Bookstore_CreateBook_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BookstoreServer).CreateBook(ctx, req.(*CreateBookRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Bookstore_GetBook_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetBookRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BookstoreServer).GetBook(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Bookstore_GetBook_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BookstoreServer).GetBook(ctx, req.(*GetBookRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Bookstore_UpdateBook_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UpdateBookRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BookstoreServer).UpdateBook(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Bookstore_UpdateBook_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BookstoreServer).UpdateBook(ctx, req.(*UpdateBookRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Bookstore_DeleteBook_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DeleteBookRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BookstoreServer).DeleteBook(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Bookstore_DeleteBook_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BookstoreServer).DeleteBook(ctx, req.(*DeleteBookRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Bookstore_ListBook_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListBookRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BookstoreServer).ListBook(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Bookstore_ListBook_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BookstoreServer).ListBook(ctx, req.(*ListBookRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Bookstore_ApplyBook_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ApplyBookRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BookstoreServer).ApplyBook(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Bookstore_ApplyBook_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BookstoreServer).ApplyBook(ctx, req.(*ApplyBookRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// Bookstore_ServiceDesc is the grpc.ServiceDesc for Bookstore service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Bookstore_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "example.bookstore.v1.Bookstore",
	HandlerType: (*BookstoreServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "CreateBook",
			Handler:    _Bookstore_CreateBook_Handler,
		},
		{
			MethodName: "GetBook",
			Handler:    _Bookstore_GetBook_Handler,
		},
		{
			MethodName: "UpdateBook",
			Handler:    _Bookstore_UpdateBook_Handler,
		},
		{
			MethodName: "DeleteBook",
			Handler:    _Bookstore_DeleteBook_Handler,
		},
		{
			MethodName: "ListBook",
			Handler:    _Bookstore_ListBook_Handler,
		},
		{
			MethodName: "ApplyBook",
			Handler:    _Bookstore_ApplyBook_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "example/bookstore/v1/bookstore.proto",
}
