package main

import (
	"gogame/base/conf"
	"gogame/base/eventloop"
	"gogame/base/logger"
	"gogame/base/service"
	"gogame/base/util"
	"gogame/common"
	"gogame/protocol"
	"gogame/protocol/pb"
	"time"
)

var (
	server *Server
)

type Server struct {
	loop *eventloop.EventLoop // 事件循环，驱动所有事件逻辑

	portalListener   *PortalListener   // portal 监听
	internalListener *InternalListener // internal 监听

	portalClients map[uint16]*routeClient
	gameClients   map[uint16]*routeClient
	teamClients   map[uint16]*routeClient

	msgToPortalCnt     int64
	lastMsgToPortalCnt int64
}

func NewServer() *Server {
	svr := &Server{
		loop:          eventloop.NewEventLoop(),
		portalClients: make(map[uint16]*routeClient),
		gameClients:   make(map[uint16]*routeClient),
		teamClients:   make(map[uint16]*routeClient),
	}
	svr.portalListener = newPortalListener(svr)
	svr.internalListener = newInternalListener(svr)
	return svr
}
func GetServerInstance() *Server {
	if server == nil {
		server = NewServer()
	}
	return server
}

func (self *Server) Init() {
	logger.Info("=============== Server start ===============")
	cfg := conf.RouteConf()
	if err := self.internalListener.Start(cfg.ServerAddr); err != nil {
		logger.Fatal("Listen internal failed, err: %v", err)
	}
	// listen to portal
	if err := self.portalListener.Start(cfg.PortalAddr); err != nil {
		logger.Fatal("Listen portal failed, err: %v", err)
	}
	util.TimeInterval(2*time.Second, func() {
		self.RunInLoop(self.Statistics)
	})
	logger.Info("server init success")
}

func (self *Server) Destroy() {
	defer util.PrintPanicStack()
	logger.Info("=============== Server stop ===============\n")
}
func (self *Server) MainLoop(sig <-chan byte) {
	defer util.PrintPanicStack()
	go func() {
		for {
			ch := <-sig
			if ch == service.SIG_STOP {
				self.loop.Close()
				return
			}
		}
	}()
	self.loop.Loop()
}

// Reload 警告，是异步调用，如需同步数据，要在RunInLoop中执行
func (self *Server) Reload() {
	//self.RunInLoop(func() {})
	logger.Info("Reload csv files")
}

// ReportState 警告，是异步调用，如需同步数据，要在RunInLoop中执行
func (self *Server) ReportState() {
	//self.RunInLoop(func() {})
	logger.Info("ReportState")
}

func (self *Server) RunInLoop(functor func()) {
	self.loop.RunInLoop(functor)
}

func (self *Server) AddPortalClient(client *routeClient) {

	if existServer, ok := self.portalClients[client.id]; ok {
		existServer.Close()
		logger.Error("portal serid %d already exist", client.id)
	}
	self.portalClients[client.id] = client
	logger.Info("Game svrid %d connect", client.id)
}

func (self *Server) RemovePortalClient(client *routeClient) {
	if _, ok := self.portalClients[client.id]; !ok {
		return
	}
	delete(self.portalClients, client.id)
	logger.Info("Game svrid %d disconnect", client.id)
}
func (self *Server) GetPortalClient(id uint16) *routeClient {
	if gameServer, ok := self.portalClients[id]; ok {
		return gameServer
	}
	return nil
}

func (self *Server) AddGameClient(client *routeClient) {

	if existServer, ok := self.gameClients[client.id]; ok {
		existServer.Close()
		logger.Error("portal serid %d already exist", client.id)
	}
	self.gameClients[client.id] = client
	logger.Info("Game svrid %d connect", client.id)
}

func (self *Server) RemoveGameClient(client *routeClient) {
	if _, ok := self.gameClients[client.id]; !ok {
		return
	}
	delete(self.gameClients, client.id)
	logger.Info("Game svrid %d disconnect", client.id)
}
func (self *Server) GetGameClient(id uint16) *routeClient {
	if gameServer, ok := self.gameClients[id]; ok {
		return gameServer
	}
	return nil
}

