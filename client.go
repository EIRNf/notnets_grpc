package notnets_grpc

import (
	"context"
	"net"
	"time"

	"google.golang.org/grpc"
)

type NotnetsAddr struct {
}

func (addr *NotnetsAddr) Network() string {
	return ""

}

func (addr *NotnetsAddr) String() string {
	return ""
}

type NotnetsConn struct {
}

func (c *NotnetsConn) Read(b []byte) (n int, err error) {
	return 0, nil

}

func (c *NotnetsConn) Write(b []byte) (n int, err error) {
	return 0, nil

}

func (c *NotnetsConn) Close() error {
	return nil

}

func (c *NotnetsConn) LocalAddr() net.Addr {
	return nil
}

func (c *NotnetsConn) RemoteAddr() net.Addr {
	return nil

}

func (c *NotnetsConn) SetDeadline(t time.Time) error {
	return nil

}

func (c *NotnetsConn) SetReadDeadline(t time.Time) error {
	return nil

}

func (c *NotnetsConn) SetWriteDeadline(t time.Time) error {
	return nil

}

type NotnetsDialer struct {
	//ctx
	//connection
	//connectTimeout
	//ConnectTimeWait
}

func (nc *NotnetsDialer) Dial(network, address string) (net.Conn, error) {
	return nil, nil

}

type NotnetsChannel struct {
}

var _ grpc.ClientConnInterface = (*NotnetsChannel)(nil)

func NewChannel(sourceAddress string, targetAddress string) *NotnetsChannel {
	// ch := new(NotnetsChannel)

	return nil

}

func (ch *NotnetsChannel) Invoke(ctx context.Context, methodName string, req, resp interface{}, opts ...grpc.CallOption) error {
	return nil
}

func (ch *NotnetsChannel) NewStream(ctx context.Context, desc *grpc.StreamDesc, methodName string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	return nil, nil
}
