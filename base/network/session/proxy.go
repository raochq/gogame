package session

import (
	"gogame/base/logger"
	"gogame/base/network"
	"time"
)

const kVerifyTime = time.Second * 30

type SessionHandler interface {
	Connect(*Session)
	Disconnect(*Session)
	Receive(*Session, []byte)
}

type SessionProxy struct {
	handler SessionHandler
}

func NewSessionProxy(handler SessionHandler) *SessionProxy {
	return &SessionProxy{handler}
}

func (this *SessionProxy) Connect(conn *network.TCPConnection) {
	session := newSession(conn, kVerifyTime)
	conn.Owner = session
	this.handler.Connect(session)
}

func (this *SessionProxy) Disconnect(conn *network.TCPConnection) {
	this.handler.Disconnect(conn.Owner.(*Session))
	conn.Owner = nil // unref userdata
}

func (this *SessionProxy) Receive(conn *network.TCPConnection, b []byte) {
	// this.handler.Receive(conn.Userdata.(*Session), b)
	session := conn.Owner.(*Session)
	if session.receiveLimit() {
		session.Shutdown()
		logger.Warn("session: receive limit and shutdown %v", conn.RemoteAddr())
		return
	}
	this.handler.Receive(session, b)
}
