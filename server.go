package notnets_grpc

import (
	"net"

	"google.golang.org/grpc"
)

type NotnetsListener struct {
}

func Listen(addr string) *NotnetsListener {
	var lis NotnetsListener
	// lis.notnets_context = RegisterServer(addr)
	// lis.addr = addr
	return &lis
}

func (lis *NotnetsListener) Accept() (conn net.Conn, err error) {
	return nil, nil
}

func (lis *NotnetsListener) Close() error {
	return nil
}

func (lis *NotnetsListener) Addr() net.Addr {
	return nil
}

type NotnetsServer struct {
	grpc.Server

	//Extra fields
}

func NewNotnetsServer(opts ...grpc.ServerOption) *NotnetsServer {
	var s NotnetsServer
	// s.Server = grpc.NewServer()
	return &s
}

func (s *NotnetsServer) Serve(lis net.Listener) error {

	//Implement Accept Loop and Listening with necessary logic

	return nil
}

func (s *NotnetsServer) Stop() {
	//Stop grpc??? How though
	s.Server.Stop()
	//Stop any notnets specifics
}

func (s *NotnetsServer) handleConnection(conn net.Conn) {
	//Called from Serve
}
