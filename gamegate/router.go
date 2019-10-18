package main

import (
	"github.com/golang/protobuf/proto"
	"gogame/base/conf"
	"gogame/base/logger"
	"gogame/base/network"
	"gogame/base/network/session"
	"gogame/base/util"
	. "gogame/common"
	. "gogame/errcode"
	. "gogame/protocol"
	"gogame/protocol/pb"
	"math/rand"
	"sync"
	"time"
)

type Router struct {
	addr   string
	client *network.TCPClient
}
type RouterMgr struct {
	sync.RWMutex
	routerMgr      map[uint16]*Router
	crossRouterMgr map[uint16]*Router
	routerMap      map[string]bool
	svrList        map[uint16]int //gamesvr列表
	teamIds        []uint16       //teamsvr列表
}

func NewRouterMgr() *RouterMgr {
	return &RouterMgr{
		routerMgr:      make(map[uint16]*Router),
		crossRouterMgr: make(map[uint16]*Router),
		routerMap:      make(map[string]bool),
		svrList:        make(map[uint16]int),
	}
}

func NewRouter(addr string) *Router {
	router := new(Router)
	router.addr = addr
	return router
}

func (r *Router) Start() {
	r.client = network.NewTCPClient(r.addr)
	go r.client.DialAndServe(r, session.DefaultSessionCodec, 2*time.Second)
	go func() {
		for {
			ok := <-r.client.ConnectChan
			if ok == false {
				RemoveSvrs(r)
				return
			}
			r.register()
		}
	}()

}
func (r *Router) Connect(con *network.TCPConnection) {
	logger.Debug("router connection router %s", con.RemoteAddr())
}

func (r *Router) Disconnect(con *network.TCPConnection) {
	logger.Debug("router disconnect %s", con.RemoteAddr())
}

func (r *Router) Receive(con *network.TCPConnection, data []byte) {
	msg := UnpackSSMessage(data)
	if msg == nil {
		return
	}

	body := &SSMessageBody{}
	err := body.UnMarshal(msg.Body)
	if err != nil {
		logger.Error("router receive unmarshal body error")
		return
	}
	switch msg.Head.SSCmd {
	case SSCMD_MSG:
		r.dispatch(msg.Head, body)
	case SSCMD_SYNC:
		r.sync(body)
	case SSCMD_HEARTBEAT:
		logger.Debug("router receive heartbeat")
	default:
		logger.Error("router receive unknown cmd: %d", msg.Head.SSCmd)
	}
}

func (r *Router) dispatch(head *SSMessageHead, ssMsg *SSMessageBody) {
	transType := head.TransType
	csMsg := &CSMessage{}
	csMsg.Head.MessageID = ssMsg.MessageID
	csMsg.Head.PacketNo = ssMsg.PacketNo
	csMsg.Body = ssMsg.Body
	if transType == TransType_ByKey {
		client := ClientMgrGetMe().GetByID(ssMsg.DstAccountID)
		if client == nil {
			// 玩家离线时，消息还未发完
			logger.Debug("dispatch get session fail for account %d", ssMsg.DstAccountID)
			return
		}
		if head.SrcType == EntityType_TeamSvr {
			if head.SrcID != client.TSID && client.TSID != 0 {
				logger.Warn("wrong TSID accountID %d from %d real %d", client.AccountID, head.SrcID, client.TSID)
			}
		}
		bContinue, err := client.HookServerMessage(ssMsg)
		if err == nil && bContinue == true {
			client.SendCSMsg(csMsg)
		}
	} else if transType == TransType_Broadcast {
		logger.Debug("Broadcast message 0x%x from account %d", ssMsg.MessageID, ssMsg.SrcAccountID)
		ClientMgrGetMe().BroadcastMsg(csMsg)
	} else {
		logger.Warn("Portal get message from router with unknown trans_type %d message 0x%x", transType, ssMsg.MessageID)
	}
}

