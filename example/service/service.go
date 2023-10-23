package main

import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	bpb "github.com/toumorokoshi/aep-sandbox/service/bookstore"
)

var bookDatabase map[string]*bpb.Book

type BookStoreServer struct {
	bpb.UnimplementedBookStoreServer
}

func NewBookStoreServer() *BookStoreServer {
	return &BookStoreServer{}
}

func (BookStoreServer) CreateBook(_ context.Context, r *bpb.CreateBookRequest) (*bpb.Book, error) {
	path := fmt.Sprintf("book/%v", r.Id)
	r.Resource.Path = path
	bookDatabase[path] = r.Resource
	return &bpb.Book{}, nil
}

func (BookStoreServer) ReadBook(_ context.Context, r *bpb.ReadBookRequest) (*bpb.Book, error) {
	if b, found := bookDatabase[r.Path]; found {
		return b, nil
	}
	return nil, status.Errorf(codes.NotFound, "book %q not found", r.Path)
}
