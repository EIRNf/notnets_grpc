package main

import (
	"github.com/EIRNf/notnets_grpc"
	"github.com/EIRNf/notnets_grpc/test_hello_service"
)

func main() {

	// f, _ := os.Create("bench.prof")
	// pprof.WriteHeapProfile(f)

	svc := &test_hello_service.TestServer{}
	svr := notnets_grpc.NewNotnetsServer()

	//Register Server and instantiate with necessary information
	test_hello_service.RegisterTestServiceServer(svr, svc)

	//Create Listener
	lis := notnets_grpc.Listen("http://127.0.0.1:8080/hello")

	svr.Serve(lis)
	// defer f.Close()
	defer svr.Stop()

}
