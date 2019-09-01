package main

import (
	"github.com/raochq/gogame/base/conf"
	"github.com/raochq/gogame/base/logger"
	"github.com/raochq/gogame/base/network"
	"github.com/raochq/gogame/base/network/session"
	"github.com/raochq/gogame/base/service"
	"strconv"
)

var cfg = conf.Cfg.GameGate

var (
	server *GateServer
)

type GateServer struct {
	tcpServer *network.TCPServer
	clientMgr *ClientManager
}

func NewServer() *GateServer {
	svr := &GateServer{}
	return svr
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
	return true
}

func (s *GateServer) ReportState() {

}
