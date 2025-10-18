package service

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	lrpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	bpb "github.com/aep-dev/aepc/example/bookstore/v1"
	_ "github.com/mattn/go-sqlite3"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
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
		CREATE TABLE publishers (
			path TEXT PRIMARY KEY,
			description TEXT
		);
		CREATE TABLE deleted_publishers (
			path TEXT PRIMARY KEY,
			description TEXT,
			expire_time TEXT
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

func TestUpdateBookWithIfMatchHeader(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	s := NewBookstoreServer(db)

	// First, create a publisher
	publisher := &bpb.Publisher{
		Description: "Test Publisher",
	}
	createdPublisher, err := s.CreatePublisher(context.Background(), &bpb.CreatePublisherRequest{
		Id:        "1",
		Publisher: publisher,
	})
	if err != nil {
		t.Fatalf("CreatePublisher failed: %v", err)
	}

	// Then create a book
	book := &bpb.Book{
		Price:     10,
		Published: true,
		Edition:   1,
	}
	createdBook, err := s.CreateBook(context.Background(), &bpb.CreateBookRequest{
		Parent: createdPublisher.Path,
		Id:     "1",
		Book:   book,
	})
	if err != nil {
		t.Fatalf("CreateBook failed: %v", err)
	}

	// Get the current book to generate its ETag
	currentBook, err := s.GetBook(context.Background(), &bpb.GetBookRequest{
		Path: createdBook.Path,
	})
	if err != nil {
		t.Fatalf("GetBook failed: %v", err)
	}

	// Generate ETag for current book
	currentETag, err := GenerateETag(currentBook)
	if err != nil {
		t.Fatalf("GenerateETag failed: %v", err)
	}

	// Test 1: Update with correct If-Match header should succeed
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("grpcgateway-if-match", currentETag))
	updatedBook := &bpb.Book{
		Price:     20,
		Published: true,
		Edition:   2,
	}
	result, err := s.UpdateBook(ctx, &bpb.UpdateBookRequest{
		Path: createdBook.Path,
		Book: updatedBook,
	})
	if err != nil {
		t.Fatalf("UpdateBook with correct If-Match failed: %v", err)
	}
	if result.Price != 20 || result.Edition != 2 {
		t.Errorf("Update did not apply correctly: price=%d, edition=%d", result.Price, result.Edition)
	}

	// Test 2: Update with incorrect If-Match header should fail
	wrongETag := `"wrongetag"`
	ctx2 := metadata.NewIncomingContext(context.Background(), metadata.Pairs("grpcgateway-if-match", wrongETag))
	_, err = s.UpdateBook(ctx2, &bpb.UpdateBookRequest{
		Path: createdBook.Path,
		Book: updatedBook,
	})
	if err == nil {
		t.Fatalf("UpdateBook with incorrect If-Match should have failed")
	}
	if status.Code(err) != codes.FailedPrecondition {
		t.Errorf("Expected FailedPrecondition error, got: %v", status.Code(err))
	}

	// Test 3: Update without If-Match header should succeed (backwards compatibility)
	updatedBook.Price = 30
	_, err = s.UpdateBook(context.Background(), &bpb.UpdateBookRequest{
		Path: createdBook.Path,
		Book: updatedBook,
	})
	if err != nil {
		t.Fatalf("UpdateBook without If-Match should succeed: %v", err)
	}

	// Test 4: Update with If-Match header for non-existent resource should fail
	ctx3 := metadata.NewIncomingContext(context.Background(), metadata.Pairs("grpcgateway-if-match", currentETag))
	_, err = s.UpdateBook(ctx3, &bpb.UpdateBookRequest{
		Path: "publishers/99/books/99",
		Book: updatedBook,
	})
	if err == nil {
		t.Fatalf("UpdateBook for non-existent resource should have failed")
	}
	if status.Code(err) != codes.NotFound {
		t.Errorf("Expected NotFound error, got: %v", status.Code(err))
	}
}

func TestUpdatePublisherWithIfMatchHeader(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	s := NewBookstoreServer(db)

	// Create a publisher
	publisher := &bpb.Publisher{
		Description: "Original Description",
	}
	createdPublisher, err := s.CreatePublisher(context.Background(), &bpb.CreatePublisherRequest{
		Id:        "1",
		Publisher: publisher,
	})
	if err != nil {
		t.Fatalf("CreatePublisher failed: %v", err)
	}

	// Generate ETag for current publisher
	currentETag, err := GenerateETag(createdPublisher)
	if err != nil {
		t.Fatalf("GenerateETag failed: %v", err)
	}

	// Test update with correct If-Match header
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("grpcgateway-if-match", currentETag))
	updatedPublisher := &bpb.Publisher{
		Description: "Updated Description",
	}
	result, err := s.UpdatePublisher(ctx, &bpb.UpdatePublisherRequest{
		Path:      createdPublisher.Path,
		Publisher: updatedPublisher,
	})
	if err != nil {
		t.Fatalf("UpdatePublisher with correct If-Match failed: %v", err)
	}
	if result.Description != "Updated Description" {
		t.Errorf("Update did not apply correctly: description=%s", result.Description)
	}

	// Test update with incorrect If-Match header
	wrongETag := `"wrongetag"`
	ctx2 := metadata.NewIncomingContext(context.Background(), metadata.Pairs("grpcgateway-if-match", wrongETag))
	_, err = s.UpdatePublisher(ctx2, &bpb.UpdatePublisherRequest{
		Path:      createdPublisher.Path,
		Publisher: updatedPublisher,
	})
	if err == nil {
		t.Fatalf("UpdatePublisher with incorrect If-Match should have failed")
	}
	if status.Code(err) != codes.FailedPrecondition {
		t.Errorf("Expected FailedPrecondition error, got: %v", status.Code(err))
	}
}

