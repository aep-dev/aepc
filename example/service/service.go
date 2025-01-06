package service

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"log/slog"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	bpb "github.com/aep-dev/aepc/example/bookstore/v1"
	_ "github.com/mattn/go-sqlite3" // sqlite3 driver
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

type BookstoreServer struct {
	bpb.UnimplementedBookstoreServer
	db *sql.DB
}

func NewBookstoreServer(db *sql.DB) *BookstoreServer {
	return &BookstoreServer{db: db}
}

func (s BookstoreServer) CreateBook(_ context.Context, r *bpb.CreateBookRequest) (*bpb.Book, error) {
	book := proto.Clone(r.Book).(*bpb.Book)
	log.Printf("creating book %q", r)
	if r.Id == "" {
		var maxID int
		err := s.db.QueryRow("SELECT COALESCE(MAX(CAST(SUBSTR(path, INSTR(path, '/books/') + 7) AS INTEGER)), 0) FROM books").Scan(&maxID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to generate ID: %v", err)
		}
		r.Id = fmt.Sprintf("%d", maxID+1)
	}
	path := fmt.Sprintf("%v/books/%v", r.Parent, r.Id)
	book.Path = path

	_, err := s.db.Exec(`
		INSERT INTO books (path, author, price, published, edition, isbn)
		VALUES (?, ?, ?, ?, ?, ?)`,
		book.Path, book.Author, book.Price, book.Published, book.Edition, book.Isbn)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create book: %v", err)
	}

	log.Printf("created book %q", path)
	return book, nil
}

func (s BookstoreServer) ApplyBook(_ context.Context, r *bpb.ApplyBookRequest) (*bpb.Book, error) {
	log.Printf("applying book request: %v", r)
	book := proto.Clone(r.Book).(*bpb.Book)
	book.Path = r.Path

	result, err := s.db.Exec(`
		INSERT INTO books (path, author, price, published, edition, isbn)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(path) DO UPDATE SET
			author = excluded.author,
			price = excluded.price,
			published = excluded.published,
			edition = excluded.edition,
			isbn = excluded.isbn`,
		book.Path, book.Author, book.Price, book.Published, book.Edition, book.Isbn)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to apply book: %v", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get rows affected: %v", err)
	}
	if rows == 0 {
		return nil, status.Errorf(codes.NotFound, "book %q not found", r.Path)
	}

	log.Printf("applied book %q", book.Path)
	return book, nil
}

func (s BookstoreServer) UpdateBook(_ context.Context, r *bpb.UpdateBookRequest) (*bpb.Book, error) {
	book := proto.Clone(r.Book).(*bpb.Book)
	book.Path = r.Path

	result, err := s.db.Exec(`
		UPDATE books
		SET author = ?, price = ?, published = ?, edition = ?
		WHERE path = ?`,
		book.Author, book.Price, book.Published, book.Edition, book.Path)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update book: %v", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get rows affected: %v", err)
	}
	if rows == 0 {
		return nil, status.Errorf(codes.NotFound, "book %q not found", r.Path)
	}

	log.Printf("updated book %q", book.Path)
	return book, nil
}

func (s BookstoreServer) DeleteBook(_ context.Context, r *bpb.DeleteBookRequest) (*emptypb.Empty, error) {
	result, err := s.db.Exec("DELETE FROM books WHERE path = ?", r.Path)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete book: %v", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get rows affected: %v", err)
	}
	if rows == 0 {
		return nil, status.Errorf(codes.NotFound, "book %q not found", r.Path)
	}

	log.Printf("deleted book %q", r.Path)
	return &emptypb.Empty{}, nil
}

func (s BookstoreServer) GetBook(_ context.Context, r *bpb.GetBookRequest) (*bpb.Book, error) {
	book := &bpb.Book{}
	err := s.db.QueryRow(`
		SELECT path, author, price, published, edition
		FROM books WHERE path = ?`, r.Path).Scan(
		&book.Path, &book.Author, &book.Price, &book.Published, &book.Edition)

	if err == sql.ErrNoRows {
		return nil, status.Errorf(codes.NotFound, "book %q not found", r.Path)
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get book: %v", err)
	}
	return book, nil
}

func (s BookstoreServer) ListBooks(_ context.Context, r *bpb.ListBooksRequest) (*bpb.ListBooksResponse, error) {
	rows, err := s.db.Query(`
		SELECT path, author, price, published, edition
		FROM books`)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list books: %v", err)
	}
	defer rows.Close()

	var books []*bpb.Book
	for rows.Next() {
		book := &bpb.Book{}
		if err := rows.Scan(&book.Path, &book.Author, &book.Price, &book.Published, &book.Edition); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to scan book: %v", err)
		}
		books = append(books, book)
	}
	if err = rows.Err(); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to iterate books: %v", err)
	}

	return &bpb.ListBooksResponse{Results: books}, nil
}

