package gateway

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	bpb "github.com/aep-dev/aepc/example/bookstore"
)

// gRPC server endpoint
func Run(grpcServerEndpoint string) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Register gRPC server endpoint
	// Note: Make sure the gRPC server is running properly and accessible
	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	err := bpb.RegisterBookstoreHandlerFromEndpoint(ctx, mux, grpcServerEndpoint, opts)
	if err != nil {
		log.Fatal(err)
	}
	if err != nil {
		fmt.Println("Error getting executable path:", err)
		return
	}

	// Construct the relative path to the data file
	dataFilePath := filepath.Join("example/bookstore/bookstore.yaml.output.openapi.json")

	// Read the file contents
	data, err := os.ReadFile(dataFilePath)
	if err != nil {
		fmt.Println("Error reading file:", err)
		return
	}
	err = mux.HandlePath("GET", "/openapi.json", func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
		w.Write(data)
	})
	if err != nil {
		log.Fatal(err)
	}

	loggingWrappedMux := loggingMiddleware(mux)

	log.Print("Starting grpc-gateway server on localhost:8081")
	// Start HTTP server (and proxy calls to gRPC server endpoint)
	if err := http.ListenAndServe(":8081", loggingWrappedMux); err != nil {
		log.Fatal(err)
	}
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. Read the body while creating a copy
		bodyBytes, _ := io.ReadAll(r.Body)
		r.Body.Close()                                    // Close the original body
		r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes)) // Replace with a re-readable buffer

		// 2. Log request details (customize as needed)
		log.Printf("Request: Method=%s, Path=%s, RemoteAddr=%s, Body=%s", r.Method, r.URL, r.RemoteAddr, string(bodyBytes))

		bodyReader := bytes.NewReader(bodyBytes)
		// Replace the original body with the TeeReader
		r.Body = io.NopCloser(bodyReader)

		// Let the next handler read the body again
		next.ServeHTTP(w, r)
	})
}

// Custom responseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
}
