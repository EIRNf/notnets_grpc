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

	"github.com/EIRNf/notnets_grpc/internal"

	"time"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding"
	grpcproto "google.golang.org/grpc/encoding/proto"
	"google.golang.org/grpc/metadata"
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

	log.Info().Msgf("Client: New Channel: %v \n ", ch.conn.queues.queues.ClientId)
	log.Info().Msgf("Client: New Channel RequestShmid: %v \n ", ch.conn.queues.queues.RequestShmaddr)
	log.Info().Msgf("Client: New Channel RespomseShmid: %v \n ", ch.conn.queues.queues.ResponseShmaddr)

	ch.conn.isConnected = true

	// ch.request_payload_buffer = make([]byte, MESSAGE_SIZE)
	// ch.request_buffer = bytes.NewReader(ch.request_payload_buffer)

	// ch.variable_read_buffer = bytes.NewBuffer(nil)
	// ch.fixed_read_buffer = make([]byte, MESSAGE_SIZE)

	// ch.request_reader = bufio.NewReader(ch.variable_read_buffer)
	// ch.request_reader = bytes.NewBuffer(nil)

	// ch.response_reader = bufio.NewReader(ch.variable_read_buffer)

	// writer = io.Writer

	// encoder = json.NewEncoder(writer)
	// decoder = json.NewDecoder(reader)

	// ch.dec = sonic.ConfigDefault.NewDecoder(ch.variable_read_buffer)

	return ch, nil
}

type NotnetsChannel struct {
	conn *NotnetsConn
	message_size int32

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

type Channel = grpc.ClientConnInterface
var _ grpc.ClientConnInterface = (*NotnetsChannel)(nil)

const UnaryRpcContentType_V1 = "application/x-protobuf"

func (ch *NotnetsChannel) Invoke(ctx context.Context, methodName string, req, resp interface{}, opts ...grpc.CallOption) error {
	//Tranlsate grpcCallOptions to Notnets call options

	// runtime.LockOSThread()
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

	codec := encoding.GetCodec(grpcproto.Name)
	request_payload_buffer, err := codec.Marshal(req)
	if err != nil {
		return err
	}
	// ch.request_reader.Write(ch.request_payload_buffer)
	request_reader := bytes.NewBuffer(request_payload_buffer)
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

	// var fixed_read_buffer []byte
	// var variable_read_buffer bytes.Buffer

	fixed_response_buffer := make([]byte, ch.message_size)
	variable_respnse_buffer := bytes.NewBuffer(nil)

	//Receive Request
	//iterate and append to dynamically allocated data until all data is read
	for {
		size, err := ch.conn.Read(fixed_response_buffer)
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
			break
		} else { // More data to read, as buffer is full
			continue
		}
	}

	response_reader := bufio.NewReader(variable_respnse_buffer)
	tmp, err := http.ReadResponse(response_reader, r)
	variable_respnse_buffer.Reset()
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

func (ch *NotnetsChannel) NewStream(ctx context.Context, desc *grpc.StreamDesc, methodName string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	return nil, nil
}