func TestETagGeneration(t *testing.T) {
	// Test ETag generation for different resources
	book1 := &bpb.Book{
		Path:      "publishers/1/books/1",
		Price:     10,
		Published: true,
		Edition:   1,
	}

	book2 := &bpb.Book{
		Path:      "publishers/1/books/1",
		Price:     20, // Different price
		Published: true,
		Edition:   1,
	}

	book3 := &bpb.Book{
		Path:      "publishers/1/books/1",
		Price:     10,
		Published: true,
		Edition:   1,
	}

	etag1, err := GenerateETag(book1)
	if err != nil {
		t.Fatalf("GenerateETag failed: %v", err)
	}

	etag2, err := GenerateETag(book2)
	if err != nil {
		t.Fatalf("GenerateETag failed: %v", err)
	}

	etag3, err := GenerateETag(book3)
	if err != nil {
		t.Fatalf("GenerateETag failed: %v", err)
	}

	// ETags for different content should be different
	if etag1 == etag2 {
		t.Error("ETags should be different for different content")
	}

	// ETags for same content should be identical
	if etag1 != etag3 {
		t.Error("ETags should be identical for identical content")
	}

	// Test ETag validation
	if !ValidateETag(etag1, etag3) {
		t.Error("ValidateETag should return true for identical ETags")
	}

	if ValidateETag(etag1, etag2) {
		t.Error("ValidateETag should return false for different ETags")
	}

	// Test ETag validation with quotes
	quotedETag := `"` + strings.Trim(etag1, `"`) + `"`
	if !ValidateETag(etag1, quotedETag) {
		t.Error("ValidateETag should handle quoted ETags correctly")
	}

	// Test publisher ETags
	pub1 := &bpb.Publisher{
		Path:        "publishers/1",
		Description: "Test Publisher",
	}

	pub2 := &bpb.Publisher{
		Path:        "publishers/1",
		Description: "Updated Publisher", // Different description
	}

	pubEtag1, err := GenerateETag(pub1)
	if err != nil {
		t.Fatalf("GenerateETag for publisher failed: %v", err)
	}

	pubEtag2, err := GenerateETag(pub2)
	if err != nil {
		t.Fatalf("GenerateETag for publisher failed: %v", err)
	}

	if pubEtag1 == pubEtag2 {
		t.Error("Publisher ETags should be different for different descriptions")
	}

	// Test that ETags are properly quoted
	if !strings.HasPrefix(etag1, `"`) || !strings.HasSuffix(etag1, `"`) {
		t.Error("ETags should be properly quoted")
	}
}

func TestExtractIfMatchHeader(t *testing.T) {
	// Test extracting If-Match header from metadata
	testETag := `"test-etag-value"`

	// Test with grpcgateway-if-match key (this is what the gateway sets)
	ctx1 := metadata.NewIncomingContext(context.Background(),
		metadata.Pairs("grpcgateway-if-match", testETag))
	extractedETag1 := extractIfMatchHeader(ctx1)
	if extractedETag1 != testETag {
		t.Errorf("Expected ETag %s, got %s", testETag, extractedETag1)
	}

	// Test with standard if-match key
	ctx2 := metadata.NewIncomingContext(context.Background(),
		metadata.Pairs("if-match", testETag))
	extractedETag2 := extractIfMatchHeader(ctx2)
	if extractedETag2 != testETag {
		t.Errorf("Expected ETag %s, got %s", testETag, extractedETag2)
	}

	// Test with no metadata
	extractedETag3 := extractIfMatchHeader(context.Background())
	if extractedETag3 != "" {
		t.Errorf("Expected empty ETag, got %s", extractedETag3)
	}

	// Test with empty metadata
	ctx4 := metadata.NewIncomingContext(context.Background(), metadata.MD{})
	extractedETag4 := extractIfMatchHeader(ctx4)
	if extractedETag4 != "" {
		t.Errorf("Expected empty ETag, got %s", extractedETag4)
	}
}

