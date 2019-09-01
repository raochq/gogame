package session

import (
	"github.com/raochq/gogame/base/logger"
	"github.com/raochq/gogame/base/network"
	"net"
	"time"
)

type Session struct {
	conn *network.TCPConnection

	limitPerSecond int
	lastCount      int
	lastTime       time.Time

	verified    bool
	verifyTimer *time.Timer
	Userdata    interface{}
}

func newSession(conn *network.TCPConnection, verifyTime time.Duration) *Session {
	session := &Session{
		conn:     conn,
		verified: false,
	}
	session.verifyTimer = time.AfterFunc(verifyTime, session.verifyTimeout)
	return session
}

func (this *Session) SetReceiveLimitPerSecond(limit int) {
	this.limitPerSecond = limit
}

func (this *Session) receiveLimit() bool {
	if this.limitPerSecond == 0 {
		return false
	}
	this.lastCount++
	if this.lastCount > this.limitPerSecond {
		now := time.Now()
		if now.Sub(this.lastTime) < time.Second {
			return true
		}
		this.lastTime = now
		this.lastCount = 0
	}
	return false
}

func (this *Session) Verify() {
	if this.verified {
		return
	}
	if this.verifyTimer.Stop() {
		logger.Info("session: Verified %v", this.conn.RemoteAddr())
		this.verified = true
	}
}

func (this *Session) IsVerified() bool {
	return this.verified
}

func (this *Session) verifyTimeout() {
	logger.Info("session: Verify timeout, shutdown %v", this.conn.RemoteAddr())
	this.conn.Shutdown()
}

func (this *Session) Send(b []byte) {
	this.conn.Send(b)
}

func (this *Session) LocalAddr() net.Addr {
	return this.conn.LocalAddr()
}

func (this *Session) RemoteAddr() net.Addr {
	return this.conn.RemoteAddr()
}

func (this *Session) Shutdown() {
	if !this.verified {
		this.verifyTimer.Stop()
	}
	logger.Info("session: Shutdown %v", this.conn.RemoteAddr())
	this.conn.Shutdown()
}
