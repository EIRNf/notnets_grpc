package notnets_grpc

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"time"
	"unsafe"

	// "github.com/golang/protobuf/proto"
	// "github.com/protocolbuffers/protobuf-go"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// type ProtoShmMessage struct {
// 	Method string          `json:"method"`
// 	ctx    context.Context `json:"ctx,omitempty"`
// 	// Headers  map[string][]byte `json:"headers,omitempty"`
// 	// Trailers map[string][]byte `json:"trailers,omitempty"`
// 	Headers  metadata.MD `json:",omitempty"`
// 	Trailers metadata.MD `json:",omitempty"`
// 	// Payload  string      `json:"payload"`
// 	// Payload []byte `json:",string"`
// 	Payload proto.Message
// 	// Payload interface{}     `protobuf:"bytes,3,opt,name=method,proto3" json:"payload"`
// }

type ShmMessage struct {
	Method string          `json:"method"`
	ctx    context.Context `json:"ctx,omitempty"`
	// Headers  map[string][]byte `json:"headers,omitempty"`
	// Trailers map[string][]byte `json:"trailers,omitempty"`
	Headers  metadata.MD `json:",omitempty"`
	Trailers metadata.MD `json:",omitempty"`
	// Payload  string      `json:"payload"`
	// Payload []byte `json:",string"`
	Payload []byte
	// Payload interface{}     `protobuf:"bytes,3,opt,name=method,proto3" json:"payload"`
}

// func (s *ShmMessage) MarshalJSON() ([]byte, error) {
// 	var json = jsoniter.ConfigCompatibleWithStandardLibrary

// 	type Alias ShmMessage
// 	return json.Marshal(&struct {
// 		*Alias
// 		Payload string `json:"payload"`
// 	}{
// 		Alias:   (*Alias)(s),
// 		Payload: base64.StdEncoding.EncodeToString(s.Payload),
// 	})

// }

// func (s *ShmMessage) UnmarshalJSON(data []byte) error {
// 	var json = jsoniter.ConfigCompatibleWithStandardLibrary
// 	type Alias ShmMessage
// 	aux := &struct {
// 		*Alias
// 		Payload string `json:"payload"`
// 	}{}

// 	if err := json.Unmarshal(data, &aux); err != nil {
// 		return err
// 	}

// 	payloadBytes, err := base64.StdEncoding.DecodeString(aux.Payload)
// 	if err != nil {
// 		return err
// 	}
// 	s.Method = aux.Method
// 	s.ctx = aux.ctx
// 	s.Headers = aux.Headers
// 	s.Trailers = aux.Trailers
// 	s.Payload = payloadBytes
// 	return nil
// }

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
	mes2.ctx = ctx
	return mes2
}

var reservedHeaders = map[string]struct{}{
	"accept-encoding":   {},
	"connection":        {},
	"content-type":      {},
	"content-length":    {},
	"keep-alive":        {},
	"te":                {},
	"trailer":           {},
	"transfer-encoding": {},
	"upgrade":           {},
}

func toHeaders(md metadata.MD, h http.Header, prefix string) {
	// binary headers must be base-64-encoded
	for k, vs := range md {
		lowerK := strings.ToLower(k)
		if _, ok := reservedHeaders[lowerK]; ok {
			// ignore reserved header keys
			continue
		}
		isBin := strings.HasSuffix(lowerK, "-bin")
		for _, v := range vs {
			if isBin {
				v = base64.URLEncoding.EncodeToString([]byte(v))
			}
			h.Add(prefix+k, v)
		}
	}
}

func headersFromContext(ctx context.Context) http.Header {
	h := http.Header{}
	if md, ok := metadata.FromOutgoingContext(ctx); ok {
		toHeaders(md, h, "")
	}
	if deadline, ok := ctx.Deadline(); ok {
		timeout := time.Until(deadline)
		millis := int64(timeout / time.Millisecond)
		if millis <= 0 {
			millis = 1
		}
		h.Set("GRPC-Timeout", fmt.Sprintf("%dm", millis))
	}
	return h
}

// headersFromContext returns HTTP request headers to send to the remote host
// based on the specified context. GRPC clients store outgoing metadata into the
// context, which is translated into headers. Also, a context deadline will be
// propagated to the server via GRPC timeout metadata.
// func headersFromContext(ctx context.Context) metadata.MD {
// 	md, ok := metadata.FromOutgoingContext(ctx)
// 	if ok {
// 		//great
// 	}

// 	return md
// }

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

// TranslateContextError converts the given error to a gRPC status error if it
// is a context error. If it is context.DeadlineExceeded, it is converted to an
// error with a status code of DeadlineExceeded. If it is context.Canceled, it
// is converted to an error with a status code of Canceled. If it is not a
// context error, it is returned without any conversion.
func TranslateContextError(err error) error {
	switch err {
	case context.DeadlineExceeded:
		return status.Errorf(codes.DeadlineExceeded, err.Error())
	case context.Canceled:
		return status.Errorf(codes.Canceled, err.Error())
	}
	return err
}

// FindUnaryMethod returns the method descriptor for the named method. If the
// method is not found in the given slice of descriptors, nil is returned.
func FindUnaryMethod(methodName string, methods []grpc.MethodDesc) *grpc.MethodDesc {
	for i := range methods {
		if methods[i].MethodName == methodName {
			return &methods[i]
		}
	}
	return nil
}

// FindStreamingMethod returns the stream descriptor for the named method. If
// the method is not found in the given slice of descriptors, nil is returned.
func FindStreamingMethod(methodName string, methods []grpc.StreamDesc) *grpc.StreamDesc {
	for i := range methods {
		if methods[i].StreamName == methodName {
			return &methods[i]
		}
	}
	return nil
}
