package notnets_grpc

import (
	"log"
	"testing"
	"time"

	"github.com/EIRNf/notnets_grpc/channel_tests_service"
	"github.com/rs/zerolog"
)

func TestGrpcOverSharedMemory(t *testing.T) {
	zerolog.SetGlobalLevel(zerolog.TraceLevel)


	svc := &channel_tests_service.TestServer{}
	// svc := &channel_tests_service.TestServer{}
	svr := NewNotnetsServer()

	//Register Server and instantiate with necessary information
	// test_hello_service.RegisterTestServiceServer(svr, svc)
	channel_tests_service.RegisterTestServiceServer(svr, svc)

	//Create Listener
	lis := Listen("http://127.0.0.1:8080/hello")

	go svr.Serve(lis)

	cc, err := Dial("FunctionalTest", "http://127.0.0.1:8080/hello", MESSAGE_SIZE)
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	time.Sleep(2 * time.Second)

	// channel_tests_service.RunChannelTestCases(t, cc, true)
	channel_tests_service.RunChannelTestCases(t, cc, true)

	svr.Stop()
}
