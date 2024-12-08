package notnets_grpc

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
	"sync"
	"unsafe"

	"github.com/EIRNf/notnets_grpc/internal"
	pool "github.com/libp2p/go-buffer-pool"

	"time"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding"
	grpcproto "google.golang.org/grpc/encoding/proto"
	"google.golang.org/grpc/metadata"

	"github.com/hashicorp/yamux"

	"github.com/valyala/fasthttp"
)

func Dial(local_addr, remote_addr string, message_size int32) (*NotnetsChannel, error) {
	//if using dialer always client
	ch := &NotnetsChannel{
		conn: &NotnetsConn{
			ClientSide:  true,
			local_addr:  &NotnetsAddr{basic: local_addr},
			remote_addr: &NotnetsAddr{basic: remote_addr},
		},
	}
	ch.conn.isConnected = false
	ch.message_size = message_size

	// ch.conn.SetDeadline(time.Second * 30)

	var tempDelay time.Duration
	log.Info().Msgf("Client: Opening New Channel %s,%s", local_addr, remote_addr)
	ch.conn.queues = ClientOpen(local_addr, remote_addr, message_size)

	if ch.conn.queues == nil { //if null means server doesn't exist yet
		for {
			log.Info().Msgf("Client: Opening New Channel Failed: Try Again\n")

			//Reattempt wit backoff
			if tempDelay == 0 {
				tempDelay = 3 * time.Second
			} else {
				tempDelay *= 2
			}
			if max := 25 * time.Second; tempDelay > max {
				tempDelay = max
			}
			timer := time.NewTimer(tempDelay)
			<-timer.C
			ch.conn.queues = ClientOpen(local_addr, remote_addr, message_size)
			if ch.conn.queues != nil {
				break
			}
		}
	}
	ch.session_conn, _ = yamux.Client(ch.conn, nil)

	log.Info().Msgf("Client: New Channel: %v \n ", ch.conn.queues.queues.ClientId)
	log.Info().Msgf("Client: New Channel RequestShmid: %v \n ", ch.conn.queues.queues.RequestShmaddr)
	log.Info().Msgf("Client: New Channel RespomseShmid: %v \n ", ch.conn.queues.queues.ResponseShmaddr)

	ch.conn.isConnected = true
	return ch, nil
}

type NotnetsChannel struct {
	conn *NotnetsConn
	message_size int32

	session_conn *yamux.Session

	read_buffer_pool pool.BufferPool
	bufioReaderPool sync.Pool

	// request_payload_buffer []byte

	// fixed_read_buffer    []byte
	// variable_read_buffer *bytes.Buffer

	// writer io.Writer
	// reader io.Reader

	// decoder json.Decoder
	// encoder json.Encoder
	// dec             sonic.Decoder
	// request_reader *bytes.Buffer
	// request_reader  *bufio.Reader
	// response_reader *bufio.Reader

	//ctx
	//connection
	//connectTimeout
	//ConnectTimeWait
}

func (s *NotnetsChannel) newBufioReader(r io.Reader) *bufio.Reader {
	if v := s.bufioReaderPool.Get(); v != nil {
		br := v.(*bufio.Reader)
		br.Reset(r)
		return br
	}
	// Note: if this reader size is ever changed, update
	// TestHandlerBodyClose's assumptions.
	return bufio.NewReader(r)
}

func  (s *NotnetsChannel) putBufioReader(br *bufio.Reader) {
	br.Reset(nil)
	s.bufioReaderPool.Put(br)
}


type Channel = grpc.ClientConnInterface
var _ grpc.ClientConnInterface = (*NotnetsChannel)(nil)

const UnaryRpcContentType_V1 = "application/x-protobuf"

