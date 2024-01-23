package notnets_grpc

import (
	"bytes"
	"context"
	"net"

	"github.com/EIRNf/notnets_grpc/internal"

	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding"

	jsoniter "github.com/json-iterator/go"
	grpcproto "google.golang.org/grpc/encoding/proto"
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
	return "notnets:" + addr.basic
}

// Does not support multiple go routines
// It does by having locks but it's not "meant" to
type NotnetsConn struct {
	ClientSide  bool
	isConnected bool

	mu             sync.RWMutex
	queues         *QueuePair
	local_addr     net.Addr
	remote_addr    net.Addr
	deadline       time.Time
	read_deadline  time.Time
	write_deadline time.Time
}

// TODO: Error handling, timeouts
func (c *NotnetsConn) Read(b []byte) (n int, err error) {
	c.mu.RLock()
	var leftover_bytes int
	if c.ClientSide {
		leftover_bytes = c.queues.ClientReceiveBuf(b, len(b))
		// if leftover_bytes == -1 {
		// }
	} else { //Server read
		leftover_bytes = c.queues.ServerReceiveBuf(b, len(b))
	}
	c.mu.RUnlock()
	return leftover_bytes, nil
}

// TODO: Error handling, timeouts
func (c *NotnetsConn) Write(b []byte) (n int, err error) {
	c.mu.Lock()
	var size int32
	if c.ClientSide {
		size = c.queues.ClientSendRpc(b, len(b))
	} else { //Server read
		size = c.queues.ServerSendRpc(b, len(b))
	}
	c.mu.Unlock()
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

func Dial(local_addr, remote_addr string) (*NotnetsChannel, error) {
	//if using dialer always client
	ch := &NotnetsChannel{
		conn: &NotnetsConn{
			ClientSide:  true,
			local_addr:  &NotnetsAddr{basic: local_addr},
			remote_addr: &NotnetsAddr{basic: remote_addr},
		},
	}
	ch.conn.isConnected = false

	// ch.conn.SetDeadline(time.Second * 30)

	var tempDelay time.Duration
	log.Info().Msgf("Client: Opening New Channel \n")
	ch.conn.queues = ClientOpen(local_addr, remote_addr, MESSAGE_SIZE)

	if ch.conn.queues == nil { //if null means server doesn't exist yet
		for {
			log.Info().Msgf("Client: Opening New Channel Failed: Try Again\n")

			//Reattempt wit backoff
			if tempDelay == 0 {
				tempDelay = 10 * time.Second
			} else {
				tempDelay *= 2
			}
			if max := 40 * time.Second; tempDelay > max {
				tempDelay = max
			}
			timer := time.NewTimer(tempDelay)
			<-timer.C
			ch.conn.queues = ClientOpen(local_addr, remote_addr, MESSAGE_SIZE)
			if ch.conn.queues != nil {
				break
			}
		}

	}

	log.Info().Msgf("Client: New Channel: %v \n ", ch.conn.queues.ClientId)
	log.Info().Msgf("Client: New Channel RequestShmid: %v \n ", ch.conn.queues.RequestShmaddr)
	log.Info().Msgf("Client: New Channel RespomseShmid: %v \n ", ch.conn.queues.ResponseShmaddr)

	ch.conn.isConnected = true

	ch.variable_read_buffer = bytes.NewBuffer(nil)
	ch.fixed_read_buffer = make([]byte, MESSAGE_SIZE)

	return ch, nil
}

type NotnetsChannel struct {
	conn *NotnetsConn

	fixed_read_buffer    []byte
	variable_read_buffer *bytes.Buffer

	//ctx
	//connection
	//connectTimeout
	//ConnectTimeWait
}

var _ grpc.ClientConnInterface = (*NotnetsChannel)(nil)

func (ch *NotnetsChannel) Invoke(ctx context.Context, methodName string, req, resp interface{}, opts ...grpc.CallOption) error {
	var json = jsoniter.ConfigCompatibleWithStandardLibrary

	//Get Call Options
	copts := internal.GetCallOptions(opts)

	// Get headersFromContext
	// reqUrl := ch.targetAddress
	// reqUrl.Path = path.Join(reqUrl.Path, methodName)
	// reqUrlStr := reqUrl.String()

	// ctx, err := internal.ApplyPerRPCCreds(ctx, copts, fmt.Sprintf("shm:0%s", reqUrlStr), true)
	// if err != nil {
	// 	return err
	// }

	codec := encoding.GetCodec(grpcproto.Name)

	serializedPayload, err := codec.Marshal(req)
	if err != nil {
		return err
	}

	messageRequest := &ShmMessage{
		Method:  methodName,
		ctx:     ctx,
		Headers: headersFromContext(ctx),
		// Trailers: trailersFrom,
		Payload: serializedPayload,
	}

	// we have the meta request
	// Marshall to build rest of system
	var serializedMessage []byte
	serializedMessage, err = json.Marshal(messageRequest)
	if err != nil {
		return err
	}

	//START MESSAGING
	// pass into shared mem queue
	ch.conn.Write(serializedMessage)

	//Receive Request
	//iterate and append to dynamically allocated data until all data is read
	var size int
	for {
		size, err = ch.conn.Read(ch.fixed_read_buffer)
		if err != nil {
			return err
		}

		ch.variable_read_buffer.Write(ch.fixed_read_buffer)
		if size == 0 { //Have full payload
			break
		}
	}

	var messageResponse ShmMessage
	dec := json.NewDecoder(ch.variable_read_buffer)
	err = dec.Decode(&messageResponse)

	if err != nil {
		return err // TODO BAD
	}

	payload := messageResponse.Payload

	copts.SetHeaders(messageResponse.Headers)
	copts.SetTrailers(messageResponse.Trailers)

	//Update total number of back and forth messages
	// fmt.Printf("finished message num %d:", ch.Metadata.NumMessages)

	// we fire up a goroutine to read the response so that we can properly
	// respect any context deadline (e.g. don't want to be blocked, reading
	// from socket, long past requested timeout).
	// respCh := make(chan struct{})
	// select {
	// case <-ctx.Done():
	// 	return statusFromContextError(ctx.Err())
	// case <-respCh:
	// }
	// if err != nil {
	// 	return err
	// }

	ret_err := codec.Unmarshal(payload, resp)
	return ret_err
}

func (ch *NotnetsChannel) NewStream(ctx context.Context, desc *grpc.StreamDesc, methodName string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	return nil, nil
}