func (self *Server) AddTeamClient(client *routeClient) {

	if existServer, ok := self.teamClients[client.id]; ok {
		existServer.Close()
		logger.Error("portal serid %d already exist", client.id)
	}
	self.teamClients[client.id] = client
	logger.Info("Game svrid %d connect", client.id)
}

func (self *Server) RemoveTeamClient(client *routeClient) {
	if _, ok := self.teamClients[client.id]; !ok {
		return
	}
	delete(self.teamClients, client.id)
	logger.Info("Game svrid %d disconnect", client.id)
}
func (self *Server) GetTeamClient(id uint16) *routeClient {
	if gameServer, ok := self.teamClients[id]; ok {
		return gameServer
	}
	return nil
}

func (this *Server) PackGameMessage() (*protocol.SSMessage, error) {
	var svrIds []int32
	for _, gameServer := range this.gameClients {
		svrIds = append(svrIds, int32(gameServer.id))
	}
	pbSyncServer := &pb.MessageSyncServer{
		ServerIDs:  svrIds,
		ServerType: common.EntityType_GameSvr,
	}
	pbBody, id, err := protocol.Marshal(pbSyncServer)
	if err != nil {
		return nil, err
	}
	ssMsgBody := &protocol.SSMessageBody{
		MessageID: id,
		Body:      pbBody,
	}
	body, err := ssMsgBody.Marshal()
	if err != nil {
		return nil, err
	}
	head := &protocol.SSMessageHead{
		SSCmd:     common.SSCMD_SYNC,
		TransType: common.TransType_ByKey,
		SrcType:   common.EntityType_Router,
		SrcID:     uint16(conf.RouteConf().ServerID),
	}
	ssMsg := &protocol.SSMessage{Head: head, Body: body}
	return ssMsg, nil
}

func (this *Server) PackTeamMessage() (*protocol.SSMessage, error) {
	var svrIds []int32
	for _, teamServer := range this.teamClients {
		svrIds = append(svrIds, int32(teamServer.id))
	}
	pbSyncServer := &pb.MessageSyncServer{
		ServerIDs:  svrIds,
		ServerType: common.EntityType_TeamSvr,
	}

	pbBody, id, err := protocol.Marshal(pbSyncServer)
	if err != nil {
		return nil, err
	}
	ssMsgBody := &protocol.SSMessageBody{
		MessageID: id,
		Body:      pbBody,
	}
	body, err := ssMsgBody.Marshal()
	if err != nil {
		return nil, err
	}
	head := &protocol.SSMessageHead{
		SSCmd:     common.SSCMD_SYNC,
		TransType: common.TransType_ByKey,
		SrcType:   common.EntityType_Router,
		SrcID:     uint16(conf.RouteConf().ServerID),
	}
	msg := &protocol.SSMessage{Head: head, Body: body}
	return msg, nil
}
func (this *Server) BroadcastPortals(msg *protocol.SSMessage) {
	for _, client := range this.portalClients {
		client.SendMsg(msg)
	}
}

func (this *Server) BroadcastGames(msg *protocol.SSMessage) {
	for _, client := range this.gameClients {
		client.SendMsg(msg)
	}
}

func (this *Server) BroadcastTeams(msg *protocol.SSMessage) {
	for _, client := range this.teamClients {
		client.SendMsg(msg)
	}
}

func (this *Server) Statistics() {
	logger.Info("Statistics portalSendCnt %d", this.msgToPortalCnt-this.lastMsgToPortalCnt)
	this.lastMsgToPortalCnt = this.msgToPortalCnt
	for _, portalServer := range this.portalClients {
		logger.Info("Statistics portal %d write cnt %d", portalServer.id, portalServer.sendCnt-portalServer.lastSendCnt)
		portalServer.lastSendCnt = portalServer.sendCnt
	}
}