func (ch *NotnetsChannel) Invoke(ctx context.Context, methodName string, req, resp interface{}, opts ...grpc.CallOption) error {
	log.Trace().Msgf("Client:  Request: %s \n ", req)

	//Get Call Options
	copts := internal.GetCallOptions(opts)

	// Get headersFromContext
	reqUrl := "//" + ch.conn.remote_addr.Network()
	reqUrl = path.Join(reqUrl, methodName)

	ctx, err := internal.ApplyPerRPCCreds(ctx, copts, fmt.Sprintf("shm:0%s", reqUrl), true)
	if err != nil {
		return err
	}
	headers := headersFromContext(ctx) //TODO
	codec := encoding.GetCodec(grpcproto.Name)
	request_payload_buffer, err := codec.Marshal(req)
	if err != nil {
		return err
	}


	request := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(request)
	request.DisableRedirectPathNormalizing = true	
	request.Header.SetMethod("POST")
	request.Header.SetRequestURI(reqUrl)
	request.SetBodyRaw(request_payload_buffer)
	request.SetHost(ch.conn.local_addr.String())
	request.Header.Add("Content-Type", UnaryRpcContentType_V1)
	
	for k, vv := range headers{
		for _, v := range vv {
			request.Header.Add(k, v)
		}
	}

	write_buf := bytes.NewBuffer(nil)
	_, err = request.WriteTo(write_buf)
	if err != nil {
		return err
	}
	


	log.Trace().Msgf("Client: Serialized Request: %s \n ", write_buf)

	//START MESSAGING
	// Open a new stream to handle this request:
	stream, err := ch.session_conn.Open()
	if err != nil {
		panic(err)
	}

	stream.Write(write_buf.Bytes())

	fixed_response_buffer := pool.Get(int(ch.message_size))
	variable_respnse_buffer := bytes.NewBuffer(nil)

	//Receive Request
	//iterate and append to dynamically allocated data until all data is read
	for {

		size, err := stream.Read(fixed_response_buffer)
		if err != nil {
			log.Error().Msgf("Client: Read Error: %s", err)
			return err
		}

		//Add control flow to support cancel?
		vsize, err := variable_respnse_buffer.Write(fixed_response_buffer[:size])
		if err != nil {
			log.Error().Msgf("Client: Variable Buffer Write Error: %s", err)
			return err
		}
		if size < int(ch.message_size) { //Have full payload, as we have a read that is smaller than buffer
			log.Trace().Msgf("Client: Received Response Size: %d", vsize)
			log.Trace().Msgf("Client: Received Response: %s", variable_respnse_buffer)
			pool.Put(fixed_response_buffer)
			break
		} else { // More data to read, as buffer is full
			continue
		}
	}

	// defer stream.Close() 

	response_reader := ch.newBufioReader(variable_respnse_buffer)
	defer ch.putBufioReader(response_reader)

	resp_tmp := fasthttp.Response{}
	resp_tmp.Read(response_reader)

	resp_tmp.Header.VisitAll(func(k, v []byte) {
		log.Trace().Msgf("Client: Response Header: %s: %s", k, v)
	})

	// gather headers and trailers

	headers = make(http.Header)
	resp_tmp.Header.VisitAll(func(k, v []byte) {
		sk := b2s(k)
		sv := b2s(v)
		if sk == fasthttp.HeaderCookie {
			sv = strings.Clone(sv)
		}
		headers.Set(sk, sv)
	})

	if len(copts.Headers) > 0 || len(copts.Trailers) > 0 {
		if err := setMetadata(headers, copts); err != nil {
			return err
		}
	}

	b := resp_tmp.Body()
	// resp_tmp
	if err != nil {
		return err
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

	// runtime.UnlockOSThread()
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

// b2s converts byte slice to a string without memory allocation.
// See https://groups.google.com/forum/#!msg/Golang-Nuts/ENgbUzYvCuU/90yGx7GUAgAJ .
func b2s(b []byte) string {
	return unsafe.String(unsafe.SliceData(b), len(b))
}

func (ch *NotnetsChannel) NewStream(ctx context.Context, desc *grpc.StreamDesc, methodName string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	return nil, nil
}