func TestPublisherUndelete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	server := NewBookstoreServer(db)

	ctx := context.Background()

	// Create a publisher
	createResp, err := server.CreatePublisher(ctx, &bpb.CreatePublisherRequest{
		Id: "test-publisher",
		Publisher: &bpb.Publisher{
			Description: "Test Publisher for Undelete",
		},
	})
	if err != nil {
		t.Fatalf("Failed to create publisher: %v", err)
	}

	publisherPath := createResp.Path

	// Delete the publisher (should move to deleted_publishers)
	_, err = server.DeletePublisher(ctx, &bpb.DeletePublisherRequest{
		Path: publisherPath,
	})
	if err != nil {
		t.Fatalf("Failed to delete publisher: %v", err)
	}

	// Verify publisher is no longer in publishers table
	_, err = server.GetPublisher(ctx, &bpb.GetPublisherRequest{
		Path: publisherPath,
	})
	if err == nil {
		t.Fatal("Expected publisher to be deleted, but it still exists")
	}
	if status.Code(err) != codes.NotFound {
		t.Fatalf("Expected NotFound error, got %v", err)
	}

	// Verify publisher is in deleted_publishers table
	deletedPath := strings.Replace(publisherPath, "publishers/", "deleted_publishers/", 1)
	deletedPublisher, err := server.GetDeletedPublisher(ctx, &bpb.GetDeletedPublisherRequest{
		Path: deletedPath,
	})
	if err != nil {
		t.Fatalf("Failed to get deleted publisher: %v", err)
	}
	if deletedPublisher.Description != "Test Publisher for Undelete" {
		t.Errorf("Expected description 'Test Publisher for Undelete', got %q", deletedPublisher.Description)
	}
	if deletedPublisher.ExpireTime == "" {
		t.Error("Expected expire_time to be set")
	}

	// List deleted publishers
	listResp, err := server.ListDeletedPublishers(ctx, &bpb.ListDeletedPublishersRequest{})
	if err != nil {
		t.Fatalf("Failed to list deleted publishers: %v", err)
	}
	if len(listResp.Results) != 1 {
		t.Errorf("Expected 1 deleted publisher, got %d", len(listResp.Results))
	}

	// Undelete the publisher
	_, err = server.UndeleteDeletedPublisher(ctx, &bpb.UndeleteDeletedPublisherRequest{
		Path: deletedPath,
	})
	if err != nil {
		t.Fatalf("Failed to undelete publisher: %v", err)
	}

	// Verify publisher is restored in publishers table
	restoredPublisher, err := server.GetPublisher(ctx, &bpb.GetPublisherRequest{
		Path: publisherPath,
	})
	if err != nil {
		t.Fatalf("Failed to get restored publisher: %v", err)
	}
	if restoredPublisher.Description != "Test Publisher for Undelete" {
		t.Errorf("Expected description 'Test Publisher for Undelete', got %q", restoredPublisher.Description)
	}

	// Verify publisher is no longer in deleted_publishers table
	_, err = server.GetDeletedPublisher(ctx, &bpb.GetDeletedPublisherRequest{
		Path: deletedPath,
	})
	if err == nil {
		t.Fatal("Expected deleted publisher to be removed, but it still exists")
	}
	if status.Code(err) != codes.NotFound {
		t.Fatalf("Expected NotFound error, got %v", err)
	}

	// Test undeleting non-existent deleted publisher
	_, err = server.UndeleteDeletedPublisher(ctx, &bpb.UndeleteDeletedPublisherRequest{
		Path: "deleted_publishers/non-existent",
	})
	if err == nil {
		t.Fatal("Expected error when undeleting non-existent publisher")
	}
	if status.Code(err) != codes.NotFound {
		t.Fatalf("Expected NotFound error, got %v", err)
	}

	// Test undeleting when publisher already exists (conflict)
	// Create another publisher first
	_, err = server.CreatePublisher(ctx, &bpb.CreatePublisherRequest{
		Id: "conflict-publisher",
		Publisher: &bpb.Publisher{
			Description: "Conflict Test Publisher",
		},
	})
	if err != nil {
		t.Fatalf("Failed to create conflict publisher: %v", err)
	}

	// Manually insert into deleted_publishers with same ID
	_, err = db.Exec("INSERT INTO deleted_publishers (path, description, expire_time) VALUES (?, ?, ?)",
		"deleted_publishers/conflict-publisher", "Old Conflict Publisher", time.Now().Add(30*24*time.Hour).Format(time.RFC3339))
	if err != nil {
		t.Fatalf("Failed to insert conflict deleted publisher: %v", err)
	}

	// Try to undelete - should fail with AlreadyExists
	_, err = server.UndeleteDeletedPublisher(ctx, &bpb.UndeleteDeletedPublisherRequest{
		Path: "deleted_publishers/conflict-publisher",
	})
	if err == nil {
		t.Fatal("Expected error when undeleting publisher that already exists")
	}
	if status.Code(err) != codes.AlreadyExists {
		t.Fatalf("Expected AlreadyExists error, got %v", err)
	}
}
