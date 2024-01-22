package notnets_grpc

// #cgo CFLAGS: -O2 -Wall -pthread
// #include <stdlib.h>
// #include <stdio.h>
// #include <errno.h>
// #include "notnets_shm/src/rpc.h"
import "C"
import (
	"unsafe"

	"github.com/rs/zerolog/log"
	"modernc.org/libc/pthread"
)

// type queue_pair struct {
// 	id            uint64
// 	request_addr  unsafe.Pointer
// 	response_addr unsafe.Pointer
// }

// func open(source_addr string, destination_addr string) queue_pair {
// 	csource_addr := C.CString(source_addr)
// 	defer free(unsafe.Pointer(csource_addr))
// 	cdestination_addrt := C.CString(destination_addr)
// 	defer free(unsafe.Pointer(cdestination_addrt))

// 	cqp := C.open(csource_addr, cdestination_addrt)
// 	var goqp = queue_pair{
// 		uint64(cqp.id),
// 		cqp.shm_request_shmaddr,
// 		cqp.shm_response_shmaddr,
// 	}
// 	return goqp
// }

// func send_rpc(conn queue_pair, buf []byte) int {

// 	//get size from byte slice
// 	cqp := _Ctype_struct_queue_pair{
// 		C.ulong(conn.id),
// 		conn.request_addr,
// 		conn.respnse_addr,
// 	}

// 	cbuf := C.CBytes(buf)
// 	defer free(cbuf)

// 	return C.send_rpc(cqp, cbuf, C.ulong(len(buf)))
// }

const MESSAGE_SIZE = 512

type QueuePair struct {
	ClientId        int
	RequestShmaddr  unsafe.Pointer
	ResponseShmaddr unsafe.Pointer
}

// server_context
type ServerContext struct {
	CoordShmaddr     uintptr
	ManagePoolThread pthread.Pthread_t
	ManagePoolState  int32
	ManagePoolMutex  pthread.Pthread_mutex_t
}

func ClientOpen(sourceAddr string, destinationAddr string, messageSize int32) (ret *QueuePair) {
	_sourceAddr := C.CString(sourceAddr)
	defer C.free(unsafe.Pointer(_sourceAddr))
	_destinationAddr := C.CString(destinationAddr)
	defer C.free(unsafe.Pointer(_destinationAddr))
	_messageSize := C.int(messageSize)
	_ret := C.client_open(_sourceAddr, _destinationAddr, _messageSize)
	log.Info().Msgf("Client: open response : %v \n ", _ret)
	ret = (*QueuePair)(unsafe.Pointer(_ret))
	log.Info().Msgf("Client: open response ret : %v \n ", ret)

	return
}

// client_send_rpc
func (conn *QueuePair) ClientSendRpc(buf []byte, size int) (ret int32) {
	_conn := (*C.queue_pair)(unsafe.Pointer(conn))
	// cbuf := C.CBytes(buf)
	_buf := unsafe.Pointer(&buf[0])
	_size := C.size_t(size)
	_ret := C.client_send_rpc(_conn, _buf, _size)
	// buf = C.GoBytes(cbuf, C.int(_size))
	// defer C.free(cbuf)
	ret = int32(_ret)
	return
}

// client_receive_buf
func (conn *QueuePair) ClientReceiveBuf(buf []byte, size int) (ret int) {
	_conn := (*C.queue_pair)(unsafe.Pointer(conn))
	_buf := unsafe.Pointer(&buf[0])
	_size := C.size_t(size)
	_ret := C.client_receive_buf(_conn, _buf, _size)
	ret = int(_ret)
	return
}

// client_close
func ClientClose(sourceAddr string, destinationAddr string) (ret int32) {
	_sourceAddr := C.CString(sourceAddr)
	defer C.free(unsafe.Pointer(_sourceAddr))
	_destinationAddr := C.CString(destinationAddr)
	defer C.free(unsafe.Pointer(_destinationAddr))
	_ret := C.client_close(_sourceAddr, _destinationAddr)
	ret = int32(_ret)
	return
}

// register_server
func RegisterServer(sourceAddr string) (ret *ServerContext) {
	_sourceAddr := C.CString(sourceAddr)
	defer C.free(unsafe.Pointer(_sourceAddr))
	_ret := C.register_server(_sourceAddr)
	ret = (*ServerContext)(unsafe.Pointer(_ret))
	return
}

// accept
func (handler *ServerContext) Accept() (ret *QueuePair) {
	_handler := (*C.server_context)(unsafe.Pointer(handler))
	_ret := C.accept(_handler)
	log.Info().Msgf("Server: open response : %v \n ", _ret)
	if _ret == nil {
		//null case, return simple type conversion
		ret = (*QueuePair)(unsafe.Pointer(_ret))
		log.Info().Msgf("Server: open response ret old : %v \n ", ret)
		return ret
	}
	// clientid := C.GoBytes(unsafe.Pointer(&_ret.client_id), C.sizeof_int)
	// request_shmaddr := C.GoBytes(unsafe.Pointer(&_ret.request_shmaddr), C.sizeof_char)
	// response_shmaddr := C.GoBytes(unsafe.Pointer(&_ret.response_shmaddr), C.sizeof_char)
	ret = &QueuePair{
		ClientId:        int(_ret.client_id),
		RequestShmaddr:  _ret.request_shmaddr,
		ResponseShmaddr: _ret.response_shmaddr,
	}
	log.Info().Msgf("Server: open response ret new : %v \n ", ret)

	// C.free((unsafe.Pointer(_ret)))

	return ret
}

// manage_pool
func (handler *ServerContext) ManagePool() {
	_handler := (*C.server_context)(unsafe.Pointer(handler))
	C.manage_pool(_handler)
}

// shutdown
func (handler *ServerContext) Shutdown() {
	_handler := (*C.server_context)(unsafe.Pointer(handler))
	C.shutdown(_handler)
}

// server_receive_buf
func (client *QueuePair) ServerReceiveBuf(buf []byte, size int) (ret int) {
	_client := (*C.queue_pair)(unsafe.Pointer(client))
	// cbuf := C.CBytes(buf)
	_buf := unsafe.Pointer(&buf[0])
	_size := C.size_t(size)
	_ret := C.server_receive_buf(_client, _buf, _size)
	// buf = C.GoBytes(cbuf, C.int(_size))
	// defer C.free(cbuf)
	ret = int(_ret)
	return
}

// server_send_rpc
func (client *QueuePair) ServerSendRpc(buf []byte, size int) (ret int32) {
	_client := (*C.queue_pair)(unsafe.Pointer(client))
	_buf := unsafe.Pointer(&buf[0])
	_size := C.size_t(size)
	_ret := C.server_send_rpc(_client, _buf, _size)
	ret = int32(_ret)
	return
}
