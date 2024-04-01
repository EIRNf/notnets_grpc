package notnets_grpc

import (
	"log"
	"runtime/debug"
	"testing"
	"time"

	"github.com/EIRNf/notnets_grpc/test_hello_service"
	"github.com/fullstorydev/grpchan"
	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/opentracing/opentracing-go"
	"github.com/rs/zerolog"
	"github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/config"
)

func BenchmarkGrpcOverSharedMemoryInterceptor(b *testing.B) {

	debug.SetGCPercent(-1)
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	cfg := &config.Configuration{
		ServiceName: "test", // Call trace of the target service. Enter the service name.
		Sampler: &config.SamplerConfig{ // Sampling policy configuration. See section 4.1.1 for details.
			Type:  "const",
			Param: 1,
		},
		Reporter: &config.ReporterConfig{ // Configure how the client reports trace information. All fields are optional.
			LogSpans:           true,
			LocalAgentHostPort: "localhost:4040",
		},
		// Token configuration
		Tags: []opentracing.Tag{ // Set the tag, where information such as token can be stored.
			opentracing.Tag{Key: "token", Value: "token"}, // Set the token
		},
	}

	tracer, closer, err := cfg.NewTracer(config.Logger(jaeger.StdLogger)) // Get the tracer based on the configuration
	if err != nil {
		log.Fatalf("tracer: %v", err)
	}

	// svr := &grpchantesting.TestServer{}
	svc := &test_hello_service.TestServer{}
	//Add interceptor option
	// svr := NewNotnetsServer(grpc.UnaryInterceptor(otgrpc.OpenTracingServerInterceptor(tracer)))

	// svr := NewNotnetsServer(UnaryInterceptor(otelgrpc.NewServerHandler()))

	svr := NewNotnetsServer(UnaryInterceptor(otgrpc.OpenTracingServerInterceptor(tracer)))
	// svr := grpc.NewServer(grpc.UnaryInterceptor(otgrpc.OpenTracingServerInterceptor(tracer)))

	//Register Server and instantiate with necessary information
	test_hello_service.RegisterTestServiceServer(svr, svc)

	//Create Listener
	lis := Listen("http://127.0.0.1:8080/hello")

	go svr.Serve(lis)

	cc, err := Dial("localhost", "http://127.0.0.1:8080/hello", MESSAGE_SIZE)
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	intercepted := grpchan.InterceptClientConn(cc, otgrpc.OpenTracingClientInterceptor(tracer), nil)
	time.Sleep(10 * time.Second)

	// grpchantesting.RunChannelTestCases(t, &cc, true)
	test_hello_service.RunChannelBenchmarkCases(b, intercepted, false)

	closer.Close()
	svr.Stop()
}
