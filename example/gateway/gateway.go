package gateway

import (
	"context"
	"net/http"

	"github.com/golang/glog"
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
		glog.Fatal(err)
	}

	// Start HTTP server (and proxy calls to gRPC server endpoint)
	if err := http.ListenAndServe(":8081", mux); err != nil {
		glog.Fatal(err)
	}
}
