package treego

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

type tcp_net struct {
	// reference to TreeApi
	api *TreeApi

	// TCP connections, which are something like a channel
	// for entire API connection for TreeScale API
	connections []*net.TCPConn

	// locking connections for concurrent access
	connection_locker *sync.Mutex
	// keeping index for writing using round rubin algorithm
	connection_index int
}

func newTcpNet(api *TreeApi) *tcp_net {
	return &tcp_net{
		api:               api,
		connections:       make([]*net.TCPConn, 0),
		connection_locker: &sync.Mutex{},
		connection_index:  0,
	}
}

// Connecting to available endpoint in TreeApi
func (network *tcp_net) connect(channels int) {
	// Resolving address of endpoint
	addr, err := net.ResolveTCPAddr("tcp", network.api.endpoint)
	if err != nil {
		go network.api.trigger_local(EVENT_ON_ERROR,
			[]byte(fmt.Sprintf("Unable to resolve given endpoint %s from TCP -> %s", network.api.endpoint, err.Error())))
		return
	}

	// getting handshake from TreeApi
	handshake_data := network.api.firstHandshake()

	for i := 0; i < channels; i++ {
		conn, err := net.DialTCP("tcp", nil, addr)
		if err != nil {
			go network.api.trigger_local(EVENT_ON_ERROR,
				[]byte(fmt.Sprintf("Unable to connect to given endpoint %s from TCP -> %s", network.api.endpoint, err.Error())))
		} else {
			network.connection_locker.Lock()
			index := len(network.connections)
			network.connections = append(network.connections, conn)
			go network.handle_connection(index, conn, handshake_data)
			go network.api.trigger_local(EVENT_ON_CHANNEL_CONNECTION, []byte{})
			network.connection_locker.Unlock()
		}
	}
}

// Writing entire data to one of the existing connection channels
func (network *tcp_net) write(buffer []byte, conn *net.TCPConn) error {
	// if we don't have specific connection to write
	// just choosing from available channels using round rubin algorithm
	if conn == nil {
		network.connection_locker.Lock()
		if len(network.connections) == 0 {
			return errors.New("There is no available channels to write to!")
		}

		// getting one of the connections
		if network.connection_index >= len(network.connections) {
			network.connection_index = 0
		}
		conn = network.connections[network.connection_index]
		network.connection_index++
		network.connection_locker.Unlock()
	}

	write_offset := 0
	data_len := len(buffer)

	for {
		n, err := conn.Write(buffer[write_offset:])
		if err != nil {
			// if we have EOF, then we need to return from write functionality
			// connection would be closed from reader
			if err == io.EOF {
				return errors.New("Connection channel closed!")
			}

			// if we got temporary network error, just sleeping for a few milliseconds
			if nerr, ok := err.(net.Error); ok && nerr.Temporary() {
				time.Sleep(time.Millisecond * 10)
			}
		}

		if n+write_offset < data_len {
			write_offset += n
			continue
		}

		// if we got here, then all data have been written
		// so just breaking loop to return
		break
	}

	return nil
}

// handling connection for readable data
func (network *tcp_net) handle_connection(index int, conn *net.TCPConn, handshake []byte) {
	err := network.write(handshake, conn)
	if err != nil {
		// if we got error during handshake write
		// just closing connection
		network.close_connection(index)
		return
	}

	// keeping 4 bytes for reading first 4 bytes
	endian_data := make([]byte, 4)

	// after writing handshake, we need to read server handshake also
	// first reading API version as a BigEndian 4 bytes
	api_version, close_conn := read_endian(conn, endian_data)
	if close_conn {
		network.close_connection(index)
		return
	}

	// checking if we got valid API version
	if api_version > MAX_API_VERSION || api_version == 0 {
		go network.api.trigger_local(EVENT_ON_ERROR, []byte(fmt.Sprintf("Invalid API version %d from TCP channel", api_version)))
		network.close_connection(index)
		return
	}

	// reading token and value from endpoint
	token_value_buf, close_conn := read_message(conn, endian_data)
	if close_conn {
		network.close_connection(index)
		return
	}

	tv_len := len(token_value_buf)
	token := token_value_buf[:tv_len-8]
	value := binary.BigEndian.Uint64(token_value_buf[tv_len-8:])
	go network.api.trigger_local(EVENT_ON_CONNECTION, []byte(fmt.Sprintf("%s|%d", token, value)))

	var data []byte

	for {
		data, close_conn = read_message(conn, endian_data)
		if close_conn {
			network.close_connection(index)
			return
		}

		go network.api.handle_data(data)
	}
}

func (network *tcp_net) close_connection(index int) {
	network.connection_locker.Lock()

	if index >= 0 && index < len(network.connections) {
		network.connections[index].Close()
		network.connections = append(network.connections[index:], network.connections[index+1:]...)

		go network.api.trigger_local(EVENT_ON_CHANNEL_DISCONNECT,
			[]byte(fmt.Sprintf("TCP Channel closed. Left %d channels", len(network.connections))))
	}

	network.connection_locker.Unlock()
}

func read_endian(conn *net.TCPConn, endian_data []byte) (uint32, bool) {
	// sometimes, we will need indexing 4 bytes because it is possible
	// that connection is not sending full 4 bytes at once
	endian_index := 0
	for {
		n, err := conn.Read(endian_data[endian_index:])
		if err != nil {
			// if we have EOF, then we need to return from write functionality
			// connection would be closed from reader
			if err == io.EOF {
				return 0, true
			}

			// if we got temporary network error, just sleeping for a few milliseconds
			if nerr, ok := err.(net.Error); ok && nerr.Temporary() {
				time.Sleep(time.Millisecond * 10)
			}
		}

		if n+endian_index < 4 {
			endian_index += n
		}

		break
	}

	return binary.BigEndian.Uint32(endian_data), false
}

func read_message(conn *net.TCPConn, endian_data []byte) ([]byte, bool) {
	length, close_conn := read_endian(conn, endian_data)
	if close_conn || length > MAX_READ_DATA_LEN {
		return nil, true
	}

	buffer := make([]byte, length)
	offset := 0

	for {
		n, err := conn.Read(buffer[offset:])
		if err != nil {
			// if we have EOF, then we need to return from write functionality
			// connection would be closed from reader
			if err == io.EOF {
				return nil, true
			}

			// if we got temporary network error, just sleeping for a few milliseconds
			if nerr, ok := err.(net.Error); ok && nerr.Temporary() {
				time.Sleep(time.Millisecond * 10)
			}
		}

		if n+offset < int(length) {
			offset += n
		}

		break
	}

	return buffer, false
}
