package service

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"log/slog"
	"net"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	_ "buf.build/gen/go/aep/api/protocolbuffers/go/aep/api"
	api "buf.build/gen/go/aep/api/protocolbuffers/go/aep/api"
	lrpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	bpb "github.com/aep-dev/aepc/example/bookstore/v1"
	_ "github.com/mattn/go-sqlite3" // sqlite3 driver
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/emptypb"
)

type operationStatus struct {
	Done   bool
	Result interface{}
	Error  error
}

type operationStore struct {
	mu         sync.Mutex
	operations map[string]*operationStatus
}

var opStore = &operationStore{
	operations: make(map[string]*operationStatus),
}

func (s *operationStore) createOperation(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.operations[id] = &operationStatus{Done: false}
}

func (s *operationStore) completeOperation(id string, result interface{}, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if op, exists := s.operations[id]; exists {
		op.Done = true
		op.Result = result
		op.Error = err
	}
}

func (s *operationStore) getOperation(id string) (*operationStatus, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	op, exists := s.operations[id]
	return op, exists
}

type BookstoreServer struct {
	bpb.UnimplementedBookstoreServer
	lrpb.UnimplementedOperationsServer
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

func (s BookstoreServer) ArchiveBook(ctx context.Context, r *bpb.ArchiveBookRequest) (*api.Operation, error) {
	log.Printf("archiving book %q", r.Path)

	operationID := fmt.Sprintf("op-%d", time.Now().UnixNano())
	opStore.createOperation(operationID)

	go func() {
		// Simulate the archiving process
		result, err := s.db.Exec(`
			UPDATE books
			SET published = false
			WHERE path = ?`,
			r.Path)
		if err != nil {
			opStore.completeOperation(operationID, nil, status.Errorf(codes.Internal, "failed to archive book: %v", err))
			return
		}

		rows, err := result.RowsAffected()
		if err != nil {
			opStore.completeOperation(operationID, nil, status.Errorf(codes.Internal, "failed to get rows affected: %v", err))
			return
		}
		if rows == 0 {
			opStore.completeOperation(operationID, nil, status.Errorf(codes.NotFound, "book %q not found", r.Path))
			return
		}

		opStore.completeOperation(operationID, &anypb.Any{}, nil)
	}()

	return &api.Operation{Path: operationID, Done: false}, nil
}

func (s BookstoreServer) GetOperation(ctx context.Context, r *lrpb.GetOperationRequest) (*lrpb.Operation, error) {
	op, exists := opStore.getOperation(r.Name)
	if !exists {
		return nil, status.Errorf(codes.NotFound, "operation %q not found", r.Name)
	}

	operation := &lrpb.Operation{
		Name: r.Name,
		Done: op.Done,
	}

	if op.Error != nil {
		operation.Result = &lrpb.Operation_Error{
			Error: status.Convert(op.Error).Proto(),
		}
	} else {
		operation.Result = &lrpb.Operation_Response{
			Response: op.Result.(*anypb.Any),
		}
	}

	return operation, nil
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

func (s BookstoreServer) CreateStore(_ context.Context, r *bpb.CreateStoreRequest) (*bpb.Store, error) {
	store := proto.Clone(r.Store).(*bpb.Store)
	log.Printf("creating store %q", r)
	if r.Id == "" {
		var maxID int
		err := s.db.QueryRow("SELECT COALESCE(MAX(CAST(SUBSTR(path, 8) AS INTEGER)), 0) FROM stores").Scan(&maxID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to generate ID: %v", err)
		}
		r.Id = fmt.Sprintf("%d", maxID+1)
	}
	path := fmt.Sprintf("stores/%v", r.Id)
	store.Path = path

	_, err := s.db.Exec(`
		INSERT INTO stores (path, name, description)
		VALUES (?, ?, ?)`,
		store.Path, store.Name, store.Description)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create store: %v", err)
	}

	log.Printf("created store %q", path)
	return store, nil
}

func (s BookstoreServer) GetStore(_ context.Context, r *bpb.GetStoreRequest) (*bpb.Store, error) {
	store := &bpb.Store{}
	err := s.db.QueryRow(`
		SELECT path, name, description
		FROM stores WHERE path = ?`, r.Path).Scan(
		&store.Path, &store.Name, &store.Description)

	if err == sql.ErrNoRows {
		return nil, status.Errorf(codes.NotFound, "store %q not found", r.Path)
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get store: %v", err)
	}
	return store, nil
}

func (s BookstoreServer) UpdateStore(_ context.Context, r *bpb.UpdateStoreRequest) (*bpb.Store, error) {
	store := proto.Clone(r.Store).(*bpb.Store)
	store.Path = r.Path

	result, err := s.db.Exec(`
		UPDATE stores
		SET name = ?, description = ?
		WHERE path = ?`,
		store.Name, store.Description, store.Path)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update store: %v", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get rows affected: %v", err)
	}
	if rows == 0 {
		return nil, status.Errorf(codes.NotFound, "store %q not found", r.Path)
	}

	log.Printf("updated store %q", store.Path)
	return store, nil
}

