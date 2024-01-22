package notnets_grpc

import (
	"log"
	"testing"

	"notnets_grpc/test_hello_service"
)

func TestGrpcOverSharedMemory(t *testing.T) {
	// conn, err := grpc.Dial(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	// if err != nil {
	// 	log.Fatalf("did not connect: %v", err)
	// }
	// defer conn.Close()

	svc := &test_hello_service.TestServer{}
	// svc := &channel_tests_service.TestServer{}
	svr := NewNotnetsServer()

	//Register Server and instantiate with necessary information
	// test_hello_service.RegisterTestServiceServer(svr, svc)
	test_hello_service.RegisterTestServiceServer(svr, svc)

	//Create Listener
	lis := Listen("http://127.0.0.1:8080/hello")

	go svr.Serve(lis)

	cc, err := Dial("localhost", "http://127.0.0.1:8080/hello")
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	// channel_tests_service.RunChannelTestCases(t, cc, true)
	test_hello_service.RunChannelTestCases(t, cc, true)

	svr.Stop()
}
