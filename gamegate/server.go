package main

import (
	"gogame/base/conf"
	"gogame/base/dao/redis"
	"gogame/base/logger"
	"gogame/base/network"
	"gogame/base/network/session"
	"gogame/base/service"
	"strconv"
)

var (
	server *GateServer
)

type GateServer struct {
	tcpServer *network.TCPServer
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
	// Redis init
	redis.InitRedis()

	cfg := conf.GameGateConf()

	port := uint64(cfg.ListenPort)
	s.tcpServer = network.NewTCPServer("0.0.0.0:" + strconv.FormatUint(port, 10))

	if s.RouterInit() == false {
		logger.Error("gate init failed as router init failed")
	}
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
	s.tcpServer.ListenAndServe(ClientMgrGetMe(), session.DefaultSessionCodec)

}
func (s *GateServer) Terminate() {
	logger.Info("GateServer terminated")
	s.tcpServer.Close()
	ClientMgrGetMe().Close()
}
func (s *GateServer) RouterInit() bool {
	s.routerMgr = NewRouterMgr()
	cfg := conf.GameGateConf()
	routerAddrs := cfg.RouterAddrs
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
