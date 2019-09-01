package network

import (
	"errors"
	"log"
	"net"
	"sync"
	"time"
)

var (
	ErrClientClosed = errors.New("network: Client closed")
)

type TCPClient struct {
	addr  string
	retry bool

	mutex      sync.Mutex
	connection *TCPConnection
	closed     bool
}

func NewTCPClient(addr string) *TCPClient {
	client := &TCPClient{
		addr:  addr,
		retry: false,
	}
	return client
}

func (this *TCPClient) EnableRetry()  { this.retry = true }
func (this *TCPClient) DisableRetry() { this.retry = false }

func dialTCP(addr string) (*net.TCPConn, error) {
	raddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return nil, err
	}
	return net.DialTCP("tcp", nil, raddr)
}

func (this *TCPClient) DialAndServe(handler TCPHandler, codec Codec) error {
	if this.isClosed() {
		return ErrClientClosed
	}

	if handler == nil {
		handler = DefaultTCPHandler
	}
	if codec == nil {
		codec = DefaultCodec
	}

	var tempDelay time.Duration // how long to sleep on connect failure
	for {
		conn, err := dialTCP(this.addr)
		if err != nil {
			if this.isClosed() {
				return ErrClientClosed
			}
			if !this.retry {
				return err
			}

			if tempDelay == 0 {
				tempDelay = 1 * time.Second
			} else {
				tempDelay *= 2
			}
			if max := 1 * time.Minute; tempDelay > max {
				tempDelay = max
			}
			log.Printf("TCPClient: dial error: %v; retrying in %v", err, tempDelay)
			time.Sleep(tempDelay)
			continue
		}
		tempDelay = 0

		connection := newTCPConnection(conn)
		if err := this.newConnection(connection); err != nil {
			connection.close()
			return err
		}
		if err := this.serveConnection(connection, handler, codec); err != nil {
			return err
		}
	}
}

func (this *TCPClient) isClosed() bool {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	return this.closed
}

func (this *TCPClient) newConnection(connection *TCPConnection) error {
	this.mutex.Lock()
	defer this.mutex.Unlock()

	if this.closed {
		return ErrClientClosed
	}
	this.connection = connection
	return nil
}

func (this *TCPClient) serveConnection(connection *TCPConnection, handler TCPHandler, codec Codec) error {
	connection.serve(handler, codec)
	// remove connection
	this.mutex.Lock()
	defer this.mutex.Unlock()

	if this.closed {
		return ErrClientClosed
	}
	this.connection = nil
	return nil
}

func (this *TCPClient) GetConnection() *TCPConnection {
	this.mutex.Lock()
	defer this.mutex.Unlock()

	if this.closed {
		return nil
	}
	return this.connection
}

func (this *TCPClient) Close() {
	this.mutex.Lock()
	defer this.mutex.Unlock()

	if this.closed {
		return
	}
	this.closed = true
	if this.connection == nil {
		return
	}
	this.connection.close()
	this.connection = nil
}