func (s BookstoreServer) DeleteStore(_ context.Context, r *bpb.DeleteStoreRequest) (*emptypb.Empty, error) {
	result, err := s.db.Exec("DELETE FROM stores WHERE path = ?", r.Path)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete store: %v", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get rows affected: %v", err)
	}
	if rows == 0 {
		return nil, status.Errorf(codes.NotFound, "store %q not found", r.Path)
	}

	log.Printf("deleted store %q", r.Path)
	return &emptypb.Empty{}, nil
}

func (s BookstoreServer) CreateItem(_ context.Context, r *bpb.CreateItemRequest) (*bpb.Item, error) {
	item := proto.Clone(r.Item).(*bpb.Item)
	log.Printf("creating item %q", r)
	if r.Id == "" {
		var maxID int
		err := s.db.QueryRow("SELECT COALESCE(MAX(CAST(SUBSTR(path, 14) AS INTEGER)), 0) FROM items").Scan(&maxID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to generate ID: %v", err)
		}
		r.Id = fmt.Sprintf("%d", maxID+1)
	}
	path := fmt.Sprintf("stores/%v/items/%v", r.Parent, r.Id)
	item.Path = path

	_, err := s.db.Exec(`
		INSERT INTO items (path, book, condition, price)
		VALUES (?, ?, ?, ?)`,
		item.Path, item.Book, item.Condition, item.Price)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create item: %v", err)
	}

	log.Printf("created item %q", path)
	return item, nil
}

func (s BookstoreServer) GetItem(_ context.Context, r *bpb.GetItemRequest) (*bpb.Item, error) {
	item := &bpb.Item{}
	err := s.db.QueryRow(`
		SELECT path, book, condition, price
		FROM items WHERE path = ?`, r.Path).Scan(
		&item.Path, &item.Book, &item.Condition, &item.Price)

	if err == sql.ErrNoRows {
		return nil, status.Errorf(codes.NotFound, "item %q not found", r.Path)
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get item: %v", err)
	}
	return item, nil
}

func (s BookstoreServer) UpdateItem(_ context.Context, r *bpb.UpdateItemRequest) (*bpb.Item, error) {
	item := proto.Clone(r.Item).(*bpb.Item)
	item.Path = r.Path

	result, err := s.db.Exec(`
		UPDATE items
		SET book = ?, condition = ?, price = ?
		WHERE path = ?`,
		item.Book, item.Condition, item.Price, item.Path)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update item: %v", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get rows affected: %v", err)
	}
	if rows == 0 {
		return nil, status.Errorf(codes.NotFound, "item %q not found", r.Path)
	}

	log.Printf("updated item %q", item.Path)
	return item, nil
}

func (s BookstoreServer) DeleteItem(_ context.Context, r *bpb.DeleteItemRequest) (*emptypb.Empty, error) {
	result, err := s.db.Exec("DELETE FROM items WHERE path = ?", r.Path)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete item: %v", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get rows affected: %v", err)
	}
	if rows == 0 {
		return nil, status.Errorf(codes.NotFound, "item %q not found", r.Path)
	}

	log.Printf("deleted item %q", r.Path)
	return &emptypb.Empty{}, nil
}

func (s BookstoreServer) MoveItem(ctx context.Context, r *bpb.MoveItemRequest) (*api.Operation, error) {
	log.Printf("moving item %q to store %q", r.Path, r.TargetStore)

	operationID := fmt.Sprintf("op-%d", time.Now().UnixNano())
	opStore.createOperation(operationID)

	go func() {
		// Simulate the moving process
		result, err := s.db.Exec(`
			UPDATE items
			SET path = ?
			WHERE path = ?`,
			fmt.Sprintf("%s/items/%s", r.TargetStore, r.Path[strings.LastIndex(r.Path, "/")+1:]), r.Path)
		if err != nil {
			opStore.completeOperation(operationID, nil, status.Errorf(codes.Internal, "failed to move item: %v", err))
			return
		}

		rows, err := result.RowsAffected()
		if err != nil {
			opStore.completeOperation(operationID, nil, status.Errorf(codes.Internal, "failed to get rows affected: %v", err))
			return
		}
		if rows == 0 {
			opStore.completeOperation(operationID, nil, status.Errorf(codes.NotFound, "item %q not found", r.Path))
			return
		}

		opStore.completeOperation(operationID, &anypb.Any{}, nil)
	}()

	return &api.Operation{Path: operationID, Done: false}, nil
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
		CREATE TABLE IF NOT EXISTS stores (
			path TEXT PRIMARY KEY,
			name TEXT,
			description TEXT
		);
		CREATE TABLE IF NOT EXISTS items (
			path TEXT PRIMARY KEY,
			book TEXT,
			condition TEXT,
			price REAL
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
