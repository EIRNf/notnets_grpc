package notnets_grpc

import (
	"log"
	"syscall"
	"testing"
	"time"

	test_hello_service "github.com/EIRNf/notnets_grpc/test_hello_service"
	"github.com/rs/zerolog"
)

func BenchmarkGrpcOverSharedMemory(b *testing.B) {

	// debug.SetGCPercent(-1)
	// runtime.MemProfileRate = 1

	syscall.Syscall(syscall.SYS_PRCTL, syscall.PR_SET_TIMERSLACK, 1, 0)
	// syscall.Syscall(syscall.SYS_SCHED_SETSCHEDULER, syscall.SCHED_FIFO, 0, 0)
	

	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	svc := &test_hello_service.TestServer{}
	svr := NewNotnetsServer(SetMessageSize(MESSAGE_SIZE))

	//Register Server and instantiate with necessary information
	test_hello_service.RegisterTestServiceServer(svr, svc)

	//Create Listener
	lis := Listen("bench")

	go svr.Serve(lis)

	cc, err := Dial("BenchTest", "bench", MESSAGE_SIZE)
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	time.Sleep(5 * time.Second)


	// grpchantesting.RunChannelTestCases(t, &cc, true)
	test_hello_service.RunChannelBenchmarkCases(b, cc, false)

	svr.Stop()
}
