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
	`)
	if err != nil {
		t.Fatalf("failed to create test table: %v", err)
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
