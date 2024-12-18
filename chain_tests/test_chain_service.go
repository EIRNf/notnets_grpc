package chaintests

// protoc --go_out=./ --go_opt=paths=source_relative --go-grpc_out=./ --go-grpc_opt=paths=source_relative test_hello.proto

import (
	"context"
)

// TestServer has default responses to the various kinds of methods.
type TestServer struct {
	HelloClient TestServiceClient

	UnimplementedTestServiceServer
}

func (s *TestServer) SayHello(ctx context.Context, in *HelloRequest) (*HelloReply, error) {

	req := &GoodbyeRequest{
		Name: in.Name,
	}
	resp, err := s.HelloClient.SayGoodbye(ctx, req)
	return &HelloReply{Message: "Hello " + resp.GetMessage()}, err
}

func (s *TestServer) SayGoodbye(ctx context.Context, in *GoodbyeRequest) (*GoodbyeReply, error) {
	return &GoodbyeReply{Message: "Goodbye " + in.GetName()}, nil
}
