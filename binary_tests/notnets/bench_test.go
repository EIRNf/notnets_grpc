package main

import (
	"log"
	"testing"
	"time"

	"github.com/EIRNf/notnets_grpc"
	"github.com/EIRNf/notnets_grpc/test_hello_service"
)

func BenchmarkGrpcOverSharedMemory(b *testing.B) {
	// f, _ := os.Create("bench.prof")

	// pprof.StartCPUProfile(f)

	cc, err := notnets_grpc.Dial("localhost", "http://127.0.0.1:8080/hello")
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	time.Sleep(10 * time.Second)

	test_hello_service.RunChannelBenchmarkCases(b, cc, false)

	// defer pprof.StopCPUProfile()

}