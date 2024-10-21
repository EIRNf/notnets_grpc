package notnets_grpc

import (
	"log"
	"runtime/debug"
	"testing"

	test_hello_service "github.com/EIRNf/notnets_grpc/test_hello_service"
	"github.com/rs/zerolog"
)

func BenchmarkGrpcOverSharedMemory(b *testing.B) {

	debug.SetGCPercent(-1)
	// runtime.MemProfileRate = 1

	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	svc := &test_hello_service.TestServer{}
	svr := NewNotnetsServer()

	//Register Server and instantiate with necessary information
	test_hello_service.RegisterTestServiceServer(svr, svc)

	//Create Listener
	lis := Listen("bench")

	go svr.Serve(lis)

	cc, err := Dial("BenchTest", "bench", MESSAGE_SIZE)
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}

	// grpchantesting.RunChannelTestCases(t, &cc, true)
	test_hello_service.RunChannelBenchmarkCases(b, cc, false)

	svr.Stop()
}
