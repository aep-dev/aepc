package service

import (
	"context"
	"fmt"
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	bpb "github.com/aep-dev/aepc/example/bookstore/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

var bookDatabase map[string]*bpb.Book

type BookstoreServer struct {
	bpb.UnimplementedBookstoreServer
}

func NewBookstoreServer() *BookstoreServer {
	return &BookstoreServer{}
}

func (BookstoreServer) CreateBook(_ context.Context, r *bpb.CreateBookRequest) (*bpb.Book, error) {
	book := proto.Clone(r.Book).(*bpb.Book)
	log.Printf("creating book %q", r)
	if r.Id == "" {
		r.Id = fmt.Sprintf("%v", len(bookDatabase)+1)
	}
	path := fmt.Sprintf("books/%v", r.Id)
	book.Id = r.Id
	book.Path = path
	bookDatabase[path] = book
	log.Printf("created book %q", path)
	return book, nil
}

func (BookstoreServer) ApplyBook(_ context.Context, r *bpb.ApplyBookRequest) (*bpb.Book, error) {
	log.Printf("applying book request: %v", r)
	originalResource := bookDatabase[r.Path]
	book := proto.Clone(r.Book).(*bpb.Book)
	book.Id = originalResource.Id
	book.Path = originalResource.Path
	bookDatabase[r.Path] = book
	log.Printf("applied book %q", book.Path)
	return book, nil
}

func (BookstoreServer) UpdateBook(_ context.Context, r *bpb.UpdateBookRequest) (*bpb.Book, error) {
	book := proto.Clone(r.Book).(*bpb.Book)
	book.Path = r.Path
	bookDatabase[r.Path] = book
	log.Printf("updated book %q at path %q", book, r.Path)
	return book, nil
}

func (BookstoreServer) DeleteBook(_ context.Context, r *bpb.DeleteBookRequest) (*emptypb.Empty, error) {
	delete(bookDatabase, r.Path)
	log.Printf("deleted book %q", r.Path)
	return &emptypb.Empty{}, nil
}

func (BookstoreServer) GetBook(_ context.Context, r *bpb.GetBookRequest) (*bpb.Book, error) {
	if b, found := bookDatabase[r.Path]; found {
		return b, nil
	}
	return nil, status.Errorf(codes.NotFound, "book %q not found", r.Path)
}

func (BookstoreServer) ListBook(_ context.Context, r *bpb.ListBookRequest) (*bpb.ListBookResponse, error) {
	var books []*bpb.Book
	for _, book := range bookDatabase {
		books = append(books, book)
	}
	return &bpb.ListBookResponse{
		Results: books,
	}, nil
}

func StartServer(targetPort int) {
	bookDatabase = make(map[string]*bpb.Book)
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
