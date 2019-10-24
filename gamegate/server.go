package main

import (
	"gogame/base/conf"
	"gogame/base/dao/redis"
	"gogame/base/logger"
	"gogame/base/network"
	"gogame/base/network/session"
	"gogame/base/service"
	"gogame/base/util"
	"math/rand"
	"strconv"
	"time"
)

var (
	server *GateServer
)

type GateServer struct {
	tcpServer *network.TCPServer
	isClosed  bool
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
	rand.Seed(time.Now().Unix())

	// Redis init
	redis.InitRedis()

	cfg := conf.GameGateConf()

	s.tcpServer = network.NewTCPServer("0.0.0.0:" + strconv.FormatUint(uint64(cfg.ListenPort), 10))
	ClientMgrGetMe()

	if RouterMgrGetMe().RouterInit() == false {
		logger.Error("gate init failed as router init failed")
	}
	IMMgrGetMe()

	s.RefreshServerList()
	util.TimeInterval(30*time.Second, s.RefreshServerList)
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
	s.isClosed = true
	s.tcpServer.Close()
	IMMgrGetMe().Close()
	ClientMgrGetMe().Close()

}
func (s *GateServer) ReportState() {

}

func (s *GateServer) RefreshServerList() {
	if s.isClosed {
		return
	}
	refreshGameSvrList()
	refreshIMSvrList()
	refreshRouterSvrList()
}
