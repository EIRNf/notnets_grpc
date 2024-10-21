package main

import (
	"log"

	"github.com/EIRNf/notnets_grpc"

	chaintests "github.com/EIRNf/notnets_grpc/chain_tests"
)

func main() {

	cc1, err := notnets_grpc.Dial("http://127.0.0.1:8080/hello", "http://127.0.0.1:8080/goodbye", notnets_grpc.MESSAGE_SIZE)
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}

	svc := &chaintests.TestServer{
		HelloClient: chaintests.NewTestServiceClient(cc1),
	}
	// svc := &channel_tests_service.TestServer{}
	svr := notnets_grpc.NewNotnetsServer()

	// Register Server and instantiate with necessary information
	// test_hello_service.RegisterTestServiceServer(svr, svc)
	chaintests.RegisterTestServiceServer(svr, svc)

	// Create Listener
	lis := notnets_grpc.Listen("http://127.0.0.1:8080/hello")
	svr.Serve(lis)
}
