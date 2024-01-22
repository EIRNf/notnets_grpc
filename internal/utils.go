package internal

import (
	"context"
	"reflect"
	"unsafe"

	"google.golang.org/grpc/metadata"
)

type ShmMessage struct {
	Method string          `json:"method"`
	Ctx    context.Context `json:"omitempty"`
	// Headers  map[string][]byte `json:"headers,omitempty"`
	// Trailers map[string][]byte `json:"trailers,omitempty"`
	Headers  metadata.MD `json:"headers"`
	Trailers metadata.MD `json:"trailers"`
	// Payload  string      `json:"payload"`
	Payload []byte
	// Payload interface{}     `protobuf:"bytes,3,opt,name=method,proto3" json:"payload"`
}

// Gets background context
// For outgoing client requests, the context controls cancellation.
//
// For incoming server requests, the context is canceled when the
// client's connection closes, the request is canceled (with HTTP/2),
// or when the ServeHTTP method returns.
func (mes *ShmMessage) Context(ctx context.Context) context.Context {
	if ctx != nil {
		return ctx
	}
	return context.Background()
}

// WithContext returns a shallow copy of r with its context changed
// to ctx. The provided ctx must be non-nil.
func (mes *ShmMessage) WithContext(ctx context.Context) *ShmMessage {
	if ctx == nil {
		panic("nil context")
	}
	mes2 := new(ShmMessage)
	*mes2 = *mes
	mes2.Ctx = ctx
	return mes2
}

// headersFromContext returns HTTP request headers to send to the remote host
// based on the specified context. GRPC clients store outgoing metadata into the
// context, which is translated into headers. Also, a context deadline will be
// propagated to the server via GRPC timeout metadata.
func headersFromContext(ctx context.Context) metadata.MD {
	md, ok := metadata.FromOutgoingContext(ctx)
	if ok {
		//great
	}

	return md
}

func unsafeGetBytes(s string) []byte {
	// fmt.Printf("unsafeGetBytes pointer: %p\n", unsafe.Pointer((*reflect.StringHeader)(unsafe.Pointer(&s)).Data))
	return (*[0x7fff0000]byte)(unsafe.Pointer(
		(*reflect.StringHeader)(unsafe.Pointer(&s)).Data),
	)[:len(s):len(s)]
}

func ByteSlice2String(bs []byte) string {
	// fmt.Printf("ByteSlice2String pointer: %p\n", unsafe.Pointer(&bs))
	return *(*string)(unsafe.Pointer(&bs))
}
