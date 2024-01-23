package notnets_grpc

import (
	"bytes"
	"context"
	"net"
	"net/http"
	"notnets_grpc/internal"
	"strings"
	"sync"
	"time"

	"github.com/fullstorydev/grpchan"
	jsoniter "github.com/json-iterator/go"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/encoding"
	grpcproto "google.golang.org/grpc/encoding/proto"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type NotnetsListener struct {
	mu              sync.Mutex
	notnets_context *ServerContext
	addr            NotnetsAddr
}

func Listen(addr string) *NotnetsListener {
	var lis NotnetsListener
	lis.notnets_context = RegisterServer(addr)
	lis.addr = NotnetsAddr{basic: addr}
	return &lis
}

func (lis *NotnetsListener) Accept() (conn net.Conn, err error) {
	lis.mu.Lock()
	defer lis.mu.Unlock()
	queue := lis.notnets_context.Accept()

	//TODO
	if queue != nil {
		conn = &NotnetsConn{
			ClientSide:  false, //This is the server implementation
			isConnected: true,
			queues:      queue,
			local_addr:  &lis.addr,
			// remote_addr: ,
		}
	} else {
		conn = nil
	}
	return conn, err
}

func (lis *NotnetsListener) Close() error {
	lis.mu.Lock()
	defer lis.mu.Unlock()
	lis.notnets_context.Shutdown()
	return nil
}

func (lis *NotnetsListener) Addr() net.Addr {
	return &lis.addr
}

// ServerOption is an option used when constructing a NewServer.
type ServerOption interface {
	apply(*NotnetsServer)
}

type serverOptFunc func(*NotnetsServer)

func (fn serverOptFunc) apply(s *NotnetsServer) {
	fn(s)
}

// HandlerOption is an option to customize some aspect of the HTTP handler
// behavior, such as rendering gRPC errors to HTTP responses.
//
// HandlerOptions also implement ServerOption.
type HandlerOption func(*handlerOpts)

type handlerOpts struct {
	errFunc func(context.Context, *status.Status, http.ResponseWriter)
}

type NotnetsServer struct {
	// grpc.Server
	handlers         grpchan.HandlerMap
	basePath         string
	opts             handlerOpts
	unaryInterceptor grpc.UnaryServerInterceptor

	// quit    *grpcsync.Event
	// done    *grpcsync.Event
	serveWG sync.WaitGroup

	// ErrorLog *log.Logger

	mu sync.RWMutex

	// Listener accepting connections on a particular IP  and port
	lis net.Listener

	prev_time time.Time

	// Map of queue pairs for boolean of active or inactive connections
	// conns map[int]*QueuePair

	conns sync.Map

	// ShmQueueInfo  *QueueInfo
	// responseQueue *Queue
	// requestQeuue  *Queue

	//Extra fields
}

func NewNotnetsServer(opts ...ServerOption) *NotnetsServer {
	var s NotnetsServer
	// s.Server = grpc.NewServer()
	s.handlers = grpchan.HandlerMap{}
	for _, o := range opts {
		o.apply(&s)
	}
	s.conns = sync.Map{}
	return &s
}

// TODO: DO we need this?
// var _ grpc.ServiceRegistrar = (*NotnetsServer)(nil)

// Register Service, also gets generated by protoc, as Register(SERVICE NAME)Server
func (s *NotnetsServer) RegisterService(desc *grpc.ServiceDesc, svr interface{}) {
	s.handlers.RegisterService(desc, svr)
	// for i := range desc.Methods {
	// 	md := desc.Methods[i]
	// 	h := handleMethod(svr, desc.ServiceName, &md, s.unaryInt, &s.opts)
	// 	s.mux.HandleFunc(path.Join(s.basePath, fmt.Sprintf("%s/%s", desc.ServiceName, md.MethodName)), h)
	// }
}

func (s *NotnetsServer) Serve(lis net.Listener) error {
	//Setup listener
	s.mu.Lock()
	s.lis = lis
	s.mu.Unlock()

	log.Info().Msgf("Serving at address: %v", s.lis.Addr())

	//Begin Accept Loop

	var tempDelay time.Duration
	for {
		rawConn, err := lis.Accept()
		if err != nil {
			log.Error().Msgf("Accept error: %v", err)
			return err
		}

		if rawConn == nil {
			log.Info().Msgf("Null queue_pair, backoff")

			if tempDelay == 0 {
				tempDelay = 5 * time.Second
			} else {
				tempDelay *= 2
			}
			if max := 20 * time.Second; tempDelay > max {
				tempDelay = max
			}
			timer := time.NewTimer(tempDelay)
			select {
			case <-timer.C:
				// case <-s.quit.Done():
				// 	timer.Stop()
				// 	return nil
			}
			continue
		}

		//Check we have not accepted this in the past
		_, ok := s.conns.Load(rawConn.(*NotnetsConn).queues.ClientId)
		if ok {
			log.Info().Msg("Already served queue_pair, backoff")

			if tempDelay == 0 {
				tempDelay = 5 * time.Second
			} else {
				tempDelay *= 2
			}
			if max := 20 * time.Second; tempDelay > max {
				tempDelay = max
			}
			timer := time.NewTimer(tempDelay)
			select {
			case <-timer.C:
				// case <-s.quit.Done():
				// 	timer.Stop()
				// 	return nil
			}
			continue
		} else {
			s.conns.Store(rawConn.(*NotnetsConn).queues.ClientId, rawConn)

			//TODO, improve multithreaded with waitgroupcs
			go func() {
				s.handleConnection(rawConn)
			}()
		}
	}
	return nil
}

func (s *NotnetsServer) Stop() {
	//Stop grpc??? How though
	// s.Stop()
	s.lis.Close()
	//Stop any notnets specifics
}

// Fork a goroutine to handle just-accepted connection

func (s *NotnetsServer) handleConnection(conn net.Conn) {
	//Called from Serve
	log.Info().Msgf("New client connection: %v", conn)

	//Check if server has been shutdown

	//Set service deadlines?

	//Launch dedicated thread to handle
	go func() {
		s.serveRequests(conn)
		// If return from this method, connection has been closed
		// Remove and start servicing, close connection
		// s.closeConnection()
	}()
}

// Actually handles the incoming message flow from the client
// Uses predeclared function
func (s *NotnetsServer) serveRequests(conn net.Conn) {

	// defer close connection
	// var wg sync.WaitGroup

	buf := make([]byte, MESSAGE_SIZE)
	b := bytes.NewBuffer(nil)
	// s.serveWG.Add(1)
	//iterate and append to dynamically allocated data until all data is read
	for {
		size, err := conn.Read(buf)
		if err != nil {
			if err != nil {
				log.Error().Msgf("Read error: %v", err)
			}
		}

		b.Write(buf)
		if size == 0 { //Have full payload
			// log.Info().Msgf("handle request: %s", s.timestamp_dif())
			s.handleMethod(conn, b)
		}
	}
	// Call handle method as we read of queue appropriately.
}

func (s *NotnetsServer) handleMethod(conn net.Conn, b *bytes.Buffer) {
	var json = jsoniter.ConfigCompatibleWithStandardLibrary

	// log.Info().Msgf("Server: Message Received: %v \n ", b.String())

	// Need a method to unmarshall general struct of
	// request, JSON for now
	// log.Info().Msgf("handle method: %s", s.timestamp_dif())

	var messageRequest ShmMessage

	decoder := json.NewDecoder(b)
	err := decoder.Decode(&messageRequest)
	if err != nil {
		log.Panic()
	}

	// log.Info().Msgf("unmarshal: %s", s.timestamp_dif())

	//Request context
	ctx := s.Context(messageRequest.ctx)

	fullName := messageRequest.Method
	strs := strings.SplitN(fullName[1:], "/", 2)
	serviceName := strs[0]
	methodName := strs[1]

	ctx, cancel, err := contextFromHeaders(ctx, messageRequest.Headers)
	if err != nil {
		// writeError(w, http.StatusBadRequest)
		return
	}

	defer cancel()

	//Get Service Descriptor and Handler
	sd, handler := s.handlers.QueryService(serviceName)
	if sd == nil {
		// service name not found
		status.Errorf(codes.Unimplemented, "service %s not implemented", messageRequest.Method)
	}
	// log.Info().Msgf("query service: %s", s.timestamp_dif())

	//Get Method Descriptor
	md := FindUnaryMethod(methodName, sd.Methods)
	if md == nil {
		// method name not found
		status.Errorf(codes.Unimplemented, "method %s/%s not implemented", serviceName, methodName)
	}

	// log.Info().Msgf("find unary: %s", s.timestamp_dif())

	//Get Codec for content type.
	codec := encoding.GetCodec(grpcproto.Name)

	// Function to unmarshal payload using proto
	dec := func(msg interface{}) error {
		val := messageRequest.Payload
		if err := codec.Unmarshal(val, msg); err != nil {
			return status.Error(codes.InvalidArgument, err.Error())
		}
		return nil
	}

	// Implements server transport stream
	sts := internal.UnaryServerTransportStream{Name: methodName}

	//Get resp write back
	resp, err := md.Handler(
		handler,
		grpc.NewContextWithServerTransportStream(ctx, &sts),
		dec,
		s.unaryInterceptor)
	if err != nil {
		status.Errorf(codes.Unknown, "Handler error: %s ", err.Error())
		//TODO: Error code must be sent back to client
	}
	// log.Info().Msgf("handle: %s", s.timestamp_dif())

	var resp_buffer []byte
	resp_buffer, err = codec.Marshal(resp)
	if err != nil {
		status.Errorf(codes.Unknown, "Codec Marshalling error: %s ", err.Error())
	}

	messageResponse := &ShmMessage{
		Method:   methodName,
		ctx:      ctx,
		Headers:  sts.GetHeaders(),
		Trailers: sts.GetTrailers(),
		Payload:  resp_buffer,
	}

	var serializedMessage []byte
	serializedMessage, err = json.Marshal(messageResponse)
	// log.Info().Msgf("marshal: %s", s.timestamp_dif())
	if err != nil {
		status.Errorf(codes.Unknown, "Codec Marshalling error: %s ", err.Error())
	}

	// log.Info().Msgf("Server: Message Sent: %v \n ", serializedMessage)

	//Begin write back
	// message := []byte("{\"method\":\"SayHello\",\"context\":{\"Context\":{\"Context\":{\"Context\":{}}}},\"headers\":null,\"trailers\":null,\"payload\":\"\\n\\u000bHello world\"}")
	conn.Write(serializedMessage)
}

func (s *NotnetsServer) Context(ctx context.Context) context.Context {
	if ctx != nil {
		return ctx
	}
	return context.Background()
}

// contextFromHeaders returns a child of the given context that is populated
// using the given headers. The headers are converted to incoming metadata that
// can be retrieved via metadata.FromIncomingContext. If the headers contain a
// GRPC timeout, that is used to create a timeout for the returned context.
func contextFromHeaders(parent context.Context, md metadata.MD) (context.Context, context.CancelFunc, error) {
	cancel := func() {} // default to no-op

	ctx := metadata.NewIncomingContext(parent, md)

	// deadline propagation
	// timeout := md.Get("GRPC-Timeout")
	// if timeout != "" {
	// 	// See GRPC wire format, "Timeout" component of request: https://grpc.io/docs/guides/wire.html#requests
	// 	suffix := timeout[len(timeout)-1]
	// 	if timeoutVal, err := strconv.ParseInt(timeout[:len(timeout)-1], 10, 64); err == nil {
	// 		var unit time.Duration
	// 		switch suffix {
	// 		case 'H':
	// 			unit = time.Hour
	// 		case 'M':
	// 			unit = time.Minute
	// 		case 'S':
	// 			unit = time.Second
	// 		case 'm':
	// 			unit = time.Millisecond
	// 		case 'u':
	// 			unit = time.Microsecond
	// 		case 'n':
	// 			unit = time.Nanosecond
	// 		}
	// 		if unit != 0 {
	// 			ctx, cancel = context.WithTimeout(ctx, time.Duration(timeoutVal)*unit)
	// 		}
	// 	}
	// }
	return ctx, cancel, nil
}