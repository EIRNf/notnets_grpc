package notnets_grpc

import (
	"testing"
	"time"

	test_hello_service "notnets_grpc/test_hello_service"
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

	cc := NewChannel("localhost", "http://127.0.0.1:8080/hello")

	time.Sleep(10 * time.Second)

	// grpchantesting.RunChannelTestCases(t, &cc, true)
	test_hello_service.RunChannelBenchmarkCases(b, cc, false)

	svr.Stop()
}
