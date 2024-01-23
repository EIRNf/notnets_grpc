package notnets_grpc

import (
	"log"
	"testing"
	"time"

	"github.com/EIRNf/notnets_grpc/test_hello_service"
)

func BenchmarkGrpcOverSharedMemory(b *testing.B) {

	// debug.SetGCPercent(-1)
	// runtime.MemProfileRate = 1

	// svr := &grpchantesting.TestServer{}
	svc := &test_hello_service.TestServer{}
	svr := NewNotnetsServer()

	//Register Server and instantiate with necessary information
	test_hello_service.RegisterTestServiceServer(svr, svc)

	//Create Listener
	lis := Listen("http://127.0.0.1:8080/hello")

	go svr.Serve(lis)

	cc, err := Dial("localhost", "http://127.0.0.1:8080/hello")
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	time.Sleep(10 * time.Second)

	// grpchantesting.RunChannelTestCases(t, &cc, true)
	test_hello_service.RunChannelBenchmarkCases(b, cc, false)

	svr.Stop()
}