func (s BookstoreServer) CreatePublisher(_ context.Context, r *bpb.CreatePublisherRequest) (*bpb.Publisher, error) {
	publisher := proto.Clone(r.Publisher).(*bpb.Publisher)
	log.Printf("creating publisher %q", r)
	if r.Id == "" {
		var maxID int
		err := s.db.QueryRow("SELECT COALESCE(MAX(CAST(SUBSTR(path, 12) AS INTEGER)), 0) FROM publishers").Scan(&maxID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to generate ID: %v", err)
		}
		r.Id = fmt.Sprintf("%d", maxID+1)
	}
	path := fmt.Sprintf("publishers/%v", r.Id)
	publisher.Path = path

	_, err := s.db.Exec(`
		INSERT INTO publishers (path, description)
		VALUES (?, ?)`,
		publisher.Path, publisher.Description)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create publisher: %v", err)
	}

	log.Printf("created publisher %q", path)
	return publisher, nil
}

func (s BookstoreServer) ApplyPublisher(_ context.Context, r *bpb.ApplyPublisherRequest) (*bpb.Publisher, error) {
	log.Printf("applying publisher request: %v", r)
	publisher := proto.Clone(r.Publisher).(*bpb.Publisher)
	publisher.Path = r.Path

	result, err := s.db.Exec(`
		INSERT INTO publishers (path, description)
		VALUES (?, ?)
		ON CONFLICT(path) DO UPDATE SET
			description = excluded.description`,
		publisher.Path, publisher.Description)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to apply publisher: %v", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get rows affected: %v", err)
	}
	if rows == 0 {
		return nil, status.Errorf(codes.NotFound, "publisher %q not found", r.Path)
	}

	log.Printf("applied publisher %q", publisher.Path)
	return publisher, nil
}

func (s BookstoreServer) UpdatePublisher(_ context.Context, r *bpb.UpdatePublisherRequest) (*bpb.Publisher, error) {
	publisher := proto.Clone(r.Publisher).(*bpb.Publisher)
	publisher.Path = r.Path

	result, err := s.db.Exec(`
		UPDATE publishers
		SET description = ?
		WHERE path = ?`,
		publisher.Description, publisher.Path)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update publisher: %v", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get rows affected: %v", err)
	}
	if rows == 0 {
		return nil, status.Errorf(codes.NotFound, "publisher %q not found", r.Path)
	}

	log.Printf("updated publisher %q", publisher.Path)
	return publisher, nil
}

func (s BookstoreServer) DeletePublisher(_ context.Context, r *bpb.DeletePublisherRequest) (*emptypb.Empty, error) {
	result, err := s.db.Exec("DELETE FROM publishers WHERE path = ?", r.Path)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete publisher: %v", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get rows affected: %v", err)
	}
	if rows == 0 {
		return nil, status.Errorf(codes.NotFound, "publisher %q not found", r.Path)
	}

	log.Printf("deleted publisher %q", r.Path)
	return &emptypb.Empty{}, nil
}

func (s BookstoreServer) GetPublisher(_ context.Context, r *bpb.GetPublisherRequest) (*bpb.Publisher, error) {
	publisher := &bpb.Publisher{}
	err := s.db.QueryRow(`
		SELECT path, description
		FROM publishers WHERE path = ?`, r.Path).Scan(
		&publisher.Path, &publisher.Description)

	if err == sql.ErrNoRows {
		return nil, status.Errorf(codes.NotFound, "publisher %q not found", r.Path)
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get publisher: %v", err)
	}
	return publisher, nil
}

func (s BookstoreServer) ListPublishers(_ context.Context, r *bpb.ListPublishersRequest) (*bpb.ListPublishersResponse, error) {
	skip := r.GetSkip()
	condition, err := convertCELToSQL(r.GetFilter())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert filter: %v", err)
	}
	if condition != "" {
		condition = "WHERE " + condition
	}
	slog.Info("list query on publishers", "condition", condition)
	rows, err := s.db.Query(`
			SELECT path, description
			FROM publishers
			`+condition+`
			LIMIT 10 OFFSET ?`, skip)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list publishers: %v", err)
	}
	defer rows.Close()

	var publishers []*bpb.Publisher
	for rows.Next() {
		publisher := &bpb.Publisher{}
		if err := rows.Scan(&publisher.Path, &publisher.Description); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to scan publisher: %v", err)
		}
		publishers = append(publishers, publisher)
	}
	if err = rows.Err(); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to iterate publishers: %v", err)
	}

	return &bpb.ListPublishersResponse{Results: publishers}, nil
}

func StartServer(targetPort int) {
	db, err := sql.Open("sqlite3", "/tmp/bookstore.db")
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS books (
			path TEXT PRIMARY KEY,
			author TEXT,
			price REAL,
			published BOOLEAN,
			edition INTEGER,
			isbn TEXT
		);
		CREATE TABLE IF NOT EXISTS publishers (
			path TEXT PRIMARY KEY,
			description TEXT
		);
	`)
	if err != nil {
		log.Fatalf("failed to create tables: %v", err)
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", targetPort))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	bpb.RegisterBookstoreServer(s, NewBookstoreServer(db))
	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
