package notnets_grpc

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"path"
	"strings"

	"github.com/EIRNf/notnets_grpc/internal"

	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding"
	encoding_proto "google.golang.org/grpc/encoding/proto"
	"google.golang.org/grpc/metadata"
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
	log.Info().Msgf("Client: Opening New Channel %s,%s\n", local_addr, remote_addr)
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

	ch.request_payload_buffer = make([]byte, MESSAGE_SIZE)
	// ch.request_buffer = bytes.NewReader(ch.request_payload_buffer)

	ch.variable_read_buffer = bytes.NewBuffer(nil)
	ch.fixed_read_buffer = make([]byte, MESSAGE_SIZE)

	// ch.request_reader = bufio.NewReader(ch.variable_read_buffer)
	ch.request_reader = bytes.NewBuffer(nil)

	ch.response_reader = bufio.NewReader(ch.variable_read_buffer)

	// writer = io.Writer

	// encoder = json.NewEncoder(writer)
	// decoder = json.NewDecoder(reader)

	// ch.dec = sonic.ConfigDefault.NewDecoder(ch.variable_read_buffer)

	return ch, nil
}

type NotnetsChannel struct {
	conn *NotnetsConn

	request_payload_buffer []byte

	fixed_read_buffer    []byte
	variable_read_buffer *bytes.Buffer

	// writer io.Writer
	// reader io.Reader

	// decoder json.Decoder
	// encoder json.Encoder
	// dec             sonic.Decoder
	request_reader *bytes.Buffer
	// request_reader  *bufio.Reader
	response_reader *bufio.Reader

	//ctx
	//connection
	//connectTimeout
	//ConnectTimeWait
}

var _ grpc.ClientConnInterface = (*NotnetsChannel)(nil)

const UnaryRpcContentType_V1 = "application/x-protobuf"

func (ch *NotnetsChannel) Invoke(ctx context.Context, methodName string, req, resp interface{}, opts ...grpc.CallOption) error {
	// var json = jsoniter.ConfigCompatibleWithStandardLibrary

	log.Trace().Msgf("Client:  Request: %s \n ", req)

	//Get Call Options
	copts := internal.GetCallOptions(opts)

	// Get headersFromContext
	reqUrl := "//" + ch.conn.remote_addr.Network()
	reqUrl = path.Join(reqUrl, methodName)
	// reqUrlStr := reqUrl.String()

	ctx, err := internal.ApplyPerRPCCreds(ctx, copts, fmt.Sprintf("shm:0%s", reqUrl), true)
	if err != nil {
		return err
	}
	h := headersFromContext(ctx)
	h.Set("Content-Type", UnaryRpcContentType_V1)

	codec := encoding.GetCodec(encoding_proto.Name)
	ch.request_payload_buffer, err = codec.Marshal(req)
	if err != nil {
		return err
	}
	// ch.request_reader.Write(ch.request_payload_buffer)
	request_reader := bytes.NewBuffer(ch.request_payload_buffer)
	r, err := http.NewRequest("POST", reqUrl, request_reader)
	if err != nil {
		return err
	}
	r.Header = h

	var writeBuffer = &bytes.Buffer{}
	r.WithContext(ctx).Write(writeBuffer)

	log.Trace().Msgf("Client: Serialized Request: %s \n ", writeBuffer)

	//START MESSAGING
	// pass into shared mem queue
	ch.conn.Write(writeBuffer.Bytes())

	//Receive Request
	//iterate and append to dynamically allocated data until all data is read
	var size int
	//Most time is spend reading, wiating on Server to finish
	for {
		size, err = ch.conn.Read(ch.fixed_read_buffer)
		if err != nil {
			return err
		}

		//Add control flow to support cancel?

		ch.variable_read_buffer.Write(ch.fixed_read_buffer)
		if size == 0 { //Have full payload
			break
		}
	}

	log.Trace().Msgf("Client: Serialized Response: %s \n ", ch.variable_read_buffer)

	ch.response_reader.Reset(ch.variable_read_buffer)
	tmp, err := http.ReadResponse(ch.response_reader, r)
	if err != nil {
		return err
	}

	//Create goroutine to handle cancels?

	b, err := io.ReadAll(tmp.Body)
	tmp.Body.Close()
	if err != nil {
		return err
	}

	// gather headers and trailers
	if len(copts.Headers) > 0 || len(copts.Trailers) > 0 {
		if err := setMetadata(tmp.Header, copts); err != nil {
			return err
		}
	}

	// copts.SetHeaders(t)
	// copts.SetTrailers(messageResponse.Trailers)

	// // gather headers and trailers
	// if len(copts.Headers) > 0 || len(copts.Trailers) > 0 {
	// 	if err := setMetadata(reply.Header, copts); err != nil {
	// 		return err
	// 	}
	// }

	// if stat := statFromResponse(reply); stat.Code() != codes.OK {
	// 	return stat.Err()
	// }

	// select {
	// case <-ctx.Done():
	// 	return statusFromContextError(ctx.Err())
	// case <-respCh:
	// }
	// if err != nil {
	// 	return err
	// }

	return codec.Unmarshal(b, resp)

}

// asMetadata converts the given HTTP headers into GRPC metadata.
func asMetadata(header http.Header) (metadata.MD, error) {
	// metadata has same shape as http.Header,
	md := metadata.MD{}
	for k, vs := range header {
		k = strings.ToLower(k)
		for _, v := range vs {
			if strings.HasSuffix(k, "-bin") {
				vv, err := base64.URLEncoding.DecodeString(v)
				if err != nil {
					return nil, err
				}
				v = string(vv)
			}
			md[k] = append(md[k], v)
		}
	}
	return md, nil
}

func setMetadata(h http.Header, copts *internal.CallOptions) error {
	hdr, err := asMetadata(h)
	if err != nil {
		return err
	}
	tlr := metadata.MD{}

	const trailerPrefix = "x-grpc-trailer-"

	for k, v := range hdr {
		if strings.HasPrefix(strings.ToLower(k), trailerPrefix) {
			trailerName := k[len(trailerPrefix):]
			if trailerName != "" {
				tlr[trailerName] = v
				delete(hdr, k)
			}
		}
	}

	copts.SetHeaders(hdr)
	copts.SetTrailers(tlr)
	return nil
}

func (ch *NotnetsChannel) NewStream(ctx context.Context, desc *grpc.StreamDesc, methodName string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	return nil, nil
}
