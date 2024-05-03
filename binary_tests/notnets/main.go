package main

import (
	"github.com/EIRNf/notnets_grpc"
	"github.com/EIRNf/notnets_grpc/test_hello_service"
	"github.com/rs/zerolog"
)

func main() {

	// debug.SetGCPercent(-1)
	// debug.SetMaxStack(2e+9)
	// debug.SetMemoryLimit(-1)

	// runtime.MemProfileRate = 1
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	// f, _ := os.Create("serv_cpu.prof")
	// pprof.WriteHeapProfile(f)
	// pprof.StartCPUProfile(f)

	svc := &test_hello_service.TestServer{}
	svr := notnets_grpc.NewNotnetsServer()

	//Register Server and instantiate with necessary information
	test_hello_service.RegisterTestServiceServer(svr, svc)

	//Create Listener
	lis := notnets_grpc.Listen("http://127.0.0.1:8080/hello")

	svr.Serve(lis)

	// time.Sleep(40 * time.Second)

	// pprof.StopCPUProfile()
	// defer f.Close()
	defer svr.Stop()

}
