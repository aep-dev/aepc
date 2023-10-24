package service

import (
	"context"
	"fmt"
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	bpb "github.com/aep-dev/aepc/example/bookstore"
)

var bookDatabase map[string]*bpb.Book

type BookstoreServer struct {
	bpb.UnimplementedBookstoreServer
}

func NewBookstoreServer() *BookstoreServer {
	return &BookstoreServer{}
}

func (BookstoreServer) CreateBook(_ context.Context, r *bpb.CreateBookRequest) (*bpb.Book, error) {
	path := fmt.Sprintf("book/%v", r.Id)
	r.Resource.Path = path
	bookDatabase[path] = r.Resource
	return &bpb.Book{}, nil
}

func (BookstoreServer) ReadBook(_ context.Context, r *bpb.ReadBookRequest) (*bpb.Book, error) {
	if b, found := bookDatabase[r.Path]; found {
		return b, nil
	}
	return nil, status.Errorf(codes.NotFound, "book %q not found", r.Path)
}

func StartServer(targetPort int) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", targetPort))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	bpb.RegisterBookstoreServer(s, NewBookstoreServer())
	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
