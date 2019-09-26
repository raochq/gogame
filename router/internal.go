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

// 监听internal server连接
type InternalListener struct {
	*Server
	loop *eventloop.EventLoop

	tcpServer *network.TCPServer
}

func newInternalListener(server *Server) *InternalListener {
	listener := &InternalListener{
		Server: server,
		loop:   server.loop,
	}
	return listener
}

func (this *InternalListener) Start(port uint16) error {
	this.tcpServer = network.NewTCPServer("0.0.0.0:" + strconv.FormatUint(uint64(port), 10))
	go this.ListenAndServe()
	return nil
}

func (this *InternalListener) ListenAndServe() {
	defer util.PrintPanicStack()
	if err := this.tcpServer.ListenAndServe(session.NewSessionProxy(this), session.DefaultSessionCodec); err != nil {
		logger.Error("TCPServer: Internal ListenAndServe failed, err: %v", err)
	}
}

// 连接建立
func (this *InternalListener) Connect(session *session.Session) {
	// do nothing
	logger.Info("InternalServer connect")
}

// 连接断开
func (this *InternalListener) Disconnect(session *session.Session) {
	this.loop.RunInLoop(func() {
		this.handleClosed(session)
	})
}

// 收到消息
func (this *InternalListener) Receive(session *session.Session, b []byte) {
	msg := UnpackSSMessage(b)
	if msg == nil {
		return
	}
	this.loop.RunInLoop(func() {
		this.handleMessage(session, msg)
	})
}

func (this *InternalListener) handleClosed(session *session.Session) {
	if !session.IsVerified() {
		return
	}
	internalServer, ok := session.Userdata.(*routeClient)
	if !ok {
		return
	}
	session.Userdata = nil // unref userdata
	switch internalServer.svrType {
	case common.EntityType_GameSvr:
		this.RemoveGameClient(internalServer)

		msg, err := this.PackGameMessage()
		if err != nil {
			logger.Warn("marshal fail %v", err)
			return
		}
		this.BroadcastPortals(msg)
		logger.Info("Broadcast gamesvrs to all portals")

	case common.EntityType_TeamSvr:
		this.RemoveTeamClient(internalServer)

		msg, err := this.PackTeamMessage()
		if err != nil {
			logger.Warn("marshal fail %v", err)
			return
		}
		this.BroadcastPortals(msg)
		logger.Info("Broadcast teamsvrs to all portals")

	default:
		logger.Error("handleClosed unknown svrtype %d", internalServer.svrType)
	}
}

func (this *InternalListener) handleMessage(session *session.Session, msg *SSMessage) {
	cmd := msg.Head.SSCmd
	switch cmd {
	case common.SSCMD_READY:
		this.handleReady(session, msg)
	default:
		if session.IsVerified() {
			if internalServer, ok := session.Userdata.(*routeClient); ok {
				this.handleVerifyMsg(internalServer, msg)
			}
		} else {
			logger.Warn("InternalListener handleMessage %d fail", cmd)
		}
	}
}

func (this *InternalListener) handleVerifyMsg(internalServer *routeClient, msg *SSMessage) {
	cmd := msg.Head.SSCmd
	switch cmd {
	case common.SSCMD_HEARTBEAT:
		this.handleHeartBeat(internalServer, msg)
	case common.SSCMD_DISCONNECT:
		this.handleDisconnect(internalServer, msg)
	case common.SSCMD_MSG:
		this.handleMsg(internalServer, msg)
	default:
		logger.Warn("InternalListener handleMessage %d fail", cmd)
	}
}

func (this *InternalListener) handleReady(session *session.Session, msg *SSMessage) {
	srcType := msg.Head.SrcType
	svrId := msg.Head.SrcID

	switch srcType {
	case common.EntityType_GameSvr:
		gameServer := newRouterClient(session, svrId, common.EntityType_GameSvr)
		session.Verify()
		session.Userdata = gameServer
		this.AddGameClient(gameServer)

		msg, err := this.PackGameMessage()
		if err != nil {
			logger.Warn("marshal fail %v", err)
			return
		}
		this.BroadcastPortals(msg)
		logger.Info("Broadcast gamesvrs to all portals")

	case common.EntityType_TeamSvr:
		teamServer := newRouterClient(session, svrId, common.EntityType_TeamSvr)
		session.Verify()
		session.Userdata = teamServer
		this.AddTeamClient(teamServer)

		msg, err := this.PackTeamMessage()
		if err != nil {
			logger.Warn("marshal fail %v", err)
			return
		}
		this.BroadcastPortals(msg)
		logger.Info("Broadcast teamsvrs to all portals")

	default:
		logger.Error("handleReady unknown svrtype: %d", srcType)
	}
}

func (this *InternalListener) handleHeartBeat(client *routeClient, msg *SSMessage) {
	client.SendMsg(msg) // 直接回发
}

func (this *InternalListener) handleDisconnect(client *routeClient, msg *SSMessage) {
	client.Close() // 主动断开连接
}

func (this *InternalListener) handleMsg(client *routeClient, msg *SSMessage) {
	svrId := msg.Head.DstID
	transType := msg.Head.TransType

	switch transType {
	case common.TransType_ByKey:
		portalServer := this.GetPortalClient(svrId)
		if portalServer == nil {
			logger.Warn("handleMsg portal %d not register", svrId)
			return
		}
		portalServer.SendMsg(msg)

	case common.TransType_Broadcast:
		this.BroadcastPortals(msg)

	default:
		logger.Warn("handleMsg transtype %d fail", transType)
	}
}