func (r *Router) sync(ssmsg *SSMessageBody) {
	syncMsg := &pb.MessageSyncServer{}
	err := proto.Unmarshal(ssmsg.Body, syncMsg)
	if err != nil {
		logger.Error("sync message err %v", err)
		return
	}
	if syncMsg.GetServerType() == EntityType_TeamSvr {
		AddCrossRouterServer(syncMsg.GetServerIDs(), r)
	} else {
		AddRouterServer(syncMsg.GetServerIDs(), r)
	}
}

// 向Router注册
func (router *Router) register() error {
	head := &SSMessageHead{
		SSCmd:   SSCMD_READY,
		SrcType: EntityType_Portal,
		SrcID:   uint16(conf.GameGateConf().ServerID),
	}
	msg := &SSMessage{Head: head, Body: nil}
	data := PackSSMessage(msg)
	if data == nil {
		logger.Error("Register to router %v fail", head)
		return EC_MarshalFail
	}
	for {
		ok := router.client.GetConnection().Send(data)
		if !ok {
			break
		}
		time.Sleep(time.Second)
	}
	logger.Info("register srctype %d srcid %d", head.SrcType, head.SrcID)
	return nil
}
func AddCrossRouterServer(svrIds []int32, router *Router) {
	// 每次都是全量同步
	svr := GetGateServerInstance().routerMgr
	var oldSvrIds []int32
	svr.Lock()

	for svrID, vRouter := range svr.crossRouterMgr {
		if vRouter == router {
			oldSvrIds = append(oldSvrIds, int32(svrID))
			delete(svr.crossRouterMgr, svrID)
		}
	}
	for _, id := range svrIds {
		svr.crossRouterMgr[uint16(id)] = router
	}
	var teamIds []uint16
	for teamId, _ := range svr.crossRouterMgr {
		teamIds = append(teamIds, teamId)
	}
	svr.teamIds = teamIds
	svr.Unlock()
	logger.Debug("update cross servers %v", svrIds)

}
func AddRouterServer(svrIds []int32, router *Router) {
	// 每次都是全量同步，先删除，插入
	svr := GetGateServerInstance().routerMgr
	var oldSvrIds []int32
	svr.Lock()
	for svrID, vRouter := range svr.routerMgr {
		if vRouter == router {
			oldSvrIds = append(oldSvrIds, int32(svrID))
			delete(svr.routerMgr, svrID)
		}
	}
	for _, id := range svrIds {
		svr.routerMgr[uint16(id)] = router
	}
	svr.Unlock()
	logger.Debug("old %v, new %v", oldSvrIds, svrIds)
	for _, id := range oldSvrIds {
		if !util.IsInt32InSlice(svrIds, id) {
			logger.Warn("server %d is closed", id)
			go ClientMgrGetMe().ServerClose(uint16(id))
		}
	}
}
func RemoveSvrs(router *Router) {
	svrMgr := GetGateServerInstance().routerMgr
	svrMgr.Lock()
	defer svrMgr.Unlock()

	for svrId, vRouter := range svrMgr.routerMgr {
		if vRouter == router {
			delete(svrMgr.routerMgr, svrId)
		}
	}
	var teamIds []uint16
	for svrId, vRouter := range svrMgr.crossRouterMgr {
		if vRouter == router {
			delete(svrMgr.crossRouterMgr, svrId)
			continue
		}
		teamIds = append(teamIds, svrId)
	}
	svrMgr.teamIds = teamIds
	delete(svrMgr.routerMap, router.addr)

}

func GameSvrAlive(serverId uint16) bool {
	svrMgr := GetGateServerInstance().routerMgr
	svrMgr.RLock()
	defer svrMgr.RUnlock()
	_, ok := svrMgr.svrList[serverId]
	return ok
}

func DispatchGameSvr() uint16 {
	svrMgr := GetGateServerInstance().routerMgr
	availSirs := make([]uint16, 0, len(svrMgr.svrList))
	svrMgr.RLock()
	for id, status := range svrMgr.svrList {
		if status == ServerStatusOpen {
			availSirs = append(availSirs, id)
		}
	}
	svrMgr.RUnlock()
	if len(availSirs) <= 0 {
		return 0
	}
	index := rand.Intn(len(availSirs))
	return availSirs[index]
}
