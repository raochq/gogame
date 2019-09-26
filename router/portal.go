package main

import (
	"gogame/base/eventloop"
	"gogame/base/logger"
	"gogame/base/network"
	"gogame/base/network/session"
	"gogame/base/util"
	"gogame/common"
	. "gogame/protocol"
	"strconv"
)

// 监听portal server连接
type PortalListener struct {
	*Server
	loop *eventloop.EventLoop

	tcpServer *network.TCPServer
}

func newPortalListener(server *Server) *PortalListener {
	listener := &PortalListener{
		Server: server,
		loop:   server.loop,
	}
	return listener
}

func (this *PortalListener) Start(port uint16) error {

	this.tcpServer = network.NewTCPServer("0.0.0.0:" + strconv.FormatUint(uint64(port), 10))
	go this.ListenAndServe()
	return nil
}

func (this *PortalListener) ListenAndServe() {
	defer util.PrintPanicStack()
	if err := this.tcpServer.ListenAndServe(session.NewSessionProxy(this), session.DefaultSessionCodec); err != nil {
		logger.Error("TCPServer: Portal ListenAndServe failed, err: %v", err)
	}
}

// 连接建立
func (this *PortalListener) Connect(session *session.Session) {
	// do nothing
	logger.Info("PortalServer connect")
}

// 连接断开
func (this *PortalListener) Disconnect(session *session.Session) {
	this.loop.RunInLoop(func() {
		this.handleClosed(session)
	})
}

// 收到消息
func (this *PortalListener) Receive(session *session.Session, b []byte) {
	msg := UnpackSSMessage(b)
	if msg == nil {
		return
	}
	this.loop.RunInLoop(func() {
		this.handleMessage(session, msg)
	})
}

// 处理断开
func (this *PortalListener) handleClosed(session *session.Session) {
	if !session.IsVerified() {
		return
	}
	client, ok := session.Userdata.(*routeClient)
	if !ok {
		return
	}
	session.Userdata = nil
	this.RemovePortalClient(client)
}

func (this *PortalListener) handleMessage(session *session.Session, msg *SSMessage) {
	cmd := msg.Head.SSCmd
	logger.Info("PortalServer handleMessage %v", cmd)
	switch cmd {
	case common.SSCMD_READY:
		this.handleReady(session, msg)
	default:
		if session.IsVerified() {
			if portalServer, ok := session.Userdata.(*routeClient); ok {
				this.handleVerifyMsg(portalServer, msg)
			}
		} else {
			logger.Warn("PortalServer handleMessage %d fail", cmd)
		}
	}
	this.msgToPortalCnt++
}

func (this *PortalListener) handleVerifyMsg(portalServer *routeClient, msg *SSMessage) {
	cmd := msg.Head.SSCmd
	switch cmd {
	case common.SSCMD_HEARTBEAT:
		this.handleHeartBeat(portalServer, msg)
	case common.SSCMD_DISCONNECT:
		this.handleDisconnect(portalServer, msg)
	case common.SSCMD_MSG:
		this.handleMsg(portalServer, msg)
	default:
		logger.Warn("PortalServer handleMessage %d fail", cmd)
	}
}

func (this *PortalListener) handleReady(session *session.Session, msg *SSMessage) {
	svrId := msg.Head.SrcID

	client := newRouterClient(session, svrId, common.EntityType_Portal)
	session.Verify()
	session.Userdata = client
	this.AddPortalClient(client)

	//同步服务器信息
	if msg, err := this.PackGameMessage(); err == nil {
		client.SendMsg(msg)
		logger.Info("Notify portal sync gamesvrs to portal %d", client.id)
	} else {
		logger.Warn("marshal fail %v", err)
	}

	if msg, err := this.PackTeamMessage(); err == nil {
		client.SendMsg(msg)
		logger.Info("Notify portal sync teamsvrs to portal %d", client.id)
	} else {
		logger.Warn("marshal fail %v", err)
	}

}

func (this *PortalListener) handleHeartBeat(client *routeClient, msg *SSMessage) {
	client.SendMsg(msg) // 直接回发
}

func (this *PortalListener) handleDisconnect(client *routeClient, msg *SSMessage) {
	client.Close() // 主动断开连接
}

func (this *PortalListener) handleMsg(portalServer *routeClient, msg *SSMessage) {
	dstType := msg.Head.DstType
	svrId := msg.Head.DstID
	transType := msg.Head.TransType

	switch dstType {
	case common.EntityType_GameSvr:
		switch transType {
		case common.TransType_ByKey:
			gameServer := this.GetGameClient(svrId)
			if gameServer == nil {
				logger.Warn("handleMsg game %d not register", svrId)
				return
			}
			gameServer.SendMsg(msg)

		case common.TransType_Broadcast:
			this.BroadcastGames(msg)

		default:
			logger.Warn("handleMsg transtype %v fail", transType)
		}

	case common.EntityType_TeamSvr:
		switch transType {
		case common.TransType_ByKey:
			teamServer := this.GetTeamClient(svrId)
			if teamServer == nil {
				logger.Warn("handleMsg team %d not register", svrId)
				return
			}
			teamServer.SendMsg(msg)

		case common.TransType_Broadcast:
			this.BroadcastTeams(msg)

		default:
			logger.Warn("handleMsg transtype %v fail", transType)
		}
	default:
		logger.Warn("handleMsg unknown svrtype: %d", dstType)
	}
}
