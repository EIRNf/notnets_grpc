package notnets_grpc

import (
	"reflect"
	"testing"
	"time"

	"github.com/libp2p/go-yamux"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
)

func TestYamuxOverSharedMemory(t *testing.T) {

	go server()
	client(t)

}
func client(t *testing.T) {
	// Get a TCP connection
	cc, err := Dial("FunctionalTest", "http://127.0.0.1:8080/hello", MESSAGE_SIZE)
	if err != nil {
		panic(err)
	}

	// config := &yamux.Config{
	// 	AcceptBacklog:          256,
	// 	EnableKeepAlive:        false,
	// 	KeepAliveInterval:      30 * time.Second,
	// 	ConnectionWriteTimeout: 500 * time.Second,
	// 	MaxStreamWindowSize:    256 * 1024,
	// 	StreamCloseTimeout:     5 * time.Minute,
	// 	StreamOpenTimeout:      75 * time.Second,
	// 	LogOutput:              os.Stderr,
	// }
	// Setup client side of yamux
	session, err := yamux.Client(cc.conn, nil)
	if err != nil {
		panic(err)
	}
	time.Sleep(10 * time.Second)

	// Open a new stream
	stream, err := session.Open()
	if err != nil {
		panic(err)
	}

	// Stream implements net.Conn
	stream.Write([]byte("ping"))
	buf := make([]byte, 4)
	stream.Read(buf)

	assert.True(t, reflect.DeepEqual(buf, "ping"))
}

func server() {
	// Accept a TCP connection
	// conn, err := listener.Accept()
	// if err != nil {
	//     panic(err)
	// }

	lis := Listen("http://127.0.0.1:8080/hello")

	var session *yamux.Session
	var tempDelay time.Duration
	for {
		rawConn, err := lis.Accept()
		if err != nil {
			log.Error().Msgf("Accept error: %s", err)
			panic(err)
		}
		if rawConn == nil {
			log.Trace().Msgf("Null queue_pair, backoff")

			if tempDelay == 0 {
				tempDelay = 3 * time.Second
			} else {
				tempDelay *= 2
			}
			if max := 25 * time.Second; tempDelay > max {
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
		// config := &yamux.Config{
		// 	AcceptBacklog:          256,
		// 	EnableKeepAlive:        false,
		// 	KeepAliveInterval:      30 * time.Second,
		// 	ConnectionWriteTimeout: 500 * time.Second,
		// 	MaxStreamWindowSize:    256 * 1024,
		// 	StreamCloseTimeout:     5 * time.Minute,
		// 	StreamOpenTimeout:      75 * time.Second,
		// 	LogOutput:              os.Stderr,
		// }
		// Setup server side of yamux
		session, err = yamux.Server(rawConn, nil)
		if err != nil {
			panic(err)
		}
		break
	}

	// Accept a stream
	stream, err := session.Accept()
	if err != nil {
		panic(err)
	}

	// Listen for a message
	buf := make([]byte, 4)
	stream.Read(buf)
	stream.Write(buf)

}
