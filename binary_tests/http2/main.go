package main

import (
	"flag"
	"fmt"
	"log"
	"net"

	"github.com/EIRNf/notnets_grpc/test_hello_service"
	"google.golang.org/grpc"
)

var (
	port = flag.Int("port", 50051, "The server port")
)

func main() {

	// debug.SetGCPercent(-1)
	// runtime.MemProfileRate = 1

	svc := &test_hello_service.TestServer{}
	flag.Parse()
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	svr := grpc.NewServer()
	test_hello_service.RegisterTestServiceServer(svr, svc)

	log.Printf("server listening at %v", lis.Addr())
	if err := svr.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}

	defer svr.Stop()

}
