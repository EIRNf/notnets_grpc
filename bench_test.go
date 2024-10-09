package notnets_grpc

import (
	"log"
	"runtime/debug"
	"testing"
	"time"

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
	lis := Listen("http://127.0.0.1:8080/hello")

	go svr.Serve(lis)

	cc, err := Dial("BenchTest", "http://127.0.0.1:8080/hello", MESSAGE_SIZE)
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	time.Sleep(3 * time.Second)

	// grpchantesting.RunChannelTestCases(t, &cc, true)
	test_hello_service.RunChannelBenchmarkCases(b, cc, false)

	svr.Stop()
}
