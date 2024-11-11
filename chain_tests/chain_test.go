package chaintests_test

import (
	"log"
	"testing"
	"time"

	"github.com/EIRNf/notnets_grpc"
	chaintests "github.com/EIRNf/notnets_grpc/chain_tests"
	test_hello_service "github.com/EIRNf/notnets_grpc/test_hello_service"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestGrpcChainOverSharedMemory(t *testing.T) {


	conn, err := grpc.Dial("http://127.0.0.1:8080/hello", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	svc1 := &test_hello_service.TestServer{}
	// svc := &channel_tests_service.TestServer{}
	svr1 := notnets_grpc.NewNotnetsServer()

	//Register Server and instantiate with necessary information
	// test_hello_service.RegisterTestServiceServer(svr, svc)
	test_hello_service.RegisterTestServiceServer(svr1, svc1)

	//Create Listener
	lis1 := notnets_grpc.Listen("http://127.0.0.1:8080/goodbye")

	go svr1.Serve(lis1)
	time.Sleep(5 * time.Second)

	cc1, err := notnets_grpc.Dial("localhost", "http://127.0.0.1:8080/goodbye", notnets_grpc.MESSAGE_SIZE)
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}

	svc := &chaintests.TestServer{
		HelloClient: chaintests.NewTestServiceClient(cc1),
	}
	// svc := &channel_tests_service.TestServer{}
	svr := notnets_grpc.NewNotnetsServer()

	//Register Server and instantiate with necessary information
	// test_hello_service.RegisterTestServiceServer(svr, svc)
	chaintests.RegisterTestServiceServer(svr, svc)

	//Create Listener
	lis := notnets_grpc.Listen("http://127.0.0.1:8080/hello")
	go svr.Serve(lis)
	time.Sleep(5 * time.Second)

	cc, err := notnets_grpc.Dial("TestChain", "http://127.0.0.1:8080/hello", notnets_grpc.MESSAGE_SIZE)
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	// channel_tests_service.RunChannelTestCases(t, cc, true)
	chaintests.RunChannelTestCases(t, cc, true)

	svr.Stop()
}
