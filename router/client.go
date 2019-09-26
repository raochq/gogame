package main

import (
	"gogame/base/network/session"
	. "gogame/protocol"
)

type routeClient struct {
	session     *session.Session
	id          uint16
	svrType     int8
	sendCnt     int64
	lastSendCnt int64
}

func newRouterClient(session *session.Session, id uint16, svrType int8) *routeClient {
	server := &routeClient{
		session: session,
		id:      id,
		svrType: svrType,
	}
	return server
}

func (this *routeClient) SendMsg(msg *SSMessage) {
	data := PackSSMessage(msg)
	this.session.Send(data)
	this.sendCnt++
}

func (this *routeClient) Close() {
	this.session.Shutdown()
}
