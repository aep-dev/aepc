// main serves as a convenience function to start
// both the service and grpc gateway simultaneously.
package main

import (
	"github.com/aep-dev/aepc/example/gateway"
	"github.com/aep-dev/aepc/example/service"
)

func main() {
	go gateway.Run("localhost:9090")
	service.StartServer(9090)
}
