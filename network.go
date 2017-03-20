package treego

import (
	"net"
	"sync"
)

const (
	ERROR_TCP_CONNECTION_WRITE = 1
	ERROR_TCP_CONNECTION_READ  = 2
)

// Main networking functionality for API
type Network struct {
	// TCP networking
	tcp_net map[string]*TcpNetwork

	// keeping pointer to API node for calling functions from network
	node *Api
}

// Keeping TCP networking separately, because we will have other networking types
type TcpNetwork struct {
	endpoint string
	// list of connection channels
	connections []*net.TCPConn
	// index for writing data using round rubin over connected channels
	conn_index uint
	// locking index because we will have real concurrency
	conn_locker *sync.Mutex

	// keeping pointer to main networking
	network *Network
}

func newNetwork(api *Api) Network {
	return Network{
		tcp_net: make(map[string]*TcpNetwork),
		node:    api,
	}
}

// Validating address, and checking which protocol to use during connection
// For now we have only TCP, but later we will use also UNIX sockets and UDP
// Connecting to given endpoint
func (network *Network) connect(endpoint string) error {
	// for now we have only TCP functionality
	// but later we will check other protocols as well

	if _, ok := network.tcp_net[endpoint]; !ok {
		network.tcp_net[endpoint] = &TcpNetwork{
			endpoint:    endpoint,
			connections: make(map[string]*net.TCPConn),
			conn_index:  0,
			conn_locker: &sync.Mutex{},
			network:     network,
		}
	}

	return network.tcp_net[endpoint].connect(endpoint)
}

// After having single connection with specific endpoint
// Using this function we can define how many concurrent transfer channels to have
func (network *Network) open_channels(endpoint string, count int) error {
	// if we have tcp networking with existing endpoint
	// just connecting with specific amount of connections passed as "count"
	// if we don't have it, adding new TCP endpoint
	if _, ok := network.tcp_net[endpoint]; !ok {
		network.tcp_net[endpoint] = &TcpNetwork{
			endpoint:    endpoint,
			connections: make(map[string]*net.TCPConn),
			conn_index:  0,
			conn_locker: &sync.Mutex{},
			network:     network,
		}
	}

	tcp := network.tcp_net[endpoint]
	for i := 0; i < count; i++ {
		err := tcp.connect(endpoint)
		// if we got error during connection just returning
		if err != nil {
			return err
		}
	}

	return nil
}

// Closing channels if we are not using them
// If passed "count == -1" then all channels would be closed including all TCP connection
func (network *Network) close_channels(endpoint string, count int) {
	if tcp, ok := network.tcp_net[endpoint]; ok {
		if count == -1 || count > len(tcp.connections) {
			count = len(tcp.connections)
		}

		tcp.conn_locker.Lock()
		for i := 0; i < count; i++ {
			// this will remove connection with specific index
			// and will close connection
			tcp.close_conn(i)
		}
		tcp.conn_locker.Unlock()

		// if we don't have connections
		// so probably we don't need this TCP networking anymore
		if len(tcp.connections) == 0 {
			// deleting tcp handler from networking
			delete(network.tcp_net, endpoint)
		}
	}
}

func (network *Network) emit(endpoint string, event *Event) bool {
	if tcp, ok := network.tcp_net[endpoint]; ok {
		// converting event to bytes and writing it to one of the TCP channels
		// doing this concurrently, because event converting process and write is sync operations
		go func(event *Event) {
			tcp.write(event.ToBytes())
		}(event)

		return true
	}

	return false
}

/** TCP Networking functionality */

// Connecting to given host
func (tcp_net *TcpNetwork) connect(address string) error {
	addr, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		return err
	}

	conn, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		return err
	}

	tcp_net.conn_locker.Lock()

	// starting read process for connection
	go tcp_net.handle_connection(conn, len(tcp_net.connections))
	// adding channel to existing connections list
	tcp_net.connections = append(tcp_net.connections, conn)

	tcp_net.conn_locker.Unlock()
	return nil
}

// Writing given byte array to connection by picking up channel
// based on basic Round Rubin algorithm
func (tcp_net *TcpNetwork) write(data []byte) {
	tcp_net.conn_locker.Lock()

	if tcp_net.conn_index >= len(tcp_net.connections) {
		tcp_net.conn_index = 0
	}
	i := tcp_net.conn_index
	tcp_net.conn_index += 1
	conn := tcp_net.connections[i]

	tcp_net.conn_locker.Unlock()

	data_len := data
	data_index := 0
	for {
		n, err := conn.Write(data[data_index:])
		if err != nil {
			tcp_net.network.node.onError(ERROR_TCP_CONNECTION_WRITE)
			break
		}

		if data_index >= data_len {
			break
		}

		data_index += n
	}
}

// Concurrently handling connection
func (tcp_net *TcpNetwork) handle_connection(conn *net.TCPConn, index int) {
	// Allocating read buffer as a 64K byte
	buffer := make([]byte, 65536)
	for {
		n, err := conn.Read(buffer)
		if err != nil {
			tcp_net.network.node.onError(ERROR_TCP_CONNECTION_READ, err)

			// if we got here then we have connection error
			// so we need to close connection
			tcp_net.conn_locker.Lock()
			tcp_net.close_conn(index)
			tcp_net.conn_locker.Unlock()
		}

		// got data which would be converted to Event Data
		// handling it concurrently
		go tcp_net.network.node.handle_data(buffer[:n], tcp_net.endpoint)
	}
}

func (tcp_net *TcpNetwork) close_conn(index int) {
	if index < 0 || index >= len(tcp_net.connections) {
		return
	}

	conn := tcp_net.connections[index]
	tcp_net.connections = tcp_net.connections[:index+copy(tcp_net.connections[index:], tcp_net.connections[index+1:])]
	conn.Close()
}
