package main

import (
	"gogame/base/conf"
	"gogame/base/logger"
	"gogame/base/network"
	"gogame/base/network/session"
	"gogame/base/service"
	"strconv"
)

var cfg = conf.Cfg.GameGate

var (
	server *GateServer
)

type GateServer struct {
	tcpServer *network.TCPServer
	clientMgr *ClientManager
	routerMgr *RouterMgr
}

func NewServer() *GateServer {
	svr := &GateServer{}
	return svr
}
func GetGateServerInstance() *GateServer {
	if server == nil {
		server = NewServer()
	}
	return server
}
func (s *GateServer) Init() {
	port := uint64(cfg.ListenPort)
	s.tcpServer = network.NewTCPServer("0.0.0.0:" + strconv.FormatUint(port, 10))

	if s.RouterInit() == false {
		logger.Error("gate init failed as router init failed")
	}
	s.clientMgr = NewClientManager()
	s.clientMgr.Init()

}

func (s *GateServer) Destroy() {

}

func (s *GateServer) MainLoop(sig <-chan byte) {
	go func() {
		for {
			ch := <-sig
			if ch == service.SIG_STOP {
				s.Terminate()
				return
			}
		}
	}()
	s.tcpServer.ListenAndServe(s.clientMgr, session.DefaultSessionCodec)

}
func (s *GateServer) Terminate() {
	logger.Info("GateServer terminated")
	s.tcpServer.Close()
	s.clientMgr.Close()
}
func (s *GateServer) RouterInit() bool {
	s.routerMgr = NewRouterMgr()
	routerAddrs := conf.Cfg.GameGate.RouterAddrs
	if len(routerAddrs) <= 0 {
		logger.Error("Router init fail no routers")
		return false
	}
	for i := 0; i < len(routerAddrs); i++ {
		logger.Debug("RouterInit connect router %s", routerAddrs[i])
		s.routerMgr.routerMap[routerAddrs[i]] = true
		router := NewRouter(routerAddrs[i])
		router.Start()
	}

	return true
}

func (s *GateServer) ReportState() {

}
