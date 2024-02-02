package main

import (
	"github.com/EIRNf/notnets_grpc"
	"github.com/EIRNf/notnets_grpc/test_hello_service"
)

func main() {
	// conn, err := grpc.Dial(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	// if err != nil {
	// 	log.Fatalf("did not connect: %v", err)
	// }
	// defer conn.Close()

	svc1 := &test_hello_service.TestServer{}
	// svc := &channel_tests_service.TestServer{}
	svr1 := notnets_grpc.NewNotnetsServer()

	//Register Server and instantiate with necessary information
	// test_hello_service.RegisterTestServiceServer(svr, svc)
	test_hello_service.RegisterTestServiceServer(svr1, svc1)

	//Create Listener
	lis1 := notnets_grpc.Listen("http://127.0.0.1:8080/goodbye")

	svr1.Serve(lis1)

}
