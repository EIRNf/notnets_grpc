package notnets_grpc

import (
	"testing"

	test_hello_service "notnets_grpc/test_hello_service"
)

type TestServer struct {
	test_hello_service.UnimplementedTestServiceServer
}

func TestGrpcOverSharedMemory(t *testing.T) {
	// conn, err := grpc.Dial(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	// if err != nil {
	// 	log.Fatalf("did not connect: %v", err)
	// }
	// defer conn.Close()

	// svr := &grpchantesting.TestServer{}
	svc := &TestServer{}
	svr := NewNotnetsServer()

	//Register Server and instantiate with necessary information

	test_hello_service.RegisterTestServiceServer(svr, svc)
	//Create Listener
	lis := Listen("http://127.0.0.1:8080/hello")

	go svr.Serve(lis)
	defer svr.Stop()

	//Replace with Dialing logicZ

	cc := NewChannel("localhost", "http://127.0.0.1:8080/hello")

	test_hello_service.RunChannelTestCases(t, cc, true)

}
