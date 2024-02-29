package notnets_grpc

import (
	"testing"
)

func TestGrpcChainOverSharedMemory(t *testing.T) {
	// conn, err := grpc.Dial(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	// if err != nil {
	// 	log.Fatalf("did not connect: %v", err)
	// }
	// defer conn.Close()

	// svc1 := &test_hello_service.TestServer{}
	// // svc := &channel_tests_service.TestServer{}
	// svr1 := NewNotnetsServer()

	// //Register Server and instantiate with necessary information
	// // test_hello_service.RegisterTestServiceServer(svr, svc)
	// test_hello_service.RegisterTestServiceServer(svr1, svc1)

	// //Create Listener
	// lis1 := Listen("http://127.0.0.1:8080/goodbye")

	// go svr1.Serve(lis1)
	// time.Sleep(5 * time.Second)

	// cc1, err := Dial("localhost", "http://127.0.0.1:8080/goodbye")
	// if err != nil {
	// 	log.Fatalf("did not connect: %v", err)
	// }

	// svc := &chaintests.TestServer{
	// 	HelloClient: test_hello_service.NewTestServiceClient(cc1),
	// }
	// // svc := &channel_tests_service.TestServer{}
	// svr := NewNotnetsServer()

	// //Register Server and instantiate with necessary information
	// // test_hello_service.RegisterTestServiceServer(svr, svc)
	// chaintests.RegisterTestServiceServer(svr, svc)

	// //Create Listener
	// lis := Listen("http://127.0.0.1:8080/hello")
	// go svr.Serve(lis)
	// time.Sleep(5 * time.Second)

	// cc, err := Dial("TestChain", "http://127.0.0.1:8080/hello")
	// if err != nil {
	// 	log.Fatalf("did not connect: %v", err)
	// }
	// // channel_tests_service.RunChannelTestCases(t, cc, true)
	// chaintests.RunChannelTestCases(t, cc, true)

	// svr.Stop()
}
