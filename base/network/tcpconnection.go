package network

import (
	"bufio"
	"log"
	"net"
	"runtime"
	"sync"
)

type TCPHandler interface {
	Connect(*TCPConnection)
	Disconnect(*TCPConnection)
	Receive(*TCPConnection, []byte)
}

type TCPConnection struct {
	conn *net.TCPConn

	bufs   [][]byte
	mutex  sync.Mutex
	cond   *sync.Cond
	closed bool

	Owner interface{}
}

type defaultTCPHandler struct {
}

func (*defaultTCPHandler) Connect(*TCPConnection) {

}

func (*defaultTCPHandler) Disconnect(*TCPConnection) {

}

func (*defaultTCPHandler) Receive(*TCPConnection, []byte) {

}

var DefaultTCPHandler = &defaultTCPHandler{}

func newTCPConnection(conn *net.TCPConn) *TCPConnection {
	connection := &TCPConnection{
		conn: conn,
	}
	connection.cond = sync.NewCond(&connection.mutex)
	conn.SetNoDelay(true) // no delay

	return connection
}

func (this *TCPConnection) serve(handler TCPHandler, codec Codec) {
	defer func() {
		if err := recover(); err != nil {
			const size = 64 << 10
			buf := make([]byte, size)
			buf = buf[:runtime.Stack(buf, false)]
			log.Printf("network: panic serving %v: %v\n%s", this.RemoteAddr(), err, buf)
		}
		this.conn.Close()
	}()
	if handler == nil {
		handler = DefaultTCPHandler
	}
	//this.handler = handler

	this.startBackgroundWrite(codec)
	defer this.stopBackgroundWrite()

	handler.Connect(this)
	defer handler.Disconnect(this)

	// loop read
	r := bufio.NewReader(this.conn)
	for {
		b, err := codec.Read(r)
		if err != nil {
			return
		}
		handler.Receive(this, b)
	}
}

func (this *TCPConnection) startBackgroundWrite(codec Codec) {
	this.mutex.Lock()
	defer this.mutex.Unlock()

	if this.closed {
		return
	}
	go this.backgroundWrite(codec)
}

func (this *TCPConnection) backgroundWrite(codec Codec) {
	defer func() {
		if err := recover(); err != nil {
			const size = 64 << 10
			buf := make([]byte, size)
			buf = buf[:runtime.Stack(buf, false)]
			log.Printf("network: panic serving %v: %v\n%s", this.RemoteAddr(), err, buf)
		}
		this.conn.Close()
	}()

	// loop write
	w := bufio.NewWriter(this.conn)
	for closed := false; !closed; {
		var bufs [][]byte

		this.mutex.Lock()
		for !this.closed && len(this.bufs) == 0 {
			this.cond.Wait()
		}
		bufs, this.bufs = this.bufs, bufs // swap
		closed = this.closed
		this.mutex.Unlock()

		for _, b := range bufs {
			if err := codec.Write(w, b); err != nil {
				this.closeSend()
				return
			}
		}
		if err := w.Flush(); err != nil {
			this.closeSend()
			return
		}
	}
}

func (this *TCPConnection) stopBackgroundWrite() {
	this.mutex.Lock()
	defer this.mutex.Unlock()

	if this.closed {
		return
	}
	this.closed = true
	this.cond.Signal()
}

func (this *TCPConnection) closeSend() {
	this.mutex.Lock()
	defer this.mutex.Unlock()

	if this.closed {
		return
	}
	this.closed = true
}

func (this *TCPConnection) LocalAddr() net.Addr {
	return this.conn.LocalAddr()
}

func (this *TCPConnection) RemoteAddr() net.Addr {
	return this.conn.RemoteAddr()
}

func (this *TCPConnection) Send(b ...[]byte) bool {
	this.mutex.Lock()
	if this.closed {
		this.mutex.Unlock()
		return false
	}
	this.bufs = append(this.bufs, b...)
	this.mutex.Unlock()

	this.cond.Signal()
	return true
}

func (this *TCPConnection) close() {
	this.conn.SetLinger(0)
	this.conn.Close()
}

func (this *TCPConnection) Shutdown() {
	this.stopBackgroundWrite() // stop write
}
