package service

import (
	"context"
	"database/sql"
	"testing"
	"time"

	lrpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	bpb "github.com/aep-dev/aepc/example/bookstore/v1"
	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	_, err = db.Exec(`
		CREATE TABLE books (
			path TEXT PRIMARY KEY,
			author TEXT,
			price REAL,
			published BOOLEAN,
			edition INTEGER,
			isbn TEXT
		);
		CREATE TABLE stores (
			path TEXT PRIMARY KEY,
			name TEXT,
			description TEXT
		);
		CREATE TABLE items (
			path TEXT PRIMARY KEY,
			book TEXT,
			condition TEXT,
			price REAL
		);
	`)
	if err != nil {
		t.Fatalf("failed to create test tables: %v", err)
	}

	return db
}

func TestArchiveBookOperation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	s := NewBookstoreServer(db)

	// Insert a test book
	_, err := db.Exec(`
		INSERT INTO books (path, author, price, published, edition, isbn)
		VALUES (?, ?, ?, ?, ?, ?)
	`, "publishers/1/books/1", "Author Name", 19.99, true, 1, "1234567890")
	if err != nil {
		t.Fatalf("failed to insert test book: %v", err)
	}

	// Archive the book
	r := &bpb.ArchiveBookRequest{Path: "publishers/1/books/1"}
	operation, err := s.ArchiveBook(context.Background(), r)
	if err != nil {
		t.Fatalf("ArchiveBook failed: %v", err)
	}

	if operation.Done {
		t.Fatalf("expected operation to not be done immediately, got true")
	}

	// Simulate waiting for the operation to complete
	time.Sleep(100 * time.Millisecond)

	// Check operation status
	getOpReq := &lrpb.GetOperationRequest{Name: operation.Path}
	opResp, err := s.GetOperation(context.Background(), getOpReq)
	if err != nil {
		t.Fatalf("GetOperation failed: %v", err)
	}

	if !opResp.Done {
		t.Fatalf("expected operation to be done, got false")
	}

	// Verify the book is archived
	var published bool
	err = db.QueryRow("SELECT published FROM books WHERE path = ?", "publishers/1/books/1").Scan(&published)
	if err != nil {
		t.Fatalf("failed to query book: %v", err)
	}

	if published {
		t.Fatalf("expected published to be false, got true")
	}
}

func TestCreateStore(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	s := NewBookstoreServer(db)

	r := &bpb.CreateStoreRequest{
		Store: &bpb.Store{
			Name:        "Test Store",
			Description: "A test store",
		},
	}

	store, err := s.CreateStore(context.Background(), r)
	if err != nil {
		t.Fatalf("CreateStore failed: %v", err)
	}

	if store.Name != "Test Store" {
		t.Fatalf("expected store name to be 'Test Store', got %q", store.Name)
	}
}

func TestCreateItem(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	s := NewBookstoreServer(db)

	r := &bpb.CreateItemRequest{
		Parent: "stores/1",
		Item: &bpb.Item{
			Book:      "publishers/1/books/1",
			Condition: "New",
			Price:     29.99,
		},
	}

	item, err := s.CreateItem(context.Background(), r)
	if err != nil {
		t.Fatalf("CreateItem failed: %v", err)
	}

	if item.Condition != "New" {
		t.Fatalf("expected item condition to be 'New', got %q", item.Condition)
	}
}

func TestMoveItem(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	s := NewBookstoreServer(db)

	// Insert a test item
	_, err := db.Exec(`
		INSERT INTO items (path, book, condition, price)
		VALUES (?, ?, ?, ?)
	`, "stores/1/items/1", "publishers/1/books/1", "New", 29.99)
	if err != nil {
		t.Fatalf("failed to insert test item: %v", err)
	}

	// Move the item
	r := &bpb.MoveItemRequest{
		Path:        "stores/1/items/1",
		TargetStore: "stores/2",
	}
	operation, err := s.MoveItem(context.Background(), r)
	if err != nil {
		t.Fatalf("MoveItem failed: %v", err)
	}

	if operation.Done {
		t.Fatalf("expected operation to not be done immediately, got true")
	}

	// Simulate waiting for the operation to complete
	time.Sleep(100 * time.Millisecond)

	// Check operation status
	getOpReq := &lrpb.GetOperationRequest{Name: operation.Path}
	opResp, err := s.GetOperation(context.Background(), getOpReq)
	if err != nil {
		t.Fatalf("GetOperation failed: %v", err)
	}

	if !opResp.Done {
		t.Fatalf("expected operation to be done, got false")
	}

	// Verify the item is moved
	var newPath string
	err = db.QueryRow("SELECT path FROM items WHERE path = ?", "stores/2/items/1").Scan(&newPath)
	if err != nil {
		t.Fatalf("failed to query moved item: %v", err)
	}

	if newPath != "stores/2/items/1" {
		t.Fatalf("expected new path to be 'stores/2/items/1', got %q", newPath)
	}
}

func TestListBooksByPublisher(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	s := NewBookstoreServer(db)

	// Serialize authors and ISBNs for test data
	authorOneSerialized := `[{"name":"Author One"}]`
	authorTwoSerialized := `[{"name":"Author Two"}]`
	isbnOneSerialized := `["1111111111"]`
	isbnTwoSerialized := `["2222222222"]`

	// Insert test books
	_, err := db.Exec(`
		INSERT INTO books (path, author, price, published, edition, isbn)
		VALUES (?, ?, ?, ?, ?, ?), (?, ?, ?, ?, ?, ?)`,
		"publishers/1/books/1", authorOneSerialized, 10, true, 1, isbnOneSerialized,
		"publishers/2/books/2", authorTwoSerialized, 15, true, 1, isbnTwoSerialized,
	)
	if err != nil {
		t.Fatalf("failed to insert test books: %v", err)
	}

	// Test filtering by publisher 1
	resp, err := s.ListBooks(context.Background(), &bpb.ListBooksRequest{Parent: "publishers/1"})
	if err != nil {
		t.Fatalf("ListBooks failed: %v", err)
	}

	if len(resp.Results) != 1 {
		t.Fatalf("expected 1 book, got %d", len(resp.Results))
	}

	if resp.Results[0].Path != "publishers/1/books/1" {
		t.Errorf("unexpected book path: %s", resp.Results[0].Path)
	}
}
