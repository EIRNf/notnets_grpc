package main

import (
	"log"
	"testing"
	"time"

	"github.com/EIRNf/notnets_grpc"
	"github.com/EIRNf/notnets_grpc/test_hello_service"
	"github.com/rs/zerolog"
)

func BenchmarkGrpcOverSharedMemory(b *testing.B) {

	// debug.SetGCPercent(-1)
	// runtime.MemProfileRate = 1
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	// f, _ := os.Create("bench.prof")

	// pprof.WriteHeapProfile(f)

	cc, err := notnets_grpc.Dial("localhost", "http://127.0.0.1:8080/hello", notnets_grpc.MESSAGE_SIZE)
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	time.Sleep(20 * time.Second)

	test_hello_service.RunChannelBenchmarkCases(b, cc, false)

	// f.Close()

	// defer pprof.Hea()

}
