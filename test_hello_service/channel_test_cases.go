package test_hello_service

import (
	"bytes"
	"context"
	"flag"
	"strings"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// RunChannelTestCases runs numerous test cases to exercise the behavior of the
// given channel. The server side of the channel needs to have a *TestServer (in
// this package) registered to provide the implementation of fsgrpc.TestService
// (proto in this package). If the channel does not support full-duplex
// communication, it must provide at least half-duplex support for bidirectional
// streams.
//
// The test cases will be defined as child tests by invoking t.Run on the given
// *testing.T.
func RunChannelTestCases(t *testing.T, ch grpc.ClientConnInterface, supportsFullDuplex bool) {
	cli := NewTestServiceClient(ch)

	t.Run("hello", func(t *testing.T) { testHello(t, cli) })
	t.Run("goodbye", func(t *testing.T) { testGoodbye(t, cli) })

}

const (
	defaultName = "world"
)

func testHello(t *testing.T, cli TestServiceClient) {
	ctx := metadata.NewOutgoingContext(context.Background(), MetadataNew(testOutgoingMd))

	name := flag.String("helloname", defaultName, "Name to greet")

	t.Run("success", func(t *testing.T) {
		req := &HelloRequest{Name: *name}
		rsp, err := cli.SayHello(ctx, req)
		if err != nil {
			t.Fatalf("RPC failed: %v", err)
		}
		if !bytes.Equal([]byte("Hello world"), []byte(rsp.GetMessage())) {
			t.Fatalf("wrong payload returned: expecting %v; got %v", testPayload, rsp.GetMessage())
		}

	})

}

func testGoodbye(t *testing.T, cli TestServiceClient) {
	ctx := metadata.NewOutgoingContext(context.Background(), MetadataNew(testOutgoingMd))

	name := flag.String("goodbyename", defaultName, "Name to greet")

	t.Run("success", func(t *testing.T) {
		req := &GoodbyeRequest{Name: *name}
		rsp, err := cli.SayGoodbye(ctx, req)
		if err != nil {
			t.Fatalf("RPC failed: %v", err)
		}
		if !bytes.Equal([]byte("Goodbye world"), []byte(rsp.GetMessage())) {
			t.Fatalf("wrong payload returned: expecting %v; got %v", testPayload, rsp.GetMessage())
		}

	})

}

func MetadataNew(m map[string][]byte) metadata.MD {
	md := metadata.MD{}
	for k, val := range m {
		key := strings.ToLower(k)
		md[key] = append(md[key], string(val))
	}
	return md
}

var (
	testPayload = []byte{100, 90, 80, 70, 60, 50, 40, 30, 20, 10, 0}

	testOutgoingMd = map[string][]byte{
		"foo":        []byte("bar"),
		"baz":        []byte("bedazzle"),
		"pickle-bin": testPayload,
	}

	// testMdHeaders = map[string][]byte{
	// 	"foo1":        []byte("bar4"),
	// 	"baz2":        []byte("bedazzle5"),
	// 	"pickle3-bin": testPayload,
	// }

	// testMdTrailers = map[string][]byte{
	// 	"4foo4":        []byte("7bar7"),
	// 	"5baz5":        []byte("8bedazzle8"),
	// 	"6pickle6-bin": testPayload,
	// }
)
