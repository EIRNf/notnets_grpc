package main

import (
	"flag"
	"log"
	"runtime/debug"
	"testing"

	"github.com/EIRNf/notnets_grpc/test_hello_service"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	addr = flag.String("addr", "localhost:50051", "the address to connect to")
)

func BenchmarkServer(b *testing.B) {
	debug.SetGCPercent(-1)
	// runtime.MemProfileRate = 1

	conn, err := grpc.Dial(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	test_hello_service.RunChannelBenchmarkCases(b, conn, false)
}
