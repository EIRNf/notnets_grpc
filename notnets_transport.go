package notnets_grpc

import (
	"net"
	"sync"
	"time"
)

type NotnetsAddr struct {
	basic string
	// IP   net.IP
	// Port int
}

func (addr *NotnetsAddr) Network() string {
	return "notnets"
}

func (addr *NotnetsAddr) String() string {
	return addr.basic
}

// Does not support multiple go routines
// It does by having locks but it's not "meant" to
type NotnetsConn struct {
	ClientSide  bool
	isConnected bool

	read_mu        sync.Mutex
	write_mu       sync.Mutex
	queues         *QueueContext
	local_addr     net.Addr
	remote_addr    net.Addr
	deadline       time.Time
	read_deadline  time.Time
	write_deadline time.Time
}

// TODO: Error handling, timeouts
func (c *NotnetsConn) Read(b []byte) (n int, err error) {
	c.read_mu.Lock()
	var bytes_read int
	if c.ClientSide { 
		bytes_read = c.queues.ClientReceiveBuf(b, len(b))
	} else { //Server read
		bytes_read = c.queues.ServerReceiveBuf(b, len(b))
	}
	c.read_mu.Unlock()
	return bytes_read, err
}

// TODO: Error handling, timeouts
func (c *NotnetsConn) Write(b []byte) (n int, err error) {
	c.write_mu.Lock()
	var size int32
	if c.ClientSide {
		size = c.queues.ClientSendRpc(b, len(b))
	} else { //Server read
		size = c.queues.ServerSendRpc(b, len(b))
	}
	c.write_mu.Unlock()
	return int(size), nil
}

// TODO: Error handling, timeouts
func (c *NotnetsConn) Close() error {
	var err error
	ret := ClientClose(c.local_addr.String(), c.remote_addr.String())
	// Error closing
	if ret == -1 {
		// log.Fatalf()
		return err
	}
	return nil
}

// TODO: Error handling, timeouts
func (c *NotnetsConn) LocalAddr() net.Addr {
	return c.local_addr
}

// TODO: Error handling, timeouts
func (c *NotnetsConn) RemoteAddr() net.Addr {
	return c.remote_addr

}

// TODO: Error handling, timeouts
func (c *NotnetsConn) SetDeadline(t time.Time) error {
	c.deadline = t
	return nil
}

// TODO: Error handling, timeouts
func (c *NotnetsConn) SetReadDeadline(t time.Time) error {
	c.read_deadline = t
	return nil
}

// TODO: Error handling, timeouts
func (c *NotnetsConn) SetWriteDeadline(t time.Time) error {
	c.write_deadline = t
	return nil
}
